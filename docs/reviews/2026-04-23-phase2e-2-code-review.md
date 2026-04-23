# Phase 2e-2 会议 MVP 代码审查报告

> 审查时间：2026-04-23
> 审查范围：backend/go-service/app/meeting/** + media-server/src/** + frontend/src/store/meeting.js / utils/mediasoup-client.js / pages/meeting/** / components/meeting/** / services/websocket.js + Task 15 / 三轮媒体回归补丁 + deploy/docker-compose.dev.yml 与 scripts/deploy-public.sh
> 审查人：code-reviewer 子代理（第二轮，重写版本）
> 审查 commit：790996f（feature/phase2e-2-meeting-mvp）

## 总体评价

Phase 2e-2 会议 MVP 的代码质量整体达到了"可交付 demo、内网试跑"的水平。设计文档 §6 规定的 12 个 REST 接口与 14 个 WS 事件均已实现，架构分层（controller → service → dao → model）总体清晰；`MeetingSignalService` 中的 `resourceTrackKey`、`memberStateKey` 采用 Redis Hash/Set 持久化信令状态，对 MVP 的"房间补推"语义覆盖完整；`HTTPMediaOrchestrator` 针对 Router 创建/关闭的幂等和重试处理（`sync.Map` 缓存 + 指数退避 `doCloseRequest`）值得作为未来同类中间件调用的范式；前端 `store/meeting.js` 三轮回归修复（精确匹配 `producerId`、入口 `_reset()`、REST 前置、in-flight Promise 锁）正确回应了竞态场景，思路正确无反复。

但本轮审查发现若干**必须在 Task 16 收尾前修复的 P0 问题**，集中在"信令层权限校验不足"和"房间创建失败降级策略错误"两类。其中 `OnProducerClose`、`OnConsumeStart` 均缺失资源归属校验，允许会议内任一成员关闭他人 producer 或把 consumer 挂到他人 recv transport 上，属于典型的横向越权；`CreateRoom` 在 mediasoup Router 创建失败时仅 `logs.Warn` 继续返回，造成后续信令全链路不可用但用户却看到"会议创建成功"的体验型阻断。此外架构分层上 `ListChatMessages` / `ListMyMeetings` 绕过 DAO 直查 `s.db`，与项目规范明确冲突。其他 P1/P2 主要围绕事务边界、后台 goroutine `context.Background()` 丢失 trace_id、以及前端几处已知的细小竞态。

**问题统计**：P0 × 4，P1 × 8，P2 × 8，Nit × 11。

## P0 阻塞性问题（必须在 Task 16 结束前修复）

### P0-1 `OnProducerClose` 未校验 producer 归属，任何参会人可关闭他人 producer

- **文件**：`backend/go-service/app/meeting/service/meeting_signal_service.go:604-625`
- **问题**：`OnProducerClose` 仅调用 `loadRoomAndParticipant` 校验发起者处于房间内，随后直接把 `payload.ProducerID` 透传给 `mediaOrchestrator.CloseProducer`。payload 中的 `producer_id` 完全由客户端决定，并且服务端**不查验**该 producer 是否由该 `userID` 所创建（`trackResource` 的 Redis Set 是现成的归属依据却未被使用）。
- **影响**：会议内任意成员只需发送 `{ room_code, producer_id: <他人的 producerID> }` 即可静默关闭其他人的摄像头/麦克风，同时广播 `meeting.member.producer.new { closed: true }` 造成 UI 误导为受害者自行关流。由于 `payload.ProducerID` 可以从本用户收到的 `meeting.member.producer.new` 事件里原样抄来，攻击门槛为 0。
- **建议修复**：
  ```go
  key := resourceTrackKey(payload.RoomCode, userID)
  isOwner, _ := s.redis.SIsMember(ctx, key, "producer:"+payload.ProducerID).Result()
  if !isOwner {
      return ErrProducerNotOwned
  }
  ```

### P0-2 `OnConsumeStart` 未校验 transport_id 归属，可把 Consumer 挂到他人 recv transport

- **文件**：`backend/go-service/app/meeting/service/meeting_signal_service.go:546-565`
- **问题**：同 P0-1 只确认用户在会议里，直接把客户端上报的 `transport_id` 透传给 `mediaOrchestrator.CreateConsumer`。media-server 虽检查 `direction === 'recv'`，但不做跨用户身份对比。
- **影响**：用户 A 可以把 consume 请求挂到用户 B 的 recv transport 上，B 意外收到 A 想要的流（信息污染 / 流量放大），A 自己则表现为黑画面/无声。
- **建议修复**：同 P0-1 模式，校验 `transport:<transportID>` ∈ 本 userID 的 Set；可选让 media-server 在 `createConsumer` 对 transport `entry.userId` 做二次断言。

### P0-3 `CreateRoom` 在 mediasoup Router 创建失败时吞错继续返回成功

- **文件**：`backend/go-service/app/meeting/service/meeting_service.go:274-284`
- **问题**：`mediaErr` 仅 `logs.Warn`，`routerID=""` 情况下仍返回 `nil`。DB 写入房间成功但 Router 实际不存在，后续 `transport.create` 必失败；同时残留的"死房间"会占用 `FindActiveByUser` 的"一人一会议"名额。
- **影响**：用户看到"会议创建成功"但所有人入会立刻失败；用户无法再创建新会议（因为旧房间还 active），必须等待清理任务或手工 DB 介入。
- **建议修复**：`mediaErr != nil` 走致命路径——补偿调用 `leaveRoom` + `MarkEnded(system_error)`，返回 500/503 让前端提示用户重试；或把 `roomDAO.Create` + `participantDAO.JoinRoom` + `CreateRouter` 组成 saga 并在最后一步失败时回滚。

### P0-4 会议密码明文拼入 `uni.navigateTo` URL

- **文件**：`frontend/src/pages/meeting/join.vue:101-108`、`frontend/src/pages/meeting/preview.vue:404-417`
- **问题**：`onNext` 拼接 `?mode=join&code=...&password=...` 作为路由 query 传递给 preview 页。uni-app H5 默认 Hash 路由不会把 query 发到服务器 referer，但**浏览器历史、浏览器扩展、开发者工具 Network 面板、地址栏截图**都会完整保留密码；与设计 §2.2.1 "邀请链接仅带 token 不带密码"的约定明显冲突；若未来切 History 路由立刻会以 referer 形式外泄。
- **建议修复**：密码改走 `meetingStore.draftJoinPayload` 临时内存态或 `uni.setStorage` 一次性传递，preview 页读取后立即清空；严禁密码出现在路由 query 上。

## P1 重要问题（应该修复）

### P1-1 主持人转让非原子：`participantDAO.TransferHost` 与 `roomDAO.UpdateHost` 分两事务

- **文件**：`backend/go-service/app/meeting/service/meeting_service.go:551-556` / `440-452`
- **问题**：`LeaveRoom` 中主持人离开触发自动转让与显式 `TransferHost` 接口均调用先 `participantDAO.TransferHost`（更新新旧参会人角色）再 `roomDAO.UpdateHost`（更新房间 `host_id`）。两步没有放在同一事务。
- **影响**：第一步成功、第二步失败时 `meeting_participants.role=host` 与 `meeting_rooms.host_id` 不一致，`assertIsHost`（以 `room.HostID` 为准）判断错误 → 所有主持人操作 403，直至手工修复。
- **建议修复**：在 `MeetingParticipantDAO.TransferHost` 内部同事务内完成 `UPDATE meeting_participants` + `UPDATE meeting_rooms.host_id = ?`，service 只调用一次。

### P1-2 `ListChatMessages` / `ListMyMeetings` 绕过 DAO 直查 GORM

- **文件**：`backend/go-service/app/meeting/service/meeting_service.go:869-904`、`622-668`
- **问题**：直接 `s.db.WithContext(ctx).Model(&model.MeetingChat{})` / `&model.MeetingRoom{}`，违反项目硬性分层约定（controller → service → dao → model）；`MeetingChatDAO`、`MeetingRoomDAO` 都已存在且提供了基础 CRUD，但未被调用。
- **影响**：不可维护，后续 schema 变更、索引优化、查询统计都散落在 service 里；单元测试难以 mock DAO。
- **建议修复**：`MeetingChatDAO.ListByRoomBefore(ctx, roomID, beforeID, limit)`；`MeetingRoomDAO.ListByIDsFiltered(ctx, ids, status, role, sinceAt, limit)`。service 调用 DAO 即可。

### P1-3 `ListMyMeetings` fetchSize 过度拉取

- **文件**：`backend/go-service/app/meeting/service/meeting_service.go:630-635`
- **问题**：`fetchSize := (limit+1)*2*3`。`limit=50` 时一次拉 306 条 participant，再根据 participant 反查房间。这种"两轮过滤 N+1"逻辑的 N 放大系数为 6。
- **影响**：`列表页每次请求都读几百行参会记录+几十次 room 查询。高并发时数据库压力陡升。
- **建议修复**：DAO 一次 JOIN `meeting_participants` + `meeting_rooms`（按 status/role/sinceAt 过滤），`LIMIT limit+1` 判 `has_more`。

### P1-4 `EndRoom` 未使用行级锁 + 非事务

- **文件**：`backend/go-service/app/meeting/service/meeting_service.go:474-518`
- **问题**：`ListActiveByRoom` 快照 → `MarkEnded` → `LeaveAllActive` 无 `SELECT ... FOR UPDATE`，且三步非同事务。与并发 `JoinRoom` 交错时：A 端调 EndRoom 拿到快照 {u1,u2}；B 端 JoinRoom 此时插入 u3；A 端 MarkEnded + 广播 `{u1,u2}`；u3 成为幽灵参会者（进了 active 房间但从未收到 `meeting.room.ended` 广播）。
- **影响**：幽灵成员需要等 `MeetingCleanupTask` 扫表（心跳超时）或被动断连才清理，期间前端状态错乱。
- **建议修复**：整段放入 `tx := s.db.Begin()`，首步 `SELECT * FROM meeting_rooms WHERE id=? FOR UPDATE`，广播在事务提交后做。

### P1-5 后台 goroutine 一律 `context.Background()` 丢失 trace_id

- **文件**（示例）：`meeting_service.go:390-396 / 446-451 / 460-465 / 558-563 / 607-611 / 853-861`、`meeting_signal_service.go:192 / 439 / 527-532 / 618-623`、`ws/handler.go:142`
- **问题**：所有 `go func` 都用 `context.Background()` 重起 ctx，丢掉原请求的 trace_id / user_id / deadline。
- **影响**：生产排障时日志无法根据 trace_id 关联"REST 接口 → 广播 → 资源清理"链路；同时无 deadline 的后台任务在 DB/Redis 卡住时会堆积。
- **建议修复**：引入 `logs.DetachContext(ctx)`（保留 trace_id / user_id 等 K/V 但不含 cancel 信号）+ `WithTimeout`：
  ```go
  bgCtx, cancel := context.WithTimeout(logs.DetachContext(ctx), 5*time.Second)
  defer cancel()
  go s.broadcastToActiveParticipants(bgCtx, ...)
  ```

### P1-6 `_broadcastSelfState` / `resumeConsumer` 前端吞错，无补偿

- **文件**：`frontend/src/store/meeting.js`（`_broadcastSelfState`）、`frontend/src/utils/mediasoup-client.js`（`resumeConsumer`）
- **问题**：`_broadcastSelfState` 走 `wsService.send`（非 ACK），失败只 `console.warn`，`member_state` Hash 不更新；`resumeConsumer` 服务端失败后 client 仍本地 `consumer.resume()`。
- **影响**：WS 抖动期间"成员面板图标灰色"回归问题会重现；远端仍 paused 时前端看到黑画面/无声但无感知。
- **建议修复**：`_broadcastSelfState` 改 `sendWithAck` + 2 次重试；`resumeConsumer` 失败后 `closeConsumer` + 重新 `consume` 重建链路。

### P1-7 `HandleHostGraceExpired` Redis TTL 自然过期盲区

- **文件**：`backend/go-service/app/meeting/service/meeting_lifecycle_service.go`
- **问题**：入参 `grace_until` 转 TTL 为 `grace_until - now + ttlBuffer`，同时写入 Redis；本地 `AfterFunc` 漂移或节点重启时依赖 `ScanExpired` 兜底。但如果 Redis 先到期删除了 key，`ScanExpired` 扫不到 → 本地 AfterFunc 到点后 `DEL` 返回 0 被跳过，主持人宽限转让丢失。
- **影响**：主持人断线后 90s 宽限期可能不触发自动转让，房间陷入"无主"直到 TTL 清理任务启动。
- **建议修复**：Lua 脚本原子 CAS `GET+DEL`；或引入 `grace_lock:<code>` NX key + 单独的到期哨兵。

### P1-8 REST `member.joined` 与 WS `pushExistingRoomState` 异步顺序不确定

- **文件**：`backend/go-service/app/meeting/service/meeting_signal_service.go:171-326`
- **问题**：`OnRoomJoin` 内 `go s.pushExistingRoomState(...)` 与 `meeting_service.JoinRoom` 内 `go s.broadcastToActiveParticipants(...)` 互相并行；`member.joined` 到达新入者时，REST 侧生成的"别人已有状态"与 WS 侧 `pushExistingMemberStates` 可能交织——极端情况下新人先收到 "未知 user_id 的 state.changed"，然后 REST 补 participant。
- **影响**：`_onMemberStateChanged` 的 placeholder 兜底尚能接住，但会造成头像/昵称短暂缺失；以及 `activeParticipants` 计数瞬时抖动。
- **建议修复**：`pushExistingRoomState` 在 `OnRoomJoin` 返回前同步完成（前端本来就等 `room.join` 的 ACK），消除并行窗口。

## P2 一般问题（建议修复）

- **P2-1** `cleanupUserResources`（`meeting_signal_service.go:631-671`）未处理 `transport:` 条目，依赖 Router 级联；用户短暂抖动重连场景会留下 1~数秒 orphan transport。
- **P2-2** `preview.vue.startPreview`（`preview.vue:188-223, 307-326`）快速切换摄像头竞态：多次调用叠加，后发起的 track 可能被先起的 `srcObject=` 覆盖。建议 promise lock + 切换防抖。
- **P2-3** `room.vue.onLoad`（`room.vue:568-574`）`redirectTo` 后缺 `return`，`onMounted` 仍会跑一遍即使跳转中；极短时间闪默认 UI。
- **P2-4** WS token 经 URL query 下发（`ws/handler.go:99` + `websocket.js:78`），反向代理/ingress 日志默认记录 query；建议迁移到 `Sec-WebSocket-Protocol` 或 cookie。
- **P2-5** `MeetingChatService` 并未独立文件，而是嵌在 `meeting_service.go:821-904`；与设计 §5.2 "分服务职责"偏差，可随 P1-2 拆分。
- **P2-6** `generateUniqueRoomCode`（`meeting_service.go`）无重试上限和监控，碰撞率极小但极端场景下可能死循环。
- **P2-7** `SendChatMessage`（`meeting_service.go:824-866`）缺服务端长度/频率限制，仅前端校验；滥用风险。
- **P2-8** 前端 `MEETING_ENDED_REASON_LABEL`（`constants/meeting.js:53-58`）与 `room.vue:584-588` optional chaining 的覆盖度复核：新增 `system_error` reason 时两处都要改。

## Minor / Nit

- `meeting_signal_service.go:640-652 / 236-241` `kind:id` 解析用 `strings.SplitN` / `IndexByte` 替代当前 `Split`。
- `meeting_service.go` 多处时间字段 `Format("2006-01-02 15:04:05")` 与 RFC3339 混用，统一 RFC3339 便于前端解析。
- `meeting_signal_service.go:73` `resourceTTL` 硬编码，应中央化到 `constants/meeting.go`。
- `ws/handler.go:50-52` `CheckOrigin` 固定返回 true 附 TODO，公网上线前必须收敛（按照白名单 origin）。
- `http_media_orchestrator.go` `doRequest` 超时 5s 对 CreateRouter 偏紧，生产建议 10s + 重试。
- `deploy/docker-compose.dev.yml:107-108` 200 个 UDP+TCP 端口范围全量暴露，生产需按实际租户规模收敛并同步 `MEDIASOUP_RTC_MAX_PORT`。
- `scripts/deploy-public.sh:86-91` `REDIS_PASSWORD=""` 放行提示与 `redis.conf requirepass` 联动校验缺失。
- `media-server/src/middlewares/internal-auth.ts:17-19` `isPrivatePath` 未兼容 `request.url` 含 query string，建议 `request.routerPath`。
- `media-server/src/services/consumer.service.ts:7-9 / 41` producer.appData 资格校验注释留给 Phase 2e-3，与 P0-2 联动。
- `frontend/src/utils/mediasoup-client.js` in-flight 锁 reject 分支未显式把 Promise 置空（reject 后仍指向已解决的 Promise，下次调用直接 throw）。
- `frontend/src/store/meeting.js` `_onMemberLeft` vs `_cleanupRemoteProducer` 清理粒度一致性复核（非回归，但代码走读时容易混淆）。

## 亮点

- **Redis Hash `member_state` + 异步补推**（`pushExistingMemberStates`）：对 SFU "后入者错过历史事件"的典型难题是教科书式解法，用最低成本的持久化解决了事件回放需求。
- **`HTTPMediaOrchestrator` `sync.Map` + `LoadOrStore`** 并发防护 + Close 半段 404 归一化幂等：是 "上游有状态、下游无状态"跨进程编排的范式。
- **前端 `ensureSendTransport/ensureRecvTransport` in-flight Promise 锁**：是对 mediasoup-client 延迟加载竞态的最小代价修复，思路干净。
- **`MeetingCleanupTask` 背景兜底 + local timer 双保险**：典型"分布式状态必须有兜底扫表"的成熟思路。
- **`leave-btn::after { content: none }`**：展现了团队对 uni-app H5 历史坑的深度理解，注释清晰（知其然且知其所以然）。
- **`_broadcastSelfState` + `pushExistingMemberStates` + `_onMemberStateChanged` 占位条目**三管齐下，把"成员面板图标灰色"问题彻底闭环，可作为未来类似"端到端同步"的参考。

## 审查外遗留建议

1. `OnConsumeStart` / `OnProducerClose` / `pushExistingRoomState` 单测覆盖（目前 0 单测，依赖 Playwright 端到端）。
2. `resolveUserDisplay` 的 N+1 查询批量化（当前每个 participant 逐个查 user）。
3. WS 多实例广播需 Redis Pub/Sub 中转（设计 §7.4 已列但未实现）。
4. mediasoup Worker 故障演练：真实 crash 一个 Worker 后 Router 迁移路径是否可用。
5. TURN ephemeral credentials（REST API），目前 `deploy/.env.public.example` 是 static credential。
6. 密码哈希 bcrypt cost 审计（user 表沿用历史，未重审）。
7. `NetworkBadge` 接入 `RTCPeerConnection.getStats` 真实质量数据（目前是占位值）。

## 附录：已核实问题一览表

| 级别 | 模块 | 文件 | 行号 | 问题概括 |
|---|---|---|---|---|
| P0 | go-service/signal | `meeting_signal_service.go` | 604-625 | `OnProducerClose` 未校验 producer 归属 |
| P0 | go-service/signal | `meeting_signal_service.go` | 546-565 | `OnConsumeStart` 未校验 transport_id 归属 |
| P0 | go-service/service | `meeting_service.go` | 274-284 | `CreateRoom` Router 失败吞错继续 |
| P0 | frontend/pages | `join.vue` + `preview.vue` | 101-108 / 404-417 | 会议密码明文拼 URL |
| P1 | go-service/service | `meeting_service.go` | 440-452 / 551-556 | 主持人转让非原子两事务 |
| P1 | go-service/service | `meeting_service.go` | 869-904 / 622-668 | `ListChatMessages/ListMyMeetings` 绕 DAO |
| P1 | go-service/service | `meeting_service.go` | 630-635 | `ListMyMeetings` fetchSize 过度拉取 |
| P1 | go-service/service | `meeting_service.go` | 474-518 | `EndRoom` 缺行锁存在幽灵参会者窗口 |
| P1 | 多处 | - | - | 后台 goroutine 用 `context.Background()` 丢 trace_id |
| P1 | frontend | `meeting.js` + `mediasoup-client.js` | - | `_broadcastSelfState/resumeConsumer` 吞错 |
| P1 | go-service/lifecycle | `meeting_lifecycle_service.go` | `HandleHostGraceExpired` | Redis TTL 自然过期盲区 |
| P1 | go-service/signal | `meeting_signal_service.go` | 171-326 | REST 与 WS 广播顺序不确定 |
| P2 | go-service/signal | `meeting_signal_service.go` | 631-671 | `cleanupUserResources` 未处理 transport |
| P2 | frontend | `preview.vue` | 188-223 / 307-326 | 快速切设备竞态 |
| P2 | frontend | `room.vue` | 568-574 | 重定向后缺 return |
| P2 | infra | `ws/handler.go` + `websocket.js` | - | WS token 放 URL query |
| P2 | go-service/service | `meeting_service.go` | 821-904 | MeetingChatService 嵌入主 service |
| P2 | go-service/service | `meeting_service.go` | `generateUniqueRoomCode` | 重试上限缺失 |
| P2 | go-service/service | `meeting_service.go` | 824-866 | SendChatMessage 无服务端限流 |
| P2 | frontend | `constants/meeting.js` | 53-58 | ENDED_REASON_LABEL 覆盖复核 |
| Nit | go-service/signal | - | - | `kind:id` 解析改用 `strings.IndexByte` |
| Nit | go-service/service | - | - | 广播时间格式不统一 |
| Nit | go-service/signal | - | 73 | resourceTTL 未中央化 |
| Nit | go-service/ws | `ws/handler.go` | 50-52 | CheckOrigin 固定 true |
| Nit | go-service/media | `http_media_orchestrator.go` | - | doRequest timeout 5s 偏紧 |
| Nit | deploy | `docker-compose.dev.yml` | 107-108 | 200 端口全暴露 |
| Nit | deploy | `scripts/deploy-public.sh` | 86-91 | REDIS_PASSWORD 放行提示 |
| Nit | media-server | `internal-auth.ts` | 17-19 | isPrivatePath 未兼容 query string |
| Nit | media-server | `consumer.service.ts` | 7-9 / 41 | producer.appData 资格校验 TODO |
| Nit | frontend/media | `mediasoup-client.js` | - | in-flight 锁 reject 分支未清空 |
| Nit | frontend/store | `meeting.js` | - | `_onMemberLeft` vs `_cleanupRemoteProducer` 清理粒度 |

## Task 16 修复追踪（2026-04-24 更新）

### 已修复（本次 Phase 2e-2 收尾完成）

| ID | 级别 | 提交 | 结论 |
|---|---|---|---|
| - | P0 × 4 | cdaa39d | Media 资源归属校验 / CreateRoom 补偿 / 会议密码迁 in-memory store，详见 commit message |
| - | P1 × 8 | ea2bf96 | 主持人转让加事务 + SELECT FOR UPDATE / ListMyMeetings 分页 / goroutine trace_id 透传 / broadcast ack 重试 / Redis TTL 分布式锁兜底 / REST+WS 顺序归一等，详见 commit message |
| - | 资源生命周期 | f5ae095 | MeetingLifecycleService.OnRoomEnded 统一撤销 grace/TTL 定时器；Redis `resourceTrackKey` / `memberStateKey` 显式 DEL；前端 Pinia `_reset` + `_pendingBroadcastTimers` 清理 |
| P2-1 | P2 | 本批次 | `cleanupUserResources` 新增 `transport` 分支 + media-server DELETE /transports/:id + Go 端 `CloseTransport` |
| P2-2 | P2 | 本批次 | `preview.vue` 加 `previewSeq` 序号 + 200ms 切换防抖，杜绝快速换摄像头竞态 |
| P2-3 | P2 | 本批次 | `room.vue` `onLoad` redirectTo 后显式 `return`，避免 onMounted 重复初始化 |
| P2-6 | P2 | 本批次 | `generateUniqueRoomCode` 失败日志 + 达到 retry 上限返回 `ErrRoomCodeConflict` |
| P2-7 | P2 | 本批次 | `SendChatMessage` 服务端长度 500 字符（utf8 rune）+ Redis INCR 滑动窗口 30 条/分钟 |
| P2-8 | P2 | 本批次 | `MEETING_ENDED_REASON_LABEL` 补齐 `kicked`；后端 `OnWSDisconnect` 用 `MeetingLeftReasonDisconnect` 常量 |
| Nit splitn | Nit | 本批次 | `meeting_signal_service.go` 的 `kind:id` 解析统一 `strings.SplitN` |
| Nit ttl | Nit | 本批次 | `resourceTTL` 中央化到 `constants.MeetingResourceTrackTTLSeconds` |
| Nit origin | Nit | 本批次 | `ws/handler.go` CheckOrigin 按 `server.ws_allowed_origins` + `server.mode` 收敛，同源放行，release 模式仅允许白名单 |
| Nit timeout | Nit | 本批次 | `http_media_orchestrator.go` 默认 `TimeoutMS` 5000→10000；CreateRouter 新增 `CreateRouterRetry`（默认 1 次，300ms 退避） |
| Nit redispass | Nit | 本批次 | `deploy-public.sh` 追加 REDIS_PASSWORD × redis.conf requirepass 联动校验；redis.conf 加 TODO 注释 |
| Nit routepath | Nit | 本批次 | `internal-auth.ts` 按 path（剔除 query/hash）匹配白名单，避免 `?` 混淆 |
| Nit inflight | Nit | 本批次 | `mediasoup-client.js` 走读确认 `finally` 已覆盖 resolve/reject 两路，加强注释 |
| Nit review | Nit | 本批次 | `_onMemberLeft` 整槽关闭 vs `_onProducerNew(closed=true)` 精确匹配 producerId，粒度正确，无需改动 |

### 推迟到独立阶段（登记存档）

| ID | 级别 | 理由 | 跟进计划 |
|---|---|---|---|
| P2-4 | P2 | WS token 从 URL query 迁到首帧鉴权需要同时改 `ws/handler.go` / 前端 `websocket.js` / 反代日志脱敏，改动面大 | Phase 2f 安全专项批次单独处理 |
| P2-5 | P2 | `MeetingChatService` 拆分属于架构重构 | Phase 2f 服务分层专项批次 |
| Nit ports | Nit | `docker-compose.dev.yml` 200 UDP/TCP 端口暴露收敛 | 正式公网部署清单（Phase 3 前） |
| Nit appdata | Nit | `consumer.service.ts` `producer.appData` 资格校验 | Phase 2e-3 流媒体合流时同步做 |
| Nit rfc3339 | Nit | 广播时间格式统一 RFC3339 | 跨模块低优先级，随 Phase 2f 协议梳理 |
