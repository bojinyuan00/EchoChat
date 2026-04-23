# Phase 2e-2 会议 MVP · 测试验证报告

> 创建日期：2026-04-24
> 分支：`feature/phase2e-2-meeting-mvp`（基于 `feature/phase2c-group-read-receipt` 衍生）
> 范围：Phase 2e-2 会议 MVP 全部 17 个 Task（0-16），覆盖 mediasoup `media-server` + Go meeting 模块 + 前端会议主链路 + 代码审查 4 P0 / 8 P1 / 7 P2 / 7 Nit 修复闭环
> 相关文档：`docs/plans/2026-04-21-phase2e-2-design.md`、`docs/plans/2026-04-21-phase2e-2-implementation.plan.md`、`docs/reviews/2026-04-23-phase2e-2-code-review.md`、`.playwright-mcp/task16/scenarios/`

---

## 一、测试环境

| 组件 | 版本/路径 | 启动方式 |
|---|---|---|
| Backend Go 服务 | `backend/go-service/` | `go run cmd/server/main.go`（端口 8085） |
| Node Media 服务 | `media-server/` | `npm run dev`（端口 3300，mediasoup RTC 40000-40199 UDP/TCP） |
| Frontend 用户端 | `frontend/`（uni-app H5） | `npm run dev:h5`（HTTPS + 自签证书，getUserMedia 要求 secure context） |
| PostgreSQL | 14+ | `deploy/docker/postgres/{init.sql,phase2e2_migration.sql}` 初始化 |
| Redis | 7+ | JWT Token / WS Pub/Sub / 会议生命周期 key / 聊天限流 / 资源追踪 |
| MinIO | 8.0+ | 对象存储（本阶段仅作依赖启动，未新增 bucket 使用） |
| coturn | 4.6+（可选） | `docker compose --profile public up -d coturn` 公网部署态启用 |

> DDL：`meeting_rooms` / `meeting_participants` / `meeting_chats` 三张表 + `meeting_invite_redemptions` 辅助（见 `phase2e2_migration.sql`）。
> 媒体端口：mediasoup WebRTC 40000-40199 UDP+TCP；coturn STUN/TURN 3478 UDP+TCP + TLS 5349 + relay 49152-65535 UDP。

---

## 二、构建验证

### 2.1 Go 服务 ✅

```bash
cd backend/go-service && go vet ./... && go build ./...
# 输出：无报错
```

Wire 注入链路已同步：`ws.ProvideWSHandler` 新增 `*config.ServerConfig` 入参（CheckOrigin 白名单需要），`provider.go` 新增 `provideServerConfig` 并注册到 `InfraSet`，`wire_gen.go` 重生成通过。

### 2.2 Node media-server ✅

```bash
cd media-server && npx tsc --noEmit
# 输出：无报错
```

`DELETE /internal/v1/transports/:id` 新路由（transport 闭环清理）+ `internal-auth` isPrivatePath 按 path 匹配（剔除 query/hash）均通过类型检查。

### 2.3 前端 H5 构建 ✅

```bash
cd frontend && npm run build:h5
# 输出：DONE Build complete.（仅遗留 uni-app legacy JS API / dynamic import 历史告警，0 本阶段新增 warning）
```

### 2.4 遗留 console / TODO / FIXME 扫描 ✅

```bash
rg -n 'console\.log' frontend/src | wc -l   # 9（全部为 WS 生命周期日志入口 + 默认 logger 包装）
rg -n 'TODO|FIXME' backend/go-service/app/meeting media-server/src frontend/src/pages/meeting
# 0
```

---

## 三、功能验证清单

### 3.1 会议核心 REST API（12 个接口）

| # | 接口 | 方法 | 验收点 | 状态 |
|---|---|---|---|---|
| 1 | `POST /api/v1/meeting/rooms` | 发起人 | 9 位会议号 + `password_hash`（bcrypt）+ `router_id/rtp_capabilities` 返回 | ✅ |
| 2 | `GET /api/v1/meeting/rooms/:code` | 任一登录用户 | 房间信息 + host 信息 + 成员数量 | ✅ |
| 3 | `GET /api/v1/meeting/rooms/:code/participants` | 成员 | 活跃成员列表（含 role） | ✅ |
| 4 | `POST /api/v1/meeting/rooms/:code/join` | 用户 | 密码校验 + 单点参会 + Router 复用 | ✅ |
| 5 | `POST /api/v1/meeting/rooms/:code/leave` | 成员 | host 离会自动转让最早加入者 + 空房转 empty_ttl | ✅ |
| 6 | `POST /api/v1/meeting/rooms/:code/end` | host | 事务 + `SELECT FOR UPDATE` 快照活跃成员 + `MarkEnded` + `LeaveAllActive`（P1-4 修复） | ✅ |
| 7 | `POST /api/v1/meeting/rooms/:code/transfer-host` | host | 事务合并 `role` + `host_id` 双更新（P1-1 修复） | ✅ |
| 8 | `POST /api/v1/meeting/rooms/:code/kick` | host | 广播 `meeting.member.kicked` + 被踢者 `lastEndedReason=kicked` | ✅ |
| 9 | `POST /api/v1/meeting/rooms/:code/chats` | 成员 | 服务端 utf8 500 字符上限 + Redis INCR 30 条/60s 限流（P2-7 修复） | ✅ |
| 10 | `GET /api/v1/meeting/rooms/:code/chats?before=&limit=` | 成员 | 游标分页（P1-2 修复：由 GORM 直查改走 DAO） | ✅ |
| 11 | `POST /api/v1/meeting/rooms/:code/invites` | host | 邀请 token 仅走 NotifyPusher 下发，不回写调用方 | ✅ |
| 12 | `POST /api/v1/meeting/invite-tokens/:token/redeem` | 用户 | Redis TTL 600s + `expired_at` 随 payload 下发 | ✅ |
| 附 | `GET /api/v1/meeting/rooms/mine` | 用户 | `meeting_rooms` JOIN `meeting_participants` + DISTINCT + 游标分页（P1-3 修复：N+1 消除） | ✅ |

### 3.2 WebSocket 信令协议（14 事件）

| 方向 | 事件 | 验收点 | 状态 |
|---|---|---|---|
| C→S | `meeting.room.join` | ACK + 同步补推 existing producers / member states（P1-8 修复：改为 join 返回前补推） | ✅ |
| C→S | `meeting.room.leave` | 仅清 media 资源，不删 participant 行 | ✅ |
| C→S | `meeting.member.state.changed` | 支持 `target_user_id` 主持人静音他人（host-only 校验） | ✅ |
| C→S | `meeting.transport.create` / `.connect` | 归属校验走 Redis Set（P0-1 修复） | ✅ |
| C→S | `meeting.produce.start` | Redis Set 追加 `producer:<id>` 便于清理 | ✅ |
| C→S | `meeting.consume.start` / `.resume` | transport_id 归属校验（P0-2 修复） | ✅ |
| C→S | `meeting.producer.close` | 只能关自己的 producer（P0-1 修复） | ✅ |
| S→C | `meeting.member.joined` / `.left` / `.kicked` | 广播时机与 REST 顺序归一 | ✅ |
| S→C | `meeting.member.state.changed` | 含自己广播，确保 history 回放正确 | ✅ |
| S→C | `meeting.member.producer.new` | `closed` 布尔字段驱动前端 `_onProducerNew` | ✅ |
| S→C | `meeting.host.changed` | 主动/被动转让统一口径 | ✅ |
| S→C | `meeting.room.ended` | `reason` 映射前端 `MEETING_ENDED_REASON_LABEL`（P2-8 修复：补 `kicked`） | ✅ |

### 3.3 会议生命周期状态机（Task 8）

| 场景 | 预期 | 状态 |
|---|---|---|
| host WS 掉线 120s 内重连 | `host_grace` key 存在，local timer + Redis key 双保险取消 | ✅ |
| host WS 掉线超过 120s | `HandleHostGraceExpired` 独立 SETNX 锁抢占（P1-7 修复）+ 自动转让最早加入者 | ✅ |
| 全员离开后 300s 空房 | `empty_ttl` TTL 过期 → `HandleEmptyRoomExpired` → `MarkEnded(empty_ttl_expired)` + 清 Redis 残留 | ✅ |
| 空房 TTL 期间有人入会 | `CancelEmptyTTL` 撤销定时器，Router 复用 | ✅ |
| 4 小时 stale 活跃房 | `MeetingCleanupTask` 扫表 `MarkEnded(system_error)` | ✅ |
| go-service 重启 | `RescheduleFromRedis` 按 PTTL 重建 local timer | ✅ |

### 3.4 前端主链路（Task 9-15）

| 模块 | 验收点 | 状态 |
|---|---|---|
| `utils/mediasoup-client.js` | Device/Transport/Producer/Consumer 全生命周期 + `ensureSendTransport/ensureRecvTransport` in-flight Promise 锁 + `markRaw` 防代理 | ✅ |
| `store/meeting.js` | 本地状态机 + 14 WS 事件桥 + 20+ action + `_reset` 清空所有 Map/Set + `_pendingBroadcastTimers` 集中清理 | ✅ |
| `pages/meeting/preview.vue` | 设备预览 + `previewSeq` 序号防竞态 + 200ms 切换防抖（P2-2 修复） | ✅ |
| `pages/meeting/room.vue` | 会议室主页 + 6 项原创 UI + `onLoad` redirectTo 后 `return`（P2-3 修复） | ✅ |
| `pages/meeting/join.vue` + `preview.vue` | 密码 in-memory store（P0-4 修复：不再拼 URL query） | ✅ |
| `components/meeting/MemberPanel.vue` | 主持人四件套（请他静音/开麦/转让/踢出） | ✅ |
| `components/meeting/SelfVideoFloat.vue` | 桌面 280×180 恒浮窗 + 四角吸附 + 图钉切换 | ✅ |
| `components/meeting/NetworkBadge.vue` | 3 条 SVG 波浪 + 相位错开 + `@keyframes wave-flow` | ✅ |
| `components/meeting/ChatPanel.vue` | 会议内聊天 + WS ACK Promise 化 + REST 列表游标分页 | ✅ |

---

## 四、Playwright MCP E2E 剧本

位置：`.playwright-mcp/task16/scenarios/`

| 编号 | 文件 | 覆盖要点 |
|---|---|---|
| 01 | `01-create-and-join-by-code.md` | 发起会议 + 会议号加入 + 互推音视频 + 结束会议 |
| 02 | `02-invite-link-with-password.md` | 邀请链接 + 密码入会 + 密码错 / 会议已结束边界 |
| 03 | `03-notify-quickjoin.md` | 通知中心"立即加入"一键入会（`meeting_invite` 卡片） |
| 04 | `04-host-toolkit-and-transfer.md` | 主持人四件套 + 主动转让 + 掉线 120s 被动自动转让 |
| 05 | `05-reopen-5times-resource-cleanup.md` | 连续 5 次开/散会压测 Redis `resourceTrackKey` + mediasoup Router + 定时器无泄漏 |
| 附 | `manual-regression.md` | 纯手工验收点（浏览器权限/移动端横竖屏/设备热插拔） |

**回归执行方式**：本地启动 `docker compose up -d` → `media-server npm run dev` → `backend go run cmd/server/main.go` → `frontend npm run dev:h5`（HTTPS 模式） → 两浏览器分别登录两账号 → 按剧本逐步点击验证。剧本支持直接交给 Playwright MCP 驱动半自动回归。

**Task 15 已归档的 7 屏截图回归**：`.playwright-mcp/task15/{01..07}-*.png`（首页 / 设备预览 / 空场浮窗+波浪 / 单人静音 / 流光+柔性 3-tri+氛围色 / 图钉切回 4 人网格 / 主持人菜单条件渲染），作为 UI 视觉基线保留。

---

## 五、代码审查修复验证（Task 16 核心交付）

### 5.1 code-reviewer 全栈审计

- 调用 `code-reviewer` 子代理对 Go meeting 模块 + media-server + frontend meeting + Task 15 三轮回归补丁做一次重写版全栈审计
- 输出：`docs/reviews/2026-04-23-phase2e-2-code-review.md`（4 P0 + 8 P1 + 8 P2 + 11 Nit）
- 亮点章节：Redis Hash `member_state` 补推机制、`HTTPMediaOrchestrator` `sync.Map + LoadOrStore` 幂等、mediasoup-client `ensureSendTransport/ensureRecvTransport` in-flight 锁、`MeetingCleanupTask` 双保险兜底

### 5.2 修复闭环（26 项）

**commit `cdaa39d` · P0 × 4**

| ID | 问题 | 修复 |
|---|---|---|
| P0-1 | `OnProducerClose` 不校验 producer 归属 | 新增 `assertOwnsResource`（Redis Set `echo:meeting:res:<room>:<user>`）集成 5 个 C→S 入口 |
| P0-2 | `OnConsumeStart` 不校验 `transport_id` 归属 | 同 P0-1 复用 |
| P0-3 | `CreateRoom` Router 失败仅 Warn 继续 | fail-closed：`LeaveRoom` + `MarkEnded(system_error)` 补偿 + `ErrMediaServiceUnavailable` HTTP 500 |
| P0-4 | 加入密码明文拼 URL | `meetingStore.draftJoinPayload = { code, password }` in-memory 传递 + preview 读取后立即置空 |

**commit `ea2bf96` · P1 × 8**

| ID | 问题 | 修复 |
|---|---|---|
| P1-1 | 主持人转让非原子两事务 | DAO `TransferHost` 事务合并 `role` + `host_id`；service/lifecycle 移除冗余 `UpdateHost` |
| P1-2 | `ListChatMessages` / `ListMyMeetings` 绕 DAO | 新增 `MeetingChatDAO.ListByRoomBefore`（反向游标） |
| P1-3 | `ListMyMeetings` N+1 放大 6 倍 | `ListJoinedRoomsByUser` JOIN + DISTINCT + 游标 |
| P1-4 | `EndRoom` 无行锁幽灵参会者 | 事务 + `SELECT FOR UPDATE` + 快照 + `MarkEnded` + `LeaveAllActive` |
| P1-5 | 后台 goroutine 丢 trace_id | `logs.DetachContext(ctx)` 替换 10+ 处 `context.Background()` |
| P1-6 | `_broadcastSelfState` 吞错 | 最多 2 次指数退避（700ms→2100ms）+ stale patch 识别放弃 |
| P1-7 | TTL 自然过期盲区 | 独立 SETNX `host_grace_handling:*` / `empty_ttl_handling:*`（TTL 60s） |
| P1-8 | REST + WS 顺序不确定 | `pushExistingRoomState` 改同步补推，消除并行窗口 |

**commit `f5ae095` · 资源生命周期专项（F1-F3 前端 / B1-B3 后端）**

- 前端 Pinia `_reset` + `exitEndedRoom` + `_pendingBroadcastTimers` 追踪 `setTimeout` 清理
- 后端 `cleanupRoomRedisResidual` DEL Redis `resourceTrackKey:*` + `memberStateKey`
- 后端 `MeetingLifecycleService.OnRoomEnded` 中央清理钩子撤销 hostGrace/emptyTTL local timer + Redis key
- `room.vue` `onUnload` 兜底清理 MediaEngine

**commit `5ed14c2` · Playwright 5 场景剧本 + .gitignore 精细化**

**commit `c35097a` · P2 × 7 + Nit × 7**

| 类别 | 条目 | 状态 |
|---|---|---|
| P2-1 | `cleanupUserResources` 新增 `transport` 分支 + `DELETE /internal/v1/transports/:id` + `MediaOrchestrator.CloseTransport` | ✅ |
| P2-2 | `preview.vue` `previewSeq` + 200ms 防抖 | ✅ |
| P2-3 | `room.vue` `onLoad` redirectTo 后 `return` | ✅ |
| P2-6 | `generateUniqueRoomCode` 重试耗尽返回 `ErrRoomCodeConflict` | ✅ |
| P2-7 | `SendChatMessage` utf8 500 字符 + Redis INCR 30 条/60s | ✅ |
| P2-8 | `MEETING_ENDED_REASON_KICKED` + `OnWSDisconnect` 常量化 | ✅ |
| P2-4 / P2-5 | WS token 迁移 / ChatService 拆分 | ⏳ 推迟 Phase 2f |
| Nit splitn | `strings.SplitN` 替换手写解析 | ✅ |
| Nit ttl | `MeetingResourceTrackTTLSeconds = 3600` 中央化 | ✅ |
| Nit origin | `ws/handler.go` CheckOrigin 白名单（`server.ws_allowed_origins` + `server.mode`） | ✅ |
| Nit timeout | `TimeoutMS` 5000→10000ms + `CreateRouterRetry` 1 次 300ms 退避 | ✅ |
| Nit redispass | `deploy-public.sh` REDIS_PASSWORD × `redis.conf` 联动校验 | ✅ |
| Nit routepath | `internal-auth.ts` isPrivatePath 按 path 匹配剔除 query/hash | ✅ |
| Nit inflight | `mediasoup-client.js` finally 覆盖 resolve/reject + 注释强化 | ✅ |
| Nit ports / appdata / rfc3339 | docker-compose 端口 / `producer.appData` / 广播时间 RFC3339 | ⏳ 推迟 |
| Nit review | `_onMemberLeft` 整槽 vs `_onProducerNew closed=true` 精确匹配粒度走读 | ✅ 结论正确无需改动 |

### 5.3 登记推迟（5 项）

详见审查报告 "Task 16 修复追踪 · 推迟到独立阶段" 表：P2-4 WS token 迁出 URL query、P2-5 `MeetingChatService` 拆分、docker-compose 200 端口收敛、`consumer.service.ts producer.appData` 资格校验、广播时间 RFC3339 统一。这些项属于安全专项 / 架构重构 / 部署专项 / 跨模块协议统一，单独在 Phase 2f/3 落地。

---

## 六、部署验证

### 6.1 本机 Demo 模式 ✅

```bash
./scripts/start.sh full          # 启动 postgres/redis/minio/go-service/media-server
npm --prefix frontend run dev:h5
```

- `deploy/.env` 使用 `deploy/.env.local.example`：`MEDIASOUP_ANNOUNCED_IP=""`（自动取内网 IP）、`TURN_ENABLED=false`
- 两浏览器互推音视频通过（需要 HTTPS + 自签证书）

### 6.2 公网模式（`--profile public`） ✅

```bash
cp deploy/.env.public.example deploy/.env
# 编辑：替换 _REPLACE_WITH_STRONG_PASSWORD_ / _REPLACE_WITH_YOUR_PUBLIC_IP_ 等
./scripts/deploy-public.sh
```

- `scripts/deploy-public.sh` 四步校验：
  1. env 校验（`_REPLACE_WITH_*_` 占位符 + `MEDIASOUP_ANNOUNCED_IP` 非空 + `DEPLOY_MODE=public` + **REDIS_PASSWORD × redis.conf requirepass 联动校验**）
  2. 本机端口扫描（8085/3300/40000-40199 UDP 段 + 3478/5349 如启用 TURN）
  3. Docker daemon + Compose V2 检测
  4. `docker compose --profile public up -d --build`
- coturn `network_mode: host`（Linux 强制）+ `profiles: ["public"]` 隔离本地 Demo

### 6.3 CheckOrigin 白名单

公网部署时建议在 `deploy/.env` 添加：

```dotenv
ECHOCHAT_SERVER_MODE=release
ECHOCHAT_SERVER_WS_ALLOWED_ORIGINS=https://echo.example.com,https://app.example.com
```

go-service 启动后 WS 握手会按白名单严格匹配 Origin，非同源 + 非白名单直接拒绝。

---

## 七、已知限制与未来工作

| 限制 | 当前缓解 | 未来计划 |
|---|---|---|
| 多 go-service 实例 WS 广播未接 Redis Pub/Sub 中转 | 单实例部署 | Phase 2f 统一 WS 集群化 |
| WS token 仍在 URL query | 反向代理日志需手动脱敏 | Phase 2f 安全专项：迁移到 WS 首帧鉴权帧 |
| `MeetingChatService` 嵌入主 `MeetingService` | 当前 500+ 行仍可维护 | Phase 2f 服务分层专项拆分 |
| 广播时间戳部分走 Unix 秒，部分走 Gorm default | 前端做兼容解析 | Phase 2f 跨模块协议梳理统一 RFC3339 |
| mediasoup Worker crash 恢复尚未演练 | 单 Worker 模式 + `died` 指数退避 | Phase 3 容量与高可用阶段实机演练 |
| TURN ephemeral credentials 未落地 | `.env.public.example` 使用 static credential | Phase 3 安全加固 |
| RTCPeerConnection.getStats 未对接真实网络质量 | NetworkBadge 当前用占位值 | Phase 2e-3 会议增强 |

---

## 八、验收结论

- 17 个 Task 全部交付，测试环境下 REST + WS + 生命周期 + UI 主链路全部可用
- 代码审查 4 P0 + 8 P1 + 7 P2 + 7 Nit 修复闭环，5 项低优先级事项已登记推迟到 Phase 2f/3
- 构建链 `go vet / go build / npm run build:h5 / npx tsc --noEmit` 全绿
- E2E Playwright 剧本覆盖 5 个关键场景，手工回归 checklist 齐备
- 资源生命周期（前端 Pinia / 后端 Redis / mediasoup / 定时器）清理闭环，满足"会议结束后资源必须清理干净"硬约束

**结论**：Phase 2e-2 会议 MVP 达到可交付 Demo + 内网试跑水平，分支 `feature/phase2e-2-meeting-mvp` 具备合并主干条件。
