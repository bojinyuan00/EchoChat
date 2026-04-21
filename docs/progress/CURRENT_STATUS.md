# EchoChat 项目开发进度

> **最后更新**：2026-04-21（Phase 2e-2 Task 7 HTTPMediaOrchestrator 落地，Go↔Node media-server HTTP 链路真实打通，端到端 16/16 PASS）
> **当前阶段**：Phase 2e-2 会议 MVP **代码开发阶段** 🚧（Task 0-7 ✅ / Task 8-16 待执行）
> **当前分支**：`feature/phase2e-2-meeting-mvp`（从 `feature/phase2c-group-read-receipt` 衍生）
> **Phase 2e 整体设计**：`docs/plans/2026-04-20-phase2e-design.md`（三子阶段路线图 + 后续规划清单）
> **Phase 2e-1 专用设计**：`docs/plans/2026-04-20-phase2e-1-design.md`（✅ 已完成）
> **Phase 2e-1 实施计划**：`docs/plans/2026-04-20-phase2e-1-implementation.plan.md`（✅ 11 个 Task 全部完成）
> **Phase 2e-1 验证报告**：`test-report-phase2e-1-notification.md`
> **Phase 2e-2 专用设计**：`docs/plans/2026-04-21-phase2e-2-design.md`（📋 设计阶段，16 章节）
> **Phase 2e-2 实施计划**：`docs/plans/2026-04-21-phase2e-2-implementation.plan.md`（📋 17 个 Task 共约 17 人日）

---

## 📋 2026-04-21 Phase 2e-2 设计阶段启动

**交付物**：两份新建文档 + 三份上游文档同步更新，已锁定 Phase 2e-2 会议 MVP 的全部架构决策与实施拆分。

### 新建文档

| 文档 | 规模 | 核心内容 |
|---|---|---|
| `docs/plans/2026-04-21-phase2e-2-design.md` | 16 章节 | 文档定位 / 范围边界 / 11 项关键决策 / 架构 mermaid 3 张 / 数据模型 DDL 3 张 / Go 后端 / Node media-server / 前端 / UI/UX / 通知对接 / 安全性能 / 风险 / 验收 / 衔接 / 关联 / 变更记录 |
| `docs/plans/2026-04-21-phase2e-2-implementation.plan.md` | 17 个 Task | Task 0 PoC → Task 16 E2E + 文档同步，共约 17 人日，含依赖拓扑 mermaid |

### 11 项关键决策（Plan 模式 5 轮澄清锁定）

| 编号 | 决策 | 选择 |
|---|---|---|
| D01 | 部署形态 | 本机 + 公网双态（环境变量切换） |
| D02 | 前端端形 | 仅 H5 浏览器 |
| D03 | Go-Node 协同 | Go 主控 + Node 无状态包装（权威状态在 Go） |
| D04 | 入会前设备预览 | MVP 纳入（独立预览页） |
| D05 | 加入体验 | 套餐 C：会议号 + 密码 + 邀请链接 + 通知中心 `meeting_invite` |
| D06 | 会议生命周期 | host 掉线 2 分钟宽限 + 自动转让（最早加入者）+ 空房 5 分钟 TTL |
| D07 | UI 风格 | 飞书简洁框架 + EchoChat 原创（流光轮廓 / 柔性网格 / 静音氛围色） |
| D08 | 主持人权限 | 四件套（静音他人 / 移除 / 转让 / 结束） |
| D09 | 媒体服务目录 | `media-server/` 根级子项目 |
| D10 | 移动端适配 | 桌面 + 手机双端响应式 |
| D11 | 会议内聊天 | MVP 纳入（独立 `meeting_chats` 表，24 小时后清理） |

### 数据模型修订（相对总设计）

- `meeting_rooms.password` → `password_hash VARCHAR(255)`（bcrypt 哈希替换明文）
- `meeting_rooms` 新增 `ended_reason`（结束原因：host_ended/empty_ttl/admin_force/system_error）
- `meeting_participants` 新增 `left_reason`
- 新表 `meeting_chats`（独立于 `im_messages`，24 小时 TTL）
- 新增 Redis key：`echo:meeting:invite:{token}`（邀请链接）、`echo:meeting:host_grace:{code}`（主持人宽限期）

### 上游文档同步

- `docs/plans/2026-04-20-phase2e-design.md` §四 精简为引用（指向 2e-2 专用设计 + 实施计划），§五 从 2e-3 范围里移除「会议邀请」（已上移 2e-2）
- `.cursor/rules/project-context.mdc` 当前进度追加 Phase 2e-2 设计阶段条目
- 本文件（CURRENT_STATUS.md）更新头部 + 新增本段

### 下一步

- **评审设计文档**：由用户 Review 两份新建文档
- **进入代码开发**：评审通过后从 Task 0（mediasoup PoC Spike）启动，预计 17 人日完成 MVP

---

## 🚀 2026-04-21 Phase 2e-2 Task 3 Go meeting 模块数据库 DDL + Model + DAO 完成

**交付**：Phase 2e-2 会议 MVP 的三张持久化表（`meeting_rooms` / `meeting_participants` / `meeting_chats`）完整落地到 PostgreSQL，配套 Go 侧 `app/meeting/{model,dao}` + 统一常量 `app/constants/meeting.go`；DDL 同时写入 `init.sql`（全量初始化）与 `phase2e2_migration.sql`（增量升级），在真实 postgres 容器跑通 CRUD + UNIQUE 约束 + CASCADE 级联删除 + 主持人转让事务。

### 产出文件

| 文件 | 行数 | 作用 |
|---|---|---|
| `deploy/docker/postgres/init.sql`（追加） | +119 | 3 张表 DDL + 9 个索引 + COMMENT 全量文档 |
| `deploy/docker/postgres/phase2e2_migration.sql` | 90 | 增量升级脚本（`IF NOT EXISTS` 幂等），用于已运行环境无损追加 |
| `backend/go-service/app/constants/meeting.go` | 110 | 8 组常量：会议类型/状态/角色/结束原因/离会原因/默认配置/WS 事件（与 group/notify 同构） |
| `backend/go-service/app/meeting/model/meeting_room.go` | 30 | `MeetingRoom` 结构体 + GORM 复合索引 tag + `TableName()` |
| `backend/go-service/app/meeting/model/meeting_participant.go` | 27 | `MeetingParticipant` 结构体 + `IsActive()` 辅助 + 联合唯一索引 tag |
| `backend/go-service/app/meeting/model/meeting_chat.go` | 18 | `MeetingChat` 结构体，纯文本 content + 房间聚合索引 |
| `backend/go-service/app/meeting/dao/meeting_room_dao.go` | 170 | 9 个方法：`Create/GetByID/GetByCode/ExistsCode/MarkStarted/MarkEnded/UpdateHost/UpdateSettings/ListByHost/ListExpiredForCleanup` |
| `backend/go-service/app/meeting/dao/meeting_participant_dao.go` | 235 | 11 个方法：`JoinRoom`（含重入复用）、`LeaveRoom`、`LeaveAllActive`、`TransferHost`（事务）、`FindActiveByUser`（JOIN 校验单点参会）、各类列表/计数/角色更新 |
| `backend/go-service/app/meeting/dao/meeting_chat_dao.go` | 85 | 4 个方法：`Create/ListByRoom`（游标分页）/`DeleteByRoomIDs`（清理任务）/`CountByRoom` |

### 关键设计决策

1. **常量目录对齐项目风格（偏离实施计划草案）**：实施计划草案写的是 `app/meeting/constants/{meeting_status,meeting_role}.go`，但项目现有风格是"模块级常量统一放在 `app/constants/<module>.go` 单文件"（见 `app/constants/group.go` / `notify.go`）。按 `project-context.mdc` 第 11 条「代码风格全局一致（最高优先级）」，本次采用 `app/constants/meeting.go` 单文件承载所有会议常量，同步修订实施计划。
2. **时间字段统一 TIMESTAMP(0)**：设计文档草案用了 `TIMESTAMPTZ`，但项目所有表（`auth_users` / `im_messages` / `notify_notifications`）统一使用 `TIMESTAMP(0)`（见 init.sql），Go model 搭配 `gorm:"type:timestamp(0)"`。本次 DDL 改为 `TIMESTAMP(0)` 保持一致。
3. **冗余索引移除**：设计文档草案写了 `idx_meeting_rooms_code`，但 `room_code UNIQUE NOT NULL` 已经自动建 B-tree 索引，冗余索引已移除避免双倍维护成本。
4. **重入复用单条参与者记录**：`JoinRoom` 使用事务，若 (room_id, user_id) 已存在且 `left_at IS NOT NULL` → UPDATE 复用该行（`joined_at=NOW, left_at=NULL, duration=0`）；仍活跃则返回 `ErrAlreadyInMeeting` 供上层转 409。避免每次重入写新记录污染审计数据。
5. **`duration` 使用 SQL 表达式计算**：`LeaveRoom` 用 `EXTRACT(EPOCH FROM (? - joined_at))::INT` 走数据库时间而非 Go 端 `time.Now()`，避免跨时区/NTP 漂移导致负 duration。
6. **`MarkEnded` 乐观锁**：仅对 `status != ended` 的行 UPDATE，重复结束只保留首次原因，不被覆盖。
7. **无 Go 单元测试（遵循项目现有风格）**：项目 Go 侧 0 个 `_test.go`，统一用"代码审查 + 真实 postgres psql 验证 + Playwright E2E"三层守护。本次 Task 3 验收用 psql 脚本跑通 8 类场景（创建、UNIQUE 约束 ×2、主持人转让事务、聊天写入、`duration` 精确匹配、CASCADE 清零），全部通过。

### 验证记录

- `go build ./...` ✅ 零报错
- `go vet ./...` ✅ 零报错
- `ReadLints app/meeting/ app/constants/meeting.go` ✅ 零 Lint 问题
- `docker exec echochat-postgres psql ... < phase2e2_migration.sql` ✅ 全部 `CREATE TABLE/INDEX/COMMENT` 成功
- `psql -c "\d meeting_*"` ✅ 3 张表结构、9 个索引、所有外键约束（含 `ON DELETE CASCADE`）正确生成
- psql 集成测试 ✅ 场景汇总：
  - `INSERT meeting_rooms` + 重复插 `room_code` → `unique_violation` 触发
  - `INSERT meeting_participants` + 重复 `(room_id,user_id)` → `unique_violation` 触发
  - `UPDATE role=0 WHERE role=1` / `UPDATE role=1 WHERE left_at IS NULL` 事务链 → 主持人转让成功
  - `INSERT meeting_chats ×2` → 2 行写入
  - `UPDATE left_at = NOW()+10s` + `duration = EXTRACT(EPOCH ...)` → `duration=10` 精确匹配
  - `DELETE meeting_rooms` → `participants` 残留 0 / `chats` 残留 0（CASCADE 生效）

### 下一步

- **Task 4**（0.5 人日）：Go 侧 `meeting` 模块的 service / controller / router 骨架，完成依赖注入 + 空实现占位，建立 `POST /api/meeting/create` 等路由的握手层。

---

## 🚀 2026-04-21 Phase 2e-2 Task 4 Go meeting 模块骨架（service / controller / router / wire）完成

**交付**：`app/meeting/` 模块 service 层 17 个空方法 + controller 层 12 个 Gin 处理器 + `/api/v1/meeting/*` 路由全局挂载 + Wire 依赖注入全局打通；附带修复 admin 模块 `MessageManageService/Controller` provider 缺失的存量问题；`go build ./...` / `go vet ./...` / `wire ./app/provider` 全绿；实机启动 server 确认 12 条路由全部注册并通过 JWT 鉴权（未授权返回 401 `缺少认证信息`）。

### 产出文件

| 文件 | 行数 | 作用 |
|---|---|---|
| `backend/go-service/app/meeting/service/interfaces.go` | 25 | 外部依赖接口抽象：`NotifyPusher` / `UserInfoResolver` / `OnlineChecker`，为后续 Task 5-15 解耦 notify/contact/ws 模块 |
| `backend/go-service/app/meeting/service/meeting_service.go` | 165 | `MeetingService` 结构体 + 8 个 sentinel error（`ErrMeetingNotFound` 等）+ 17 个空方法占位（全部返回 `ErrNotImplemented`），为 Task 5-10 业务逻辑预留挂载点 |
| `backend/go-service/app/meeting/controller/meeting_controller.go` | 150 | `MeetingController` + `responseNotImplemented`（返回 501）+ `requireUserID` 辅助 + 12 个 Gin 处理器，全部返回 501 占位 |
| `backend/go-service/app/meeting/router.go` | 35 | `RegisterRoutes()` 将 12 条路由按设计文档挂到 `/api/v1/meeting/*`，统一套用 `jwtAuth` 中间件 |
| `backend/go-service/app/meeting/provider.go` | 22 | `MeetingSet = wire.NewSet(DAO×3, Service, Controller)`，与其他模块 `Set` 命名一致 |
| `backend/go-service/app/provider/wire.go`（改） | +10 | 挂入 `meetingApp.MeetingSet` + 3 条 `wire.Bind`：`NotifyPusher→NotifyService`、`UserInfoResolver→FriendshipDAO`、`OnlineChecker→ws.OnlineService` |
| `backend/go-service/app/provider/provider.go`（改） | +6 | `App` struct 新增 `MeetingService` / `MeetingController` 字段 + `NewApp` 形参 |
| `backend/go-service/app/provider/wire_gen.go`（自动生成） | +30 | `wire` 命令自动重生成，按拓扑序串联 meeting 模块依赖 |
| `backend/go-service/router/router.go`（改） | +3 | `meetingApp.RegisterRoutes(engine, app.MeetingController, jwtAuth)` 挂载 |
| `backend/go-service/app/admin/provider.go`（改） | +6 | **存量修复**：补齐 `MessageManageDAO/Service/Controller` 至 `AdminSet`，修复旧版 wire 未能发现 provider 的 bug |

### 路由清单（12 条全部验证）

| 方法 | 路径 | 处理器 | 当前行为 |
|---|---|---|---|
| POST | `/api/v1/meeting/rooms` | `CreateRoom` | 501 NotImplemented |
| GET | `/api/v1/meeting/rooms` | `ListMyMeetings` | 501 |
| GET | `/api/v1/meeting/rooms/:code` | `GetRoom` | 501 |
| POST | `/api/v1/meeting/rooms/:code/join` | `JoinRoom` | 501 |
| POST | `/api/v1/meeting/rooms/:code/leave` | `LeaveRoom` | 501 |
| POST | `/api/v1/meeting/rooms/:code/end` | `EndRoom` | 501 |
| POST | `/api/v1/meeting/rooms/:code/transfer-host` | `TransferHost` | 501 |
| POST | `/api/v1/meeting/rooms/:code/kick` | `KickMember` | 501 |
| POST | `/api/v1/meeting/rooms/:code/invite` | `InviteUsers` | 501 |
| POST | `/api/v1/meeting/invites/:token/redeem` | `RedeemInvite` | 501 |
| POST | `/api/v1/meeting/rooms/:code/chats` | `SendChat` | 501 |
| GET | `/api/v1/meeting/rooms/:code/chats` | `ListChats` | 501 |

### 关键设计决策

- **接口隔离（`interfaces.go`）**：对 notify/contact/ws 只依赖接口而非具体类型，避免后续实现时出现循环依赖；`OnlineChecker.IsOnline` 签名与现存 `ws.OnlineService` 一致（返回单个 `bool`，内部吞噬 error），保持最小改动面。
- **骨架返回 501（而非 404/200）**：`responseNotImplemented` 统一返回 501 + `ErrNotImplemented` 消息，前端联调/Postman 验证时能明确区分"未实现"与"路由缺失"；与 group/contact 模块骨架风格保持一致。
- **存量问题一并修复**：`admin/provider.go` 漏注册 `MessageManage{DAO,Service,Controller}` 是一个跟 Task 4 无关的 wire 老 bug，本轮顺手修掉，使 `wire ./app/provider` 重生成不再报错；已在 commit 描述中注明。
- **Wire Bind 方向**：`wire.Bind(new(Interface), new(*ConcreteType))` 遵循"接口依赖指向具体类型"的惯例，与 Phase 2e-1 notify 模块的 Bind 写法保持一致。
- **路由顺序**：`RegisterRoutes` 中 12 条路由按"会议生命周期 → 成员管理 → 邀请 → 聊天"的业务流排列，与设计文档 §5.3 的清单逐一对应。

### 验证执行

1. `go build ./...` → 无任何 warning/error
2. `go vet ./...` → 无提示
3. `go run -mod=mod github.com/google/wire/cmd/wire ./app/provider` → `wire_gen.go` 成功重生成
4. `GIN_MODE=debug go run cmd/server/main.go` 后台启动 → 日志打印 12 条 `[GIN-debug] ... meeting/controller.(*MeetingController).XxxRoom-fm (6 handlers)`，与路由表一一匹配
5. `curl -X POST http://localhost:8085/api/v1/meeting/rooms`（无 token）→ **401** `{"code":401,"message":"缺少认证信息",...}`，JWT 中间件生效
6. `curl -X GET http://localhost:8085/api/v1/meeting/rooms` → **401**
7. `curl -X POST http://localhost:8085/api/v1/meeting/invites/abc/redeem` → **401**
8. `pkill -f "go run cmd/server/main.go"` → 进程退出，端口 8085 释放

### 下一步

- **Task 5**（1.5 人日）：`MeetingService.CreateRoom` + `JoinRoom` + `LeaveRoom` + `EndRoom` 核心业务逻辑（6 位会议号生成 + bcrypt 密码校验 + 人数上限 + Redis host 宽限期 Timer 骨架），替换当前 `ErrNotImplemented` 占位。

---

## 🎯 2026-04-21 Phase 2e-2 Task 7 HTTPMediaOrchestrator 落地（Go↔Node 真实媒体链路打通）

**交付**：将 `NoopMediaOrchestrator` 替换为真实的 `HTTPMediaOrchestrator`，`MeetingService` / `MeetingSignalService` 的 9 个 media 调用全部走 HTTP 到 Node `media-server` 的 `/internal/v1/*` API；端到端验证脚本 `docs/verify/meeting_t7_verify.mjs` **16/16 PASS**，日志确认 Go 真正触发 mediasoup 真实 `router created` / `webrtc transport created` / `router closed explicitly` 事件。

### 产出文件

| 文件 | 行数 | 作用 |
|---|---|---|
| `backend/go-service/config/config.go`（改） | +15 | 新增 `MediaServerConfig{BaseURL, InternalToken, TimeoutMS, CloseTimeoutMS, CloseRetry}`，挂接到 `Config.MediaServer` |
| `backend/go-service/config/config.dev.yaml`（改） | +8 | 新增 `media_server` 配置段（`http://localhost:3300` + 与 `media-server/.env` 一致的共享密钥） |
| `backend/go-service/config/config.docker.yaml`（改） | +8 | 同上，`base_url` 改为 Docker 网络内 `http://media-server:3300` |
| `backend/go-service/app/meeting/service/http_media_orchestrator.go`（新） | 340 | 实现 `MediaOrchestrator` 8 方法；`net/http` 标准库 + `context` 驱动超时；创建类 5s 超时无重试，关闭类 2s 超时 + 最多 2 次指数退避（200/500ms）；`roomCode → routerID` `sync.Map` 本地缓存满足 §6.6 的 `CloseRouter(roomCode)` 契约；错误统一映射为 `ErrMediaResourceNotFound`（Node 404）或 `ErrMediaServerError`（5xx/超时/网络错） |
| `backend/go-service/app/meeting/provider.go`（改） | ±4 | `MeetingSet` 的 `NewNoopMediaOrchestrator` 替换为 `NewHTTPMediaOrchestrator`，`wire.Bind` 指向新实现 |
| `backend/go-service/app/provider/wire_gen.go`（改） | ±3 | wire 自动图里 `noopMediaOrchestrator := NewNoopMediaOrchestrator()` 改为 `httpMediaOrchestrator := NewHTTPMediaOrchestrator(cfg)`，两个消费点同步替换 |
| `docs/verify/meeting_t7_verify.mjs`（新） | 140 | E2E 验证脚本：健康检查 + 错 token 401 兜底 + REST 创建/加入/结束会议 + WS room.join + 真实 mediasoup transport.create（断言 `transport.id` 非 `noop-` 前缀 + `iceCandidates[]` 非空 + `dtlsParameters.fingerprints[]` 存在）+ 404 幂等关闭 producer |

### 关键设计决策

1. **关闭类幂等重试 vs 创建类一次性透传**：设计 §6.6 明确要求关闭类"失败重试 2 次（仅幂等的关闭类操作）"。HTTP 实现里通过独立的 `doCloseRequest` 函数与 `attempts = CloseRetry + 1` 循环实现；创建类直接走 `doRequest` 不重试，避免产生"客户端以为没创建成功但 Node 侧已创建"的孤儿资源。
2. **`roomCode ↔ routerID` 本地缓存**：设计 §6.6 规定 `CloseRouter(ctx, roomCode)` 入参为 `roomCode`，但 Node 的 `DELETE /routers/:routerId` 以 `routerID` 为主键。`HTTPMediaOrchestrator` 在 `CreateRouter` 成功后把映射存入 `sync.Map`，`CloseRouter` / `CreateTransport` / `CreateConsumer` 都从缓存反查。go-service 重启后缓存丢失，此时 Node 也已随进程重启释放 Router，状态自然同步。
3. **错误语义区分**：`ErrMediaResourceNotFound` 用于 404（资源已不存在，`CloseProducer` / `CloseConsumer` / `CloseRouter` 幂等转 nil），`ErrMediaServerError` 用于其它异常（5xx / 网络错 / 超时 / 序列化错），上层 `MeetingSignalService` 可通过 `errors.Is` 精准区分并在 WS ACK 里给出差异化提示。
4. **配置分层**：`media_server.internal_token` 在开发环境写 yaml 方便联调；生产环境通过 `ECHOCHAT_MEDIA_SERVER_INTERNAL_TOKEN` 环境变量覆盖。`base_url` 也按环境差异化（dev 走 `localhost`，docker 走服务名 `media-server`）。
5. **接口保持 8 方法（未落地 `ResumeConsumer`）**：设计 §6.6 的 `NodeClient` 第 9 方法 `ResumeConsumer` 在 Node 已实现（`POST /consumers/:id/resume`），但当前 WS 契约未暴露对应 C→S 事件；为保持 Task 7 最小侵入，暂不在 `MediaOrchestrator` 接口增加该方法，留待 Task 9 前端 mediasoup-client 接入时按需补齐（前端拉取 Consumer 后通常需要 resume 解除 Node 侧的默认 paused）。

### E2E 验证（16/16 PASS）

```text
PASS: media-server healthz
PASS: media-server /internal/info (token ok)
PASS: media-server rejects wrong token (401)
PASS: register + login 2 users
PASS: POST /meeting/rooms returns 201            ← Go 内部调 CreateRouter 成功
PASS: room_code returned
PASS: user B join success                         ← JoinRoom 内部再次调 CreateRouter
PASS: A WS room.join ok
PASS: B WS room.join ok
PASS: A transport.create returned ok              ← WS 信令经 Go 透传到 Node
PASS: transport.id is real (not noop-)            ← 证明非占位，真实 mediasoup transport id
PASS: iceParameters is an object (non-empty from Node)
PASS: iceCandidates[] non-empty (proves real mediasoup transport)
PASS: dtlsParameters.fingerprints[] present       ← 真实 DTLS 指纹
PASS: 404 mapped to ok (idempotent close)         ← 关闭不存在的 producer 幂等 ok
PASS: host end meeting ok                         ← 触发 CloseRouter
```

media-server 日志同步确认：
```
[17:35:51.641] INFO: router created
[17:35:51.647] INFO: router created
[17:35:51.660] INFO: webrtc transport created
[17:35:51.669] INFO: transport closed and removed from map
[17:35:51.670] INFO: router closed and removed from map
[17:35:51.670] INFO: router closed explicitly
```

### 已知待改进项（留给后续 Task）

1. **`CreateRoom` 与 `JoinRoom` 各自调一次 `CreateRouter`**（Task 5 遗留行为）：当前每次有用户加入都会尝试重新创建 Router，Node 没做去重，最新一次会覆盖本地缓存指向新 Router。Task 8（生命周期状态机）需修复为"仅在房间第一次被创建时调 CreateRouter"，并在 Task 5 的 `JoinRoom` 里改为查 Router 复用。
2. **`ResumeConsumer` 未暴露**：见上面决策 5。Task 9 前端集成前需在 `MediaOrchestrator` 补齐并加 WS `meeting.consume.resume` 事件（或直接在 `CreateConsumer` 后端自动 resume）。
3. **Transport 状态心跳拉取**：设计 §10.4 提到 Go 每 30s 拉 `/internal/v1/transports/:id/stats`，当前 Node 未实现该接口；Task 10（可观测性）统一补齐。

### 下一步

进入 **Task 8：会议生命周期状态机**（host 宽限期 + 自动转让 + 空房 TTL），依赖 Task 5/7 已落地的 DAO 与 HTTPMediaOrchestrator；预估 0.5 人日。

---

## 🚀 2026-04-21 Phase 2e-2 Task 6 WebSocket 信令协议（13 事件）落地

**交付**：`meeting.*` 事件族从 Task 5 的 `PublishToUser` 循环升级为完整的 WS 信令协议；新建 `MeetingBroadcaster`（统一广播层）、`MeetingSignalService`（8 个 C→S 事件业务逻辑 + 资源追踪）、`MeetingWSHandler`（controller 薄层），`MediaOrchestrator` 接口扩容至 9 个方法覆盖 mediasoup 全生命周期（Task 7 真实实现前由 `NoopMediaOrchestrator` 占位）；端到端 WS 冒烟脚本 `/tmp/meeting_ws_t6_test.mjs` **18/18 PASS**，覆盖 8 C→S 白名单事件 + 3 S→C 广播 + 3 类错误路径。

### 产出文件

| 文件 | 行数 | 作用 |
|---|---|---|
| `backend/go-service/app/constants/meeting.go`（改） | +25 | WS 事件常量与设计 §6.3 对齐（3 房间 + 5 成员 + 5 媒体 + 1 聊天），新增 `MeetingWSClientEvents` 白名单切片限制客户端只能发起 8 个 C→S 事件 |
| `backend/go-service/app/meeting/service/interfaces.go`（重构） | 180 | `MediaOrchestrator` 扩容到 9 方法（Router/Transport/Producer/Consumer 全生命周期）+ 配套 DTO（`TransportInfo` / `ConsumerInfo` / `CreateTransportReq` / `CreateProducerReq` / `CreateConsumerReq`）+ `NoopMediaOrchestrator` 9 个占位实现（stub ID + 最小 JSON） |
| `backend/go-service/app/meeting/service/meeting_broadcaster.go`（新） | 75 | `MeetingBroadcaster`：`BroadcastToMeeting`（查询活跃 participant → 批量 `PubSub.PublishToUser` + 可选 exclude）+ `PublishToUser`（定向推送）+ 并发安全的错误汇集 |
| `backend/go-service/app/meeting/service/meeting_service.go`（改） | ±30 | 12 个 REST 方法重构：统一改为调用 `broadcaster.BroadcastToMeeting` / `broadcaster.PublishToUser`，移除直连 `ws.PubSub` 依赖，代码量精简约 15% |
| `backend/go-service/app/meeting/service/meeting_signal_service.go`（新） | 430 | `MeetingSignalService` 8 个 C→S 事件（`OnRoomJoin`/`OnRoomLeave`/`OnMemberStateChanged`/`OnTransportCreate`/`OnTransportConnect`/`OnProduceStart`/`OnConsumeStart`/`OnProducerClose`）+ Redis 资源追踪 `echo:meeting:resources:{room_id}:{user_id}`（Set 结构，TTL 1 小时）+ `cleanupUserResources`（WS 断开钩子调用）+ host 权限校验（非 host 改他人状态返回 `仅主持人可执行此操作`） |
| `backend/go-service/app/meeting/controller/meeting_ws_handler.go`（新） | 200 | `MeetingWSHandler` 薄层：构造时调用 `hub.RegisterEvent` 注册 8 C→S 事件，每个 handler 仅负责 JSON 反序列化 + 调 `signalSvc.On*` + 构造 ACK（`code=0/-1` + `message`）|
| `backend/go-service/app/meeting/provider.go`（改） | +4 | `MeetingSet` 补全 `NewMeetingBroadcaster` / `NewMeetingSignalService` / `NewMeetingWSHandler` |
| `backend/go-service/app/provider/{provider,wire_gen}.go`（改） | +8 | `App` 结构体新增 `MeetingSignalService` / `MeetingWSHandler` 字段，`wire` 重新生成 |
| `docs/api/frontend/meeting.md`（追加 2 节） | +200 | 新增 §WebSocket 信令协议（Task 6）：16 事件总览表 + 8 C→S 事件完整请求/ACK/广播契约 + S→C 广播契约 + 架构说明 + 错误处理表；§验证记录补充 Task 6 结果 |

### 8 C→S 白名单事件契约

| 事件 | 入参关键字段 | ACK data | 副作用 |
|------|-------------|---------|--------|
| `meeting.room.join` | room_code | `{}` | 校验活跃参会记录 |
| `meeting.room.leave` | room_code | `{}` | 清理该用户所有 transport/producer/consumer |
| `meeting.member.state.changed` | room_code, [target_user_id], audio_enabled?, video_enabled?, hand_raised? | `{}` | 广播 `meeting.member.state.changed`；非 host 改他人 → `-1` |
| `meeting.transport.create` | room_code, direction(send/recv) | `{id, iceParameters, iceCandidates, dtlsParameters}` | 资源追踪 Redis set |
| `meeting.transport.connect` | room_code, transport_id, dtls_parameters | `{}` | mediasoup connect |
| `meeting.produce.start` | room_code, transport_id, kind, rtp_parameters | `{producer_id}` | 广播 `meeting.member.producer.new`；资源追踪 |
| `meeting.consume.start` | room_code, transport_id, producer_id, rtp_capabilities | `{id, producerId, kind, rtpParameters}` | 资源追踪 |
| `meeting.producer.close` | room_code, producer_id | `{}` | 广播 `meeting.member.producer.new closed=true` |

### 验证执行（Node.js + ws）

1. `go build ./...` / `go vet ./...` / `wire ./app/provider` 全绿
2. 启动 server，跑 `/tmp/meeting_ws_t6_test.mjs`（双用户 + WS_TRACE 模式）
3. 用例清单（18 个全部 PASS）：
   - 房间/成员：`room.join` 双端 ACK、`state.changed` 自我静音 ACK + 对端广播、host 强制静音 ACK、**非 host 强制静音他人 → `-1 仅主持人可执行此操作`**
   - 媒体：`transport.create` send/recv 双向、`transport.connect`、`produce.start` ACK + `producer.new` 广播、`consume.start`、`producer.close` ACK + `producer.new closed=true` 广播
   - 离会/错误：`room.leave` + `member.left` 对端广播、不存在会议号 `room.join` → `-1`
4. 测试期间修复 waitEvent 死循环 bug（msgQueue pop→push 自循环导致 ack 永远等不到）→ 改为 stash 临时缓冲区，完成后统一归还

### 关键设计决策

1. **Broadcaster 单独抽层**：避免 service 方法散落直接 `PubSub.PublishToUser`，后续接入 Redis Cluster / 切换广播实现只需改一个文件；同时 Task 5 的 REST 广播与 Task 6 的 WS 事件广播完全复用同一个对象。
2. **C→S 白名单机制**：`app/constants/meeting.go:MeetingWSClientEvents` 列出 8 个允许客户端发起的事件，`ws.Hub` 在分发前先过滤，防止恶意客户端直接发 `meeting.room.ended` 伪造房间结束。
3. **资源追踪用 Redis Set**：每创建一个 transport/producer/consumer 都 `SADD echo:meeting:resources:{room_id}:{user_id} <resource_id>`，WS 断开或 room.leave 时 `SMEMBERS` 遍历清理；TTL 1 小时防止遗留占用，即使 Go 进程崩溃也不会泄漏 mediasoup 资源。
4. **MediaOrchestrator 先抽 9 方法再实现**：Task 6 仍用 `Noop` 占位，但接口已完整定义 `CreateRouter / CloseRouter / CreateTransport / ConnectTransport / CreateProducer / CloseProducer / CreateConsumer / CloseConsumer`（外加 DTO 型号），Task 7 只需替换绑定即可让 WS 端变为真实 mediasoup，**无需修改 signal service / handler 代码**。
5. **ACK `code` 语义统一**：成功 `0`、业务失败 `-1`（+ 中文 message），与 REST 领域错误口径完全一致，前端可直接复用一套 error toast 组件。
6. **`meeting.room.leave` 只清 WS 资源不改 participant 表**：真正离会需 REST `/leave`（会影响 duration / host 自动转让）；此设计允许客户端 WS 重连时发 leave+join 刷新 transport 而不退会。

### 下一步

- **Task 7**（1.5 人日）：`HTTPMediaOrchestrator` 实现 — Go 调 Node media-server 9 个内部 REST API（Task 2 已全部跑通），替换 `NoopMediaOrchestrator`，前后端 WS 契约完全不变。
- **Task 8**（2 人日）：Vue 前端 mediasoup-client 接入 + 会议室页面骨架，按本次 §WebSocket 信令协议契约实现 `Transport.connect / produce` 回调。

---

## 🚀 2026-04-21 Phase 2e-2 Task 5 Go meeting 模块 12 个 REST 接口业务逻辑全量落地

**交付**：`MeetingService` 12 个业务方法 + `MeetingController` 12 个 Gin 处理器从 501 占位升级为真实实现，完整的领域错误码映射、DTO 绑定、DAO 契约修复、权限辅助函数；端到端验证脚本 `/tmp/meeting_t5_test.sh` **19/19 PASS**，覆盖 12 接口 happy path + 5 类错误路径；`go build ./...` / `go vet ./...` / `wire ./app/provider` 全绿。

### 产出文件

| 文件 | 行数 | 作用 |
|---|---|---|
| `backend/go-service/app/dto/meeting_dto.go` | 169 | 13 个 DTO：`MeetingRoomDTO`/`MeetingParticipantDTO`/`MeetingChatDTO` 基础 + 10 个请求/响应类型（`CreateMeetingRoomRequest`/`JoinMeetingRoomRequest`/`InviteUsersRequest`/`KickMemberRequest`/`TransferHostRequest`/`SendMeetingChatRequest`/`ListMyMeetingsRequest`/`ListMeetingChatsRequest`/`RedeemInviteTokenResponse` 等） |
| `backend/go-service/pkg/utils/meeting_code.go` | 40 | `GenerateMeetingRoomCode()` 生成 9 位 `XXX-XXX-XXX` 会议号（crypto/rand + 3 组 3 位数字）；`GenerateMeetingInviteToken()` 生成 32 位 hex 邀请令牌 |
| `backend/go-service/app/meeting/service/interfaces.go`（改） | +25 | 新增 `MediaOrchestrator` 接口 + `NoopMediaOrchestrator` 占位实现（Task 7 替换为真实 HTTP 客户端） |
| `backend/go-service/app/meeting/service/meeting_service.go`（重写） | 800 | 12 个业务方法 + 11 个领域错误（`ErrMeetingPasswordLocked`/`ErrAlreadyInOtherMeeting` 等）+ `assertIsActiveParticipant`/`assertIsHost`/`generateUniqueRoomCode`/`broadcastToActiveParticipants` 辅助，注入 `MediaOrchestrator` + `ws.PubSub` 完成广播 |
| `backend/go-service/app/meeting/controller/meeting_controller.go`（重写） | 420 | 12 个 Gin 处理器 + `handleError` 领域错误 → HTTP 状态码映射 + `roomToDTO`/`participantToDTO`/`chatToDTO` 转换 + `requireUserID` 统一鉴权辅助 |
| `backend/go-service/app/meeting/router.go`（改） | ±5 | 路径对齐设计文档：`GET /rooms` → `GET /rooms/mine`；`POST /invites/:token/redeem` → `POST /invite-tokens/:token/redeem` |
| `backend/go-service/app/meeting/provider.go`（改） | +4 | `MeetingSet` 加入 `NewNoopMediaOrchestrator` + `wire.Bind(MediaOrchestrator, NoopMediaOrchestrator)` |
| `backend/go-service/app/meeting/dao/meeting_room_dao.go`（改） | ±8 | **DAO 契约修复**：`GetByID`/`GetByCode` 将 `gorm.ErrRecordNotFound` 转为 `(nil, nil)`，由 service 层用 `room == nil` 判定 |
| `backend/go-service/app/meeting/dao/meeting_participant_dao.go`（改） | ±6 | **DAO 契约修复**：`GetByRoomAndUser`/`FindActiveByUser` 同上转换 |
| `docs/api/frontend/meeting.md`（重写） | 280 | Phase 2e-2 MVP 的 12 接口完整 API 文档（路径总览 + 领域错误码映射表 + 12 接口详细参数/响应示例 + WebSocket 事件关联表） |

### 验证执行（3 用户端到端）

1. `go build ./...` / `go vet ./...` / `wire` 全绿
2. 启动 server 并跑 `meeting_t5_test.sh`（A/B/C 三用户场景）：
   - 12 接口 happy path：`CreateRoom`(带/不带密码)/`GetRoomByCode`/`JoinRoom`(含密码)/`LeaveRoom`/`EndRoom`/`TransferHost`/`KickMember`/`InviteUsers`/`RedeemInviteToken`/`SendChat`/`ListChats`/`ListMyMeetings`
   - 5 类错误路径：密码错误(400) / 房间不存在(404) / 单点参会冲突(400) / 非 host 越权(403) / 邀请链接失效(400)
   - 结果：**PASS=19 / FAIL=0**
3. DB 侧核查 `meeting_rooms.status` / `meeting_participants.left_at/duration` / `meeting_chats` 记录写入正确；Redis 侧核查 `echo:meeting:invite:{token}` key 的 TTL=600s
4. 服务端日志全链路 trace_id 串联，WS 广播 `meeting.member.joined/left/chat/host.changed/room.ended` 事件通过 `PubSub.PublishToUser` 逐人推送

### 关键设计决策

- **DAO 契约统一**：所有"按主键/唯一键查单条"的 DAO 方法一律将 `gorm.ErrRecordNotFound` 转换为 `(nil, nil)`，service 层统一以 `result == nil` 判定并返回领域错误（`ErrMeetingNotFound` 等）。消除此前 500 误报问题。
- **Stub 策略**（Task 5 阶段）：
  - `MediaOrchestrator.CreateRouter/CloseRouter` 当前为 Noop（返回占位字符串），Task 7 引入 HTTP 客户端调 Node media-server
  - WS 广播暂用 `pubsub.PublishToUser` 逐人循环，Task 6 封装为 `BroadcastToMeeting` 后接口无感替换
  - `NotifyPusher.PushBatch`（Phase 2e-1 成果）直接复用，`meeting_invite` 类型的通知已由 NotifyService 正确处理
- **单点参会**：通过 `meeting_participants` 关联 `meeting_rooms.status != 2` 判断一个用户是否已在活跃会议中，避免同时多会议产生混乱（`ErrAlreadyInOtherMeeting`）
- **密码限流**：同 `(user_id, code)` 5 次内错自动触发 Redis 锁 `echo:meeting:pwd:fail:{code}:{user_id}` TTL 10 分钟（`ErrMeetingPasswordLocked`），防止暴力破解
- **host 离会自动转让**：host leave 时若仍有其他活跃成员，自动将 host 转移到"最早加入者"（`ORDER BY joined_at ASC LIMIT 1`），并广播 `meeting.host.changed`；若仅 host 一人则房间标记 `ended_reason=empty_ttl`
- **邀请 token 安全**：响应体**不**返回 token，仅通过 `NotifyPusher.PushBatch` 的 `Extra.invite_token` 定向下发给被邀请者；兑换后保留 60 秒冗余（允许页面刷新），随后 Redis TTL 自然过期
- **HTTP 状态码**：创建类接口（CreateRoom / SendChat）统一返回 **201 Created**；动作类接口（Join/Leave/End/Kick/TransferHost/Invite/Redeem）返回 **200 OK**；领域错误按"资源不存在=404 / 权限不足=403 / 业务规则=400"三档映射

### 下一步

- **Task 6**（2 人日）：WebSocket 信令协议落地 — `meeting.*` 事件帧 + mediasoup Transport/Producer/Consumer signaling 桥接 + `ws.BroadcastToMeeting` 替换 Task 5 的 `PublishToUser` 循环。
- **Task 7**（1.5 人日）：Go → Node HTTP 客户端 `HTTPMediaOrchestrator`，接入真实 mediasoup Router，替换 `NoopMediaOrchestrator`。

---

## 🚀 2026-04-21 Phase 2e-2 Task 2 Router/Transport/Producer/Consumer 核心内部 REST API 完成

**交付**：`media-server/` 的 9 个内部 REST API 全部落地 + zod 请求校验 + AppError 统一错误响应 + observer-close 自清理 + 58 个 vitest 单元/集成测试（覆盖率 **80.89%**），9 接口 happy-path + 6 类错误路径全部手动验证通过。

### 产出文件
| 文件 | 规模 | 作用 |
|---|---|---|
| `media-server/src/schemas/common.ts` | 20 行 | `idStringSchema` / `roomCodeSchema` / `userIdSchema` / `okResponseSchema`（共用基础 schema） |
| `media-server/src/schemas/router.schema.ts` | 10 行 | `createRouterBodySchema` + `routerIdParamSchema` |
| `media-server/src/schemas/transport.schema.ts` | 30 行 | `transportDirectionSchema` + `create/connectTransportBodySchema` + DTLS fingerprint 严格校验 |
| `media-server/src/schemas/producer.schema.ts` | 12 行 | `mediaKindSchema` + `createProducerBodySchema` |
| `media-server/src/schemas/consumer.schema.ts` | 10 行 | `createConsumerBodySchema` + `consumerIdParamSchema` |
| `media-server/src/utils/errors.ts` | 45 行 | `AppError`（5 种 code → 404/409/400/500/503 状态码映射）+ `notFound` / `conflict` 辅助函数 |
| `media-server/src/middlewares/error-handler.ts` | 55 行 | 统一错误处理：`ZodError`→400 / `AppError`→对应 status / Fastify 4xx 透传 / 未知错误→500 |
| `media-server/src/mediasoup/codecs.ts` | 28 行 | `MEDIA_CODECS`（opus + VP8 + H264，与 PoC 完全一致） |
| `media-server/src/services/router.service.ts` | 85 行 | Router Map + `maxRouters` 限制 + observer close 自清理 + 统计接口 |
| `media-server/src/services/transport.service.ts` | 145 行 | Transport Map + `listenIps` 构建（announcedIp 可选）+ connect 幂等冲突校验 + 错误包装 |
| `media-server/src/services/producer.service.ts` | 95 行 | Producer Map + send-direction 校验（recv transport 禁止 produce）+ 错误包装 |
| `media-server/src/services/consumer.service.ts` | 130 行 | Consumer Map + recv-direction 校验 + `router.canConsume` 检查 + paused-on-create + `producerclose` 自动关闭 |
| `media-server/src/routes/{router,transport,producer,consumer}.route.ts` | 共 ~95 行 | 9 个接口分文件挂载，均调用对应 zod schema.parse + service 层，专注薄 controller |
| `media-server/src/app.ts` | +18 行 | 注册 `registerErrorHandler` + 以 `/internal/v1` 前缀挂载 4 组路由 |
| `media-server/vitest.config.ts` + `tests/setup.ts` | - | vitest forks pool + 覆盖率 v8 + 自动注入测试 env（silent 日志 + 专用 RTC 端口段 40800-40899） |
| `media-server/tests/{schemas,errors,app}.spec.ts` + `tests/services/*.spec.ts` | 7 文件 / 58 测试 | schemas + AppError + 4 services + HTTP 层集成 |

### 9 个内部 REST API 清单（挂载在 `/internal/v1`）

| # | 方法+路径 | 成功状态 | 说明 |
|---|---|---|---|
| 1 | POST /routers | 201 | 创建 mediasoup Router（含房间限额检查） |
| 2 | DELETE /routers/:routerId | 200 | 显式关闭 Router（observer close 自动清理 map） |
| 3 | POST /transports | 201 | 创建 WebRtcTransport（send/recv 两方向） |
| 4 | POST /transports/:id/connect | 200 | DTLS connect（二次调用返回 409 CONFLICT） |
| 5 | POST /producers | 201 | 仅允许在 send transport 上创建 |
| 6 | DELETE /producers/:id | 200 | 显式关闭 Producer |
| 7 | POST /consumers | 201 | 仅允许在 recv transport；`router.canConsume` 失败 → 400 CAN_NOT_CONSUME；创建后 `paused=true` |
| 8 | POST /consumers/:id/resume | 200 | 客户端 `transport.consume` 成功后调用，避免首帧丢失 |
| 9 | DELETE /consumers/:id | 200 | 显式关闭 Consumer |

### 统一错误响应格式

| Code | HTTP | 场景 |
|---|---|---|
| `UNAUTHORIZED` | 401 | 缺失/错误 `X-Internal-Token` |
| `VALIDATION_ERROR` | 400 | zod 校验失败；body 含 `fieldErrors[]{path,code,message}` |
| `NOT_FOUND` | 404 | router / transport / producer / consumer id 不存在 |
| `CONFLICT` | 409 | 重复 connect；recv transport 上尝试 produce；send transport 上尝试 consume |
| `CAN_NOT_CONSUME` | 400 | `router.canConsume` 返回 false（rtpCapabilities 不兼容） |
| `ROUTER_LIMIT_EXCEEDED` | 503 | 活跃 router 超过 `MEDIASOUP_MAX_ROUTERS` |
| `MEDIASOUP_ERROR` | 500 | mediasoup 层抛错（透明包装，含 transportId 等 details） |
| `INTERNAL_ERROR` | 500 | 未分类异常（同时 error 级日志落盘） |

### 测试验收实测

| 维度 | 命令 | 结果 |
|---|---|---|
| 类型校验 | `npm run typecheck` | 0 错误 |
| 代码规范 | `npm run lint` | 0 错误 |
| 单测 | `npm test` | **58 passed / 0 failed**（7 个 spec 文件，~900ms） |
| 覆盖率 | `npx vitest run --coverage` | **statements 80.89% / branches 76.03% / functions 90.9% / lines 80.89%**（远超 60% 目标） |
| 健康探测 | `curl /healthz` + `/readyz` | 200 + `ok:true` |
| 鉴权 | 无/错 token → 401、正确 token → 200 | 通过 |
| 9 接口 happy path | 手动 curl（见下） | 通过 |
| 错误路径 | unknown router→404 / produce on recv→409 / delete unknown→404 / resume unknown→404 / 小写 roomCode→400 / 缺 token→401 | 全部按预期返回 |

### 代码覆盖率明细（v8）

| 模块 | Stmts | 备注 |
|---|---|---|
| schemas/* | 100% | 全部 5 个 schema 文件 |
| utils/errors.ts | 100% | 5 种 AppError code 全覆盖 |
| mediasoup/codecs.ts | 100% | - |
| routes/* | 91.26% | 未覆盖为 error 分支的 catch（由 e2e 真实 mediasoup 触发） |
| middlewares/internal-auth.ts | 100% | 无 token / 错 token / 正确 token / 白名单 4 条分支 |
| middlewares/error-handler.ts | 63.79% | 未触发 Fastify 内置 4xx 透传分支（属 happy case） |
| services/router.service.ts | 97.95% | 仅 `_clearRouterMap` 内部 try/catch 未触发 |
| services/transport.service.ts | 81.57% | - |
| services/producer.service.ts | 84.4% | - |
| services/consumer.service.ts | 47.22% | happy path 需真实 WebRTC 连接（由 Task 9 前端 E2E 覆盖） |
| mediasoup/worker.ts | 68.18% | 重启路径需故意 kill worker 触发（集成环境） |

### 关键工程决策

1. **zod.parse 手动调用 + 全局错误处理器**：放弃 `fastify-type-provider-zod`（避免引入 zod v4 依赖冲突），改为每个 handler 内显式 `.parse()`，由 `setErrorHandler` 统一捕获 `ZodError` → 400。依赖面小、行为直观、不牺牲安全性。
2. **Map + observer.once('close') 自清理**：所有 service 层都以 `Map<id, Entry>` 持有资源，并在创建时 `observer.once('close', () => map.delete(id))`；无论外部主动 `close()` 还是上游级联关闭（router→transport→producer→consumer），map 都自动收敛，杜绝泄漏。
3. **Consumer `paused:true` 强约束 + `producerclose` 级联**：严格遵循 mediasoup 官方推荐 —— 服务端创建后总是 paused，等客户端 `transport.consume()` 成功后再调 `/resume`；同时监听 `producerclose` 自动关闭下游 consumer，防止对端已关闭但本端仍占流的幽灵资源。
4. **direction 强约束**：producer 只允许 send transport、consumer 只允许 recv transport，违反即 409 CONFLICT；从 API 层就隔断"send 上混消费"这类难以排查的状态错误。
5. **AppError 扁平化错误码 + details**：`{ code, message, details?}` 形式，前端/Go 后端都能用 `error.code === 'NOT_FOUND'` 精准分支，避免依赖 message 字符串。

### 代码审查修复（`code-reviewer` 子代理，2026-04-21）

子代理总评"**有条件通过**"（0 Blocker / 2 Major / 10 Minor / 10 Nits / 5 亮点），2 Major + 4 高价值 Minor 已全部当场修复，剩余 Minor/Nits 延后至 Task 16 收尾时清扫。

| 编号 | 级别 | 问题 | 修复 | 文件 |
|---|---|---|---|---|
| M1 | Major | `_clearXxxMap` 无守卫，生产环境可误调用销毁全部资源 | 新增 `src/utils/test-guard.ts#assertTestOnly`，4 个 service 的 `_clearXxxMap` 首行调用，`NODE_ENV !== 'test'` 直接抛错 | `src/utils/test-guard.ts` + 4 个 service.ts |
| M2 | Major | `rtpParameters`/`rtpCapabilities` 仅用 `z.record(z.string(), z.unknown())`，空对象直接走到 mediasoup 层被包成 500 | 新增 `src/schemas/rtp.ts`，对 codecs 做最小结构校验（mimeType/clockRate/payloadType 必填，codecs 数组 ≥1），消除 `as unknown as` 双跳断言，客户端参数错误现在正确返回 400 VALIDATION_ERROR | `src/schemas/rtp.ts` + `producer.schema.ts` / `consumer.schema.ts` / `producer.route.ts` / `consumer.route.ts` |
| m1 | Minor | `connectTransport` 在 `await transport.connect` 前没置位 `connected`，并发重复请求被 mediasoup 包成 500 | 改为乐观锁：先 `entry.connected = true` 再 await，失败时回退为 false | `src/services/transport.service.ts` |
| m2 | Minor | 返回 `producerPaused: producer.paused` 语义不如 `consumer.producerPaused`，且阻碍未来跨进程 PipeTransport | 改读 `consumer.producerPaused` | `src/services/consumer.service.ts` |
| m3 | Minor | `consumer.on('producerclose', ...)` 风格不统一（事件只触发一次） | 改为 `consumer.once(...)`，与其他 observer 语义一致 | `src/services/consumer.service.ts` |
| m5 | Minor | `internal-auth` 使用正向白名单（`/healthz`/`/readyz`），未来新增 `/metrics`/`/docs` 等公共端点容易漏加 | 改为**反向白名单** `PRIVATE_PATH_PREFIXES = ['/internal/']`，默认开放，仅私有前缀强制校验 | `src/middlewares/internal-auth.ts` |

### 修复后验收

| 维度 | 结果 |
|---|---|
| typecheck | 0 错误 |
| lint | 0 错误 |
| vitest | **65 passed / 0 failed**（新增 rtp schema 校验 4 测试、`test-guard` 2 测试、consumer CAN_NOT_CONSUME 细分测试 1） |
| 覆盖率 | **stmts 82.87% / branches 75.83% / funcs 91.3% / lines 82.87%**（较修复前 80.89% 提升 ~2 pp） |
| `internal-auth` 覆盖率 | 从 92.3% → **100%** |
| `schemas` 覆盖率 | 新增 `rtp.ts` 后仍保持 **100%** |

### 延后处理清单（Task 16 收尾）

- m4 `tryGetRouter` 未被引用 → 决定留给 Task 7 Go `NodeClient` 健康探测使用，加 `@internal` JSDoc 即可
- m6 `rtpCapabilities` 更精细校验 → 随 Task 9 前端 mediasoup-client 对接时结合真实报文完善
- m7 logger redact 覆盖面 → 加 `'*.headers["x-internal-token"]'` 通配符
- m8 `userIdSchema` 与 Go 侧契约对齐 → Task 7 实装后统一收敛
- m9 worker.died 级联清理 → Task 16 补 chaos 测试时加 `drainXxxMap`（无副作用纯清 map）
- m10 requestId 贯穿 → Task 7 `NodeClient` 头部注入 `X-Request-ID` 时一体化实现
- n1-n10 均为风格类项，不阻塞

### 下一步

- **Task 3**：Go 侧 `meeting` 模块数据库 DDL + Model + DAO（对齐设计 §5.1 的 3 张表），预计 0.5 人日
- Task 2 修复后新增文件：`src/schemas/rtp.ts`、`src/utils/test-guard.ts`、`tests/test-guard.spec.ts`；新增单测 7 个（共 65 个）

---

## 🚀 2026-04-21 Phase 2e-2 Task 1 media-server 项目骨架完成（含 Fastify 5 升级）

**交付**：正式 `media-server/` 子项目骨架落盘 + 原地升级至 Fastify 5 → 锁定 **mediasoup 3.19.0 + fastify 5.8.5 + fastify-plugin 5.1.0 + @fastify/sensible 6.0.4 + @fastify/websocket 11.2.0 + pino 9.3.2 + zod 3.23.8**，`/healthz` + `/internal/*` + `X-Internal-Token` 鉴权 + Worker `died` 自动重启全部通过本机验证，pino 统一为单实例。

### 🔁 2026-04-21 升级补充：Fastify 4 → 5

- **升级理由**：Fastify 4 已于 2025-06-30 结束官方 LTS 支持（到 2026-04 已过保约 10 个月）；v5 最新 5.8.5（2026-03）、插件生态（@fastify/websocket 11.x / @fastify/sensible 6.x / fastify-plugin 5.x）均已 GA 支持；且可一步消灭前一轮由类型系统限制造成的两个 pino 实例
- **实际改动**：`package.json` 4 行版本号 + `src/app.ts`：`logger: buildLoggerOptions()` → `loggerInstance: logger`（回归 pino 单实例）
- **兼容性确认**（Fastify 5 破坏性变更逐条核查）：
  - Node.js ≥20：Dockerfile 已用 `node:20-bookworm-slim` ✅
  - 完整 JSON Schema 校验：我们 Task 2 规划用 `zod` 完整 schema，无 shorthand 残留 ✅
  - `.listen()` 对象签名：已用 `app.listen({ host, port })` ✅
  - Plugin 纯 async：已用 `fp(async (fastify) => {...})` ✅
- **回归实测**：`npm install` 319 包 / `typecheck` + `lint` 0 错误 / `curl /healthz` + `/internal/info` 401/200 / `kill -9 <workerPid>` → 1s 内新 Worker 上线
- **日志行为验证**：启动输出两行 `Server listening at ...` 来自 Fastify 内部日志、`media-server listening` 来自业务逻辑，**格式/时间戳/pid 完全一致**，确认单实例生效

### 产出文件

| 文件 | 规模 | 作用 |
|---|---|---|
| `media-server/package.json` | - | 锁定依赖（升级后）：mediasoup@3.19.0 / fastify@5.8.5 / fastify-plugin@5.1.0 / @fastify/sensible@6.0.4 / @fastify/websocket@11.2.0 / pino@9.3.2 / zod@3.23.8；开发链：tsx / typescript@5.5 / eslint / prettier / vitest |
| `media-server/tsconfig.json` + `tsconfig.build.json` | - | TS 5.5 严格模式、ES2022 + Bundler 解析、dist 输出裁切 tests/poc |
| `media-server/.eslintrc.json` + `.prettierrc` | - | @typescript-eslint + prettier 协同，强制 `consistent-type-imports` |
| `media-server/.env.example` + `.gitignore` + `.dockerignore` | 60 行 | 双态部署必备变量（announcedIp / rtcPort 范围 / internalToken / logPretty） |
| `media-server/src/config.ts` | 110 行 | 自研 dotenv 加载 + zod 严格校验 + 端口区间交叉校验 + 失败直接 `process.exit(1)` |
| `media-server/src/utils/logger.ts` | 45 行 | pino + pino-pretty（dev）+ token redact + `childLogger` |
| `media-server/src/mediasoup/worker.ts` | 115 行 | Worker 单例 + `died` 指数退避（1s/2s/4s/8s/16s/30s 封顶）+ snapshot（pid / restartAttempts） |
| `media-server/src/middlewares/internal-auth.ts` | 55 行 | `fastify-plugin` 包装 onRequest hook + `timingSafeEqual` 防侧信道 + `/healthz` 和 `/readyz` 白名单 |
| `media-server/src/app.ts` | 125 行 | Fastify 入口 + `/healthz` + `/readyz`（503 when 未 ready）+ `/internal/info` + 优雅停机（SIGINT/SIGTERM + unhandledRejection/uncaughtException） |
| `media-server/Dockerfile` | 多阶段 | node:20-bookworm-slim；builder 装 python3 + build-essential 编译 mediasoup worker；runtime 裁 dev deps + 非 root 用户 + curl HEALTHCHECK + 显式暴露 40000-40199 UDP/TCP |
| `media-server/README.md` | 6 节 | 目录结构 / 快速开始 / 鉴权校验 / npm 脚本 / Docker 构建 / 配置说明 / Task 1 验收清单 |

### 验收实测（本机 macOS + Node 20，已清理 .env）

| 场景 | 命令 | 结果 |
|---|---|---|
| 依赖安装 | `npm install` | 315 包，3 分钟，mediasoup C++ worker 编译通过 |
| 类型校验 | `npm run typecheck` | 0 错误 |
| 代码规范 | `npm run lint` | 0 错误 |
| 启动验证 | `npm run dev` | 2s 内 "media-server listening" + Worker PID 打印 |
| 健康探测 | `curl /healthz` | `{"ok":true,"mediasoupVersion":"3.19.0","workerPid":10584,"workerRestartAttempts":0,"uptimeSec":16}` |
| 就绪探测 | `curl /readyz` | `{"ready":true}` |
| 无 Token 访问 | `curl /internal/info` | HTTP 401 + `{"code":"UNAUTHORIZED"}` |
| 错 Token 访问 | `curl -H "X-Internal-Token: wrong" ...` | HTTP 401 |
| 正确 Token 访问 | `curl -H "X-Internal-Token: <env>" /internal/info` | 200，返回 mediasoup 版本 / worker 状态 / listen 配置 |
| Worker 自愈 | `kill -9 <workerPid>` | 日志 "worker died" → 1s 后自动 "worker started"，新 PID 11141 在线，`/healthz` 恢复 `ok:true` |

### 关键工程决策

1. **Fastify 5 + 单 pino 实例**：升级至 Fastify 5 后使用原生 `loggerInstance: logger`，Fastify 请求日志与业务日志共享同一个 pino 实例，消除 v4 时代的两个独立实例；配合 Fastify 4 已出 LTS 的事实，回归官方维护版本
2. **mediasoup 3.19 类型导入**：统一从 `mediasoup/types` 子路径引入（`Worker` / `WorkerLogLevel`），与 3.14 PoC 阶段 `mediasoup.types` 命名空间兼容
3. **内部鉴权侧信道防护**：`timingSafeEqual` 替代 `===`，对长度不等也先拉齐再比，防止 token 长度被时序探测
4. **Worker 指数退避重启**：`1s → 2s → 4s → 8s → 16s → 30s`（封顶 30s），重启成功后 `restartAttempts` 归零；避免快速失败时 CPU 飙高
5. **Docker 运行时轻量化**：builder 阶段 `npm prune --omit=dev` 裁掉 dev 依赖，runtime 仅保留必要的 curl + libstdc++，非 root 用户运行

### 下一步

- **Task 2**：Router/Transport/Produce/Consume 核心内部 API（对齐 PoC 的 paused-then-resume + simulcast 三档 encodings），预计 1.5 人日

---

## 🚀 2026-04-21 Phase 2e-2 Task 0 mediasoup PoC Spike 完成

**交付**：`media-server/poc/` 跑通 2 浏览器 ↔ Node ↔ mediasoup 完整 SFU 链路，技术栈可用性验证通过，mediasoup 选型锁定。

### 产出文件
| 文件 | 规模 | 作用 |
|---|---|---|
| `media-server/poc/server.mjs` | 248 行 | Fastify + WS 信令 + mediasoup Worker/Router |
| `media-server/poc/public/client.mjs` | 215 行 | mediasoup-client Device/Transport/Producer/Consumer 完整流程 |
| `media-server/poc/public/index.html` | 101 行 | 极简 UI（视频网格 + 日志 + 加入/离开按钮） |
| `media-server/poc/package.json` | - | 固定版本依赖（mediasoup@3.14.11 / fastify@4.28.1 / pino@9.3.2） |
| `media-server/docs/poc-notes.md` | 9 章节 | 架构图 / 实测数据 / 7 项关键坑 / 技术栈判定 / 复用映射 / 启动步骤 |

### 实测数据（Playwright 双 tab 自动化）
- 2 人会议：4 transports / 4 producers / 4 consumers / RSS **61 MB**
- peer 关闭：consumers 自动从 4 → 0，无泄漏
- 8 人线性外推：16 transports / 16 producers / **112 consumers** / 约 200 MB RSS（Node 单进程承载充裕）
- Node 24.12.0 下 mediasoup C++ 编译 **53 秒**完成，无环境问题

### 锁定的 7 项关键坑
1. 本机 Demo `MEDIASOUP_ANNOUNCED_IP` 必须留空（非 `127.0.0.1`），让 Chromium 自动替换 `0.0.0.0`
2. mediasoup-client 无 UMD bundle，Task 9 正式前端需 `npm + vite` 打包（PoC 用 esm.sh CDN）
3. Consumer 必须 `paused:true` 创建 → 客户端 `transport.consume` → 服务端 `resumeConsumer`，否则首帧丢失
4. Worker `died` 事件必须监听 + 外部进程管理器重启
5. `enableUdp:true + enableTcp:true + preferUdp:true` 开箱即用，DTLS 无需额外配置
6. Simulcast 三档 encodings `{150k/400k/1M}` 与设计文档完全一致，Task 2/9 可直接复用
7. Playwright Chromium 默认带 `--use-fake-device-for-media-stream`，E2E 自动化无阻

### 决策结论
- **维持 mediasoup + fastify + mediasoup-client 选型**，不改用 livekit-server
- Task 1-2 可直接复用 PoC 的 Worker/Router 初始化、Transport 创建参数、Consumer paused-then-resume 模式
- Task 9 前端 Store 可复用 `SignalClient` 的 Promise 化 WS 请求 + reqId 配对模式

### 下一步
- **Task 1**：`media-server/` 正式骨架（TS + Fastify + 内部鉴权中间件 + Dockerfile），预计 0.5 人日

---

## ✅ 2026-04-20 Phase 2e-1 通知系统完成

**交付范围**：统一通知中心，覆盖好友 / 群聊 / 会议（预留枚举）/ 系统广播 四大类，共 11 种业务通知类型。

### 后端（`backend/go-service/`）
| 模块 | 路径 | 作用 |
|---|---|---|
| 数据表 | `deploy/docker/postgres/init.sql` | 新增 `notify_notifications` + 3 索引 |
| 常量 | `app/constants/notify.go` | 11 种 type 常量 + 4 种 category + WS 事件名 |
| Model | `app/notify/model/notification.go` | GORM 模型 |
| DTO | `app/dto/notify_dto.go` | 请求/响应/广播 DTO |
| DAO | `app/notify/dao/notification_dao.go` | CRUD + 批量 + 未读统计 + 清理 + 全量用户列表 |
| Service | `app/notify/service/{notify_service.go,pusher.go}` | 业务逻辑 + Pusher 接口 + 持久化+推送降级 |
| Controller | `app/notify/controller/notification_controller.go` | 4 用户接口 + 1 管理员广播接口 |
| Router | `app/notify/router.go` | `/api/v1/notifications/*` + `/api/v1/admin/notifications/broadcast` |
| Cleanup Task | `app/notify/task/cleanup_task.go` | 30 天已读通知定时清理（默认每日） |
| Provider | `app/notify/provider.go` + `app/provider/{provider,wire,wire_gen}.go` | Wire 注入 + NotifyPusher / NotifyConnectHook 接口绑定 |
| WS 钩子 | `app/ws/handler.go` | 连接建立 → 推送 `notify.unread.total` 补偿 |
| contact 集成 | `app/contact/service/contact_service.go` | 3 处 Pusher.Push：friend_request/accepted/rejected |
| group 集成 | `app/group/service/group_service.go` | 6 处 Pusher.Push：invite/join_request/approved/rejected/kicked/role_changed |
| main.go | `cmd/server/main.go` | 启动 CleanupTask + defer Stop |

### 前端（`frontend/src/`）
| 文件 | 作用 |
|---|---|
| `api/notify.js` | 4 个 REST 封装 |
| `constants/notify.js` | 前端 type / category / 图标 / 颜色常量 |
| `store/notify.js` | Pinia Store：5 分类分页缓存 + 未读数 + WS 事件（notify.new / notify.unread.total） |
| `components/notify/NotifyItem.vue` | 通用通知卡片（按 type 渲染 + 内联接受/拒绝） |
| `pages/notify/index.vue` | 通知中心主页（顶部铃铛 + 5 Tab + 下拉刷新 + 无限滚动 + 全部已读） |
| `pages/profile/index.vue` | 新增铃铛入口 + 徽标 + 菜单项 |
| `pages.json` | 注册 `pages/notify/index` |
| `App.vue` / `pages/auth/login.vue` | 全局初始化 notifyStore 监听 + 未读数拉取 |
| `store/user.js` | logout 时调用 `notifyStore.reset()` 清缓存 |
| `store/contact.js` / `store/group.js` | 清理散落 toast 和冗余 `notify.friend.request` / `group.join.request` 直接处理 |

### 架构决策
- **单端 WS 连接架构**：沿用现有 `ws.Hub`，**不做多端已读同步**。多端改造推迟到 Phase 2f/二期（已在设计文档 §3.1/§3.5/§九记录）。
- **跨模块依赖方向**：`contact` / `group` → `notify`（严格单向），通过接口 `notifyService.Pusher` 注入，类似 Phase 2a `ws.FriendIDsGetter → contact.FriendshipDAO` 模式。
- **降级策略**：Pusher 先入库、后推送；WS 推送失败不回滚入库；入库失败仅 Warn 日志不影响业务。
- **游标分页**：通知列表使用 `before_id` + `limit` 替代传统 `page`，天然抗数据插入扰动。
- **30 天清理**：默认每 24 小时扫描一次，删除 `is_read=true AND created_at < NOW() - INTERVAL 30 DAY`；未读永久保留。

### Task 完成情况（11/11）
- [x] Task 0: 数据库 DDL + constants + DTO + model
- [x] Task 1: Notify 模块骨架（DAO + Service + Pusher 接口 + Wire）
- [x] Task 2: REST API（4 用户 + 1 管理员）
- [x] Task 3: WS 事件 `notify.new` + `notify.unread.total`
- [x] Task 4: contact 集成（3 类）
- [x] Task 5: group 集成（6 类）
- [x] Task 6: 30 天清理定时任务
- [x] Task 7: 前端 Pinia Store + API + WS 监听
- [x] Task 8: 通知中心页 + NotifyItem + 5 分类 Tab
- [x] Task 9: profile 入口 + 冗余清理（contact.js:155 / group.js:292）
- [x] Task 10: E2E 验证清单（`test-report-phase2e-1-notification.md`）
- [x] Task 11: 文档同步 + `code-reviewer` 子代理审查（本文件 + API 文档 + project-context.mdc + 设计文档 §3.1/§3.5/§八/§九 修订）

### Task 11 代码审查成果（2026-04-20）
- `code-reviewer` 整体结论：**有条件通过**（1 Blocker / 5 Major / 11 Minor / 10 亮点）
- **Blocker 已当场修复**：前端 `markAllRead` 契约错位 —— 原将 `category` 放入 PUT body 而后端读 query，已改为 `?category=xxx` Query 拼接，并同步 `docs/api/frontend/notify.md` §4
- **Major / Minor 项**：均不阻塞合入，已纳入 `docs/plans/2026-04-20-phase2e-design.md` §九 Phase 2f 清理清单
- 涉及修复文件：`frontend/src/api/notify.js`、`docs/api/frontend/notify.md`、`test-report-phase2e-1-notification.md` §七
- 验证：`cd backend/go-service && go build ./...` 通过

### Playwright MCP 端到端验证成果（2026-04-20）
- 使用 Playwright MCP 驱动 H5 浏览器完整走查通知中心：登录 → 进入 `/pages/notify/index` → 构造好友申请 → 管理员广播 → 点击通知 → 标记已读 → 跳转 → 全部已读，均通过
- **WS 实时推送链路通过**：admin 广播后 1s 内页面自动插入通知到列表顶部，「全部/系统」角标实时变化，证明 `notify.new` 事件、前端 `_onNotifyNew` 处理器、UI 响应式更新三段链路一致
- **额外修复 2 个交互 Bug**（Playwright 现场发现）：
  - 🔴 Bug-1：`NotifyItem` 自定义 emit 事件名 `tap` 与 uni-app 原生 DOM 事件冲突，导致 Event 对象覆盖 notify 参数 → `PUT .../undefined/read` 400。修复：emit 名改为 `item-tap` / `item-accept` / `item-reject`
  - 🟡 Bug-2：`store/notify.js#markRead` 在 `_patchAll` 后再判断 `!target.is_read` 永假，unreadTotal 没有递减。修复：预先快照 `wasUnread`
- 修改文件：`frontend/src/components/notify/NotifyItem.vue`、`frontend/src/pages/notify/index.vue`、`frontend/src/store/notify.js`
- 详细过程见 `test-report-phase2e-1-notification.md` §八

### TabBar「我的」聚合未读红点（2026-04-21）
- **背景**：通知未读状态之前仅在 `/pages/profile/index` 内可见（铃铛 badge、菜单项 badge），用户在其他 tabBar 页面（消息/联系人/会议）无法感知有新事件，必须被动切进"我的"才能发现
- **设计**：采用业界主流做法（微信/QQ/钉钉的"我"Tab 模式）—— tabBar「我的」图标右上角显示**纯红点（无数字）**作为"我的"模块**聚合未读指示器**
  - 信息层级分离：tabBar 一级导航只承载 Boolean（有/无未读），具体数字留在二级页面
  - 聚合开放集合：当前只聚合 `notifyStore.unreadTotal`，未来可无缝追加「资料待完善」「安全提醒」「新版本可用」等
  - 语义清晰：保留原 `getBadge(index)`（数字，用于消息/联系人）；新增 `hasDot(index)`（布尔，用于"我的"）；模板先数字后红点优先级渲染
- **实现文件**：`frontend/src/components/CustomTabBar.vue`（新增 `hasDot` 方法 + `.tab-dot` 样式）
- **Playwright 回归**：有 3 条未读时消息页/我的页 tabBar 红点均亮起；点"全部已读"后 tabBar 红点消失、铃铛 badge 消失、菜单 badge 消失三层同步响应；所有场景均通过
- **详细设计**：见 `docs/plans/2026-04-20-phase2e-1-design.md` §6.4

---


---

## 📘 2026-04-20 Phase 2e 规划：会议与通知系统

**拆分理由**：原 Phase 2e（会议+通知）工作量约 23-33 人日，引入全新技术栈（mediasoup + Node.js），必须按风险梯度拆分。

| 子阶段 | 范围 | 周期 | 状态 |
|---|---|---|---|
| **2e-1 通知系统** | 统一通知中心（11 种类型）+ 双通道推送 + 跨模块 Pusher 接口 | 3-4 天 | ✅ 已完成 |
| **2e-2 会议 MVP** | mediasoup Node 媒体服务 + 即时会议（≤8人）+ 基础音视频控制 | 10-14 天 | 📋 待开发 |
| **2e-3 会议增强** | 预约会议 + 定时提醒 + 会议邀请（复用 2e-1 通知） | 7-10 天 | 📋 待开发 |

**关键决策**：
- 维持原 mediasoup SFU 架构（不改用 Mesh）
- 会议 MVP 仅音视频通话（不含录制/屏幕共享/预约）
- 管理端扩展推迟到 Phase 2f（新增阶段）
- 通知系统覆盖全部 10+ 种业务事件，含 `meeting_invite`/`meeting_reminder` 类型预留

**后续规划清单**（含推迟项，见设计文档 §九）：
- Phase 2f：会议管理后台 + 通知广播发布 UI + 管理端仪表板等
- 第二期：屏幕共享、录制、虚拟背景、微信登录、互动直播
- 第三期：微服务拆分、K8s、多 Worker 集群、AI 辅助

---

## 🐛 2026-04-20 Bug 修复：已读状态刷新后丢失（变回"未读"）

**问题**：单聊/群聊页面中大量消息显示"已读"的情况下，刷新浏览器 → 所有已读标签全部变为"未读" / "N人已读"消失，直到对方再次触发读消息才恢复 —— 严重影响真实性与信任度。

**根因（单句）**：前端 `readStatusMap`（对方已读位置）和 `groupReadCountMap`（群聊已读计数）仅由 WebSocket 事件 `im.message.read.ack / im.message.read.count` 填充，从未在页面加载时从后端拉取，刷新后 state 归零且无 API 补回 → `isRead()` 判定 `msg.id <= 0` 恒为 false。后端数据库 `im_conversation_members.last_read_msg_id` 与 `im_message_reads` 数据其实都存在，只是未暴露给前端初始化。

**修复方案（前后端联动，不新增 API，仅扩展 `HistoryMessageResponse`）**：

| 端 | 改动 |
|---|---|
| 后端 DTO | `HistoryMessageResponse` 新增 `peer_last_read_msg_id int64`（单聊）、`read_count_map map[int64]int`（群聊 omitempty） |
| 后端 Service | `GetHistoryMessages` 按会话类型分支：type=1 → `GetPeerUserID` + `GetMember` 拿对方 `last_read_msg_id`；type=2 → 过滤自己发送的消息 ID，调 `readRecorder.GetReadCountBatch` |
| 前端 Store | `loadHistoryMessages` 仅在「首次加载（messages.length===0）」时回填，避免后续"加载更多"时用历史数据覆盖 WS 增量的最新态 |

**涉及文件**：
- `backend/go-service/app/dto/im_dto.go`：DTO 扩展
- `backend/go-service/app/im/service/im_service.go`：GetHistoryMessages 回填逻辑
- `frontend/src/store/chat.js`：loadHistoryMessages 消费新字段

**Playwright 端到端验证**：
1. 数据库事实：会话 6 中 `bojinyuan(7)` last_read_msg_id=202（已读到 id=202）
2. duanlingyun 登录 → API 返回 `peer_last_read_msg_id=202` ✓
3. 会话页首屏：21 条自己发送的消息 → 20 条"已读" + 1 条新插入的 id=203"未读" ✓（与期望完全吻合）
4. 群聊验证：bojinyuan 登录 → 群 8 API 返回 `read_count_map={94:1, 95:1, ..., 139:2, 140:2}`（共 13 条自己发的消息）→ 页面正确显示 11 个"1人已读" + 2 个"2人已读" ✓
5. 视觉回归：同一截图中语音"未听红点"仍正常工作（表明两项 UX 优化无互相干扰）

**设计权衡**：
- 为什么不新增独立 API：避免每次进入会话多一次往返请求
- 为什么仅首次加载回填：后续 WS 事件已能增量更新，用历史数据覆盖反而可能倒退
- 为什么群聊只查"自己发的消息"：前端仅在 `isSelf=true` 时展示"N人已读"，避免无用 DB 查询

---

## 🎙️ 2026-04-20 UX 优化：语音消息「未听红点」（仿微信）

**需求**：通用"已读"状态基于滚动可见性，适合文字/图片（肉眼可直接阅读）；但**语音必须点击播放才算"听过"**，仅"已读"不能反映用户是否真的听过。参照微信设计，为「对方发来的语音」叠加独立的"未播放"视觉提示（红点）。

**设计决策（均为推荐方案）**：
| 维度 | 取舍 |
|---|---|
| 作用范围 | 仅「对方发来的」语音显示红点（自己发的无此语义） |
| 持久化 | 本地 localStorage（按 userId 隔离） —— 不同步后端，属于私人视图态 |
| 触发时机 | 用户点击播放即刻清除红点（不要求播放完整） |
| 作用层级 | 仅聊天详情页；会话列表不处理 |

**核心实现**：
- `chat store` 新增 `voicePlayedMap: {[msgId]: true}` + `markVoicePlayed / isVoicePlayed / loadVoicePlayedState / resetVoicePlayedState` 4 个 API
- localStorage key：`echo:voice-played:{userId}`，完整 JSON 覆盖写入
- `MsgVoice.vue` 自主消费 store（父级无需改动，单聊/群聊同构生效）
  - 模板：`<view v-if="showUnplayedDot" class="unplayed-dot" />`（10rpx 红圆，紧贴气泡右侧）
  - `showUnplayedDot = computed(() => !isSelf && msg.id && !chatStore.isVoicePlayed(msg.id))`
  - `onTogglePlay` 首行：`if (!isSelf && msg.id) chatStore.markVoicePlayed(msg.id)`
- `user.store.logout()`：动态导入 chat store 调用 `resetVoicePlayedState()` 防止串用户
- `chat.store.initWsListeners()`：按当前 `userStore.userInfo.id` 懒加载对应缓存

**涉及文件**：
- `frontend/src/store/chat.js`：新增 voicePlayed 状态 + 4 个方法
- `frontend/src/components/msg/MsgVoice.vue`：模板加红点 + 消费 store
- `frontend/src/store/user.js`：logout 清理运行时 state

**Playwright 验证**：
1. duanlingyun(id=13) 登录 → 会话 6（内含自己 3 条语音 + 对方 5 条语音）
2. 初始：8 条语音 → 5 个红点（全部对方发） + 0 个红点（自己 3 条 `voice-self`）✓
3. 点击任意两条对方语音 → 红点数 5→4→3 ✓
4. localStorage 写入 `echo:voice-played:13 = {195:true, 196:true}` ✓
5. 页面刷新 → 仍只剩 3 个红点（持久化生效）✓

---

## 💡 2026-04-20 UX 优化：聊天页「新消息悬浮提示 + 按需已读」

**问题**：用户停留在聊天页旧消息位置时，对方发来新消息 → 页面不会自动滚动到底 → 但 store 层一律自动 `markRead` → 对方看到"已读"但用户实际没看到消息，产生体验错位。

**方案（混合策略，业内通用）**：
- 贴底（距底 < 150px）时：新消息自动滚动到底部 + 标记已读（保留原有体验）
- 远离底部时：不滚动、不标记已读，右下角显示「↓ N 条新消息」悬浮胶囊按钮
- 点击悬浮按钮：滚到底 + 清零计数 + 标记已读
- 用户手动滚回底部：自动隐藏悬浮 + 标记已读

**技术实现（`maxScrollTop` 追踪策略）**：
- `@scroll` 事件中维护"历史最大 scrollTop"，`maxScrollTop - scrollTop < 150` 判断是否贴底
- 不依赖 `@scrolltolower` 作为唯一权威信号，对 `scroll-into-view` + 动态 scrollHeight 都正确响应
- Store 层移除 `_onNewMessage` 中对当前会话的自动 `markRead`，把决策权交给页面

**涉及文件**：
- `frontend/src/store/chat.js`：移除新消息到达时的自动 `markRead`
- `frontend/src/pages/chat/conversation.vue`：单聊滚动感知 + 悬浮按钮
- `frontend/src/pages/group/conversation.vue`：群聊同构改造

**验证结果**：Playwright 全流程通过三种场景（贴底自动滚、远离底部显示悬浮、点击悬浮滚到底清零）。

### 🐛 UX 优化后续修复（2026-04-20）：watch 逻辑误把"加载历史消息"计入 newMsgCount

**问题**：用户在页面中段滚动时触发「加载更多」→ 从数组**头部**插入 20 条历史消息 → `messages.value.length` 变化 → watch 误以为是"对方的新消息"并 `newMsgCount += 20`，悬浮按钮显示"40+ 条新消息"。

**根因**：watch 回调只看 `messages.length` 增量（delta），未区分"头部插入历史"与"尾部追加新消息"。读取 `messages[length-1]` 拿到的是已存在的最后一条消息，若恰好是对方发的即通过 fromOther 判断并累加。

**修复策略**：监听「末尾消息标识（id / client_msg_id）」是否变化，而非仅看长度：
- 头部插入历史消息 → tail 未变 → 直接返回，不做任何处理
- 尾部追加一条新消息 → tail 变化 → 才进入"fromSelf/fromOther/nearBottom"判断分支
- 自己发的消息 tail 也会变但会被 `fromSelf` 拦下

**真实双账号验证（duanlingyun ↔ bojinyuan，会话 id=6）**：

| 操作 | messages.length | scrollTop | scrollHeight | 悬浮按钮 |
|---|---|---|---|---|
| 初始贴底 | 20 | 1419 | 2066 | 无 |
| 滚至顶部触发加载更多 ×3 | 76 | 0 | 4823 | **无** ✅ |
| 对方发 1 条 | 77 | 1003 | 4877 | "1 条新消息" ✅ |
| 对方再发 2 条 | 79 | 1003 | 4877 | "3 条新消息" ✅ |
| 再次加载更多历史 | 79+10 | 0 | 4986 | **仍为 "3 条新消息"** ✅ |
| 点击悬浮 | 79+10 | 4339 (贴底) | 4986 | 消失 ✅ |

**涉及文件**：
- `frontend/src/pages/chat/conversation.vue`：`lastMsgCount` → `lastTailKey` 改造
- `frontend/src/pages/group/conversation.vue`：相同改造（群聊）

### 🐛 语音消息 H5 兼容修复（2026-04-20）

**问题链**（按排查顺序）：
1. **事件层**：PC Chrome 按住"按住说话"按钮无反应 —— 只绑定了 `@touchstart/move/end`，现代浏览器鼠标不合成 touch 事件
2. **录音 API 层**：修复事件绑定后报 `method 'uni.getRecorderManager' not supported` —— uni-app H5 端不支持原生 `getRecorderManager`
3. **上传层**：即便录到 Blob，`uni.uploadFile(blob:URL)` 推断出的 filename 无扩展名，被后端白名单拦下
4. **后端校验层**：`allowedVoiceExts` 只允许 `.mp3/.wav/.aac/.m4a`，不接受 H5 MediaRecorder 输出的 webm/ogg

**修复链**：
| 层 | 改动 |
|---|---|
| 事件 | VoiceRecorder 同时监听 touch + mouse 事件，`pressing` 重入守卫，`@mouseleave` 兜底取消 |
| 录音 | 新增 `H5Recorder` 类，基于 `MediaRecorder` + `getUserMedia` 实现 uni 录音接口（`onStart/onStop/onError/start/stop`），`createRecorder()` 工厂按平台自动选择 |
| 上传 | `uploadVoice` 增加 blob 参数分支，H5 走 `fetch` + `FormData` 精确控制 filename（基于 mimeType 推断扩展名，如 `voice-{ts}.webm`）|
| 后端 | `allowedVoiceExts` 新增 `.webm`、`.ogg`，错误消息同步更新 |

**验证结果（Playwright 自动化）**：

| 阶段 | class | 按钮文字 | 遮罩 | 计时 |
|---|---|---|---|---|
| 按压中 (4s) | `record-btn recording` | "松开发送" | ✓ 显示 | "3\"" |
| 按压中 (5.5s) | 同上 | 同上 | ✓ | "5\"" （跳动正常）|
| 松开后 | `record-btn` | "按住 说话" | 消失 | — |

数据库验证：连续 3 次录音上传成功（1/2/5 秒，size 18K/35K/75K），`im_messages` type=3（语音）记录正确写入，MinIO 中 `.webm` 文件可访问。

**涉及文件**：
- `frontend/src/components/chat/VoiceRecorder.vue`：核心改造（事件兼容 + H5Recorder 类）
- `frontend/src/api/file.js`：`uploadVoice` 增加 blob 分支，新增 `_doUploadBlob` 私有方法
- `frontend/src/pages/chat/conversation.vue` & `frontend/src/pages/group/conversation.vue`：`onVoiceRecorded` 按 mimeType 生成 fileName 并传 blob
- `backend/go-service/app/file/service/file_service.go`：`allowedVoiceExts` 扩展 + 错误消息更新

---

## 🔧 2026-04-20 Bug 修复清单

| # | 根因 | 修复 | 文件 |
|---|---|---|---|
| B1 | 文件上传 BASE_URL 硬编码为 `:8080`（与实际 `:8085` 不符）导致 `ERR_CONNECTION_REFUSED` | 改为复用 `@/utils/request` 中的统一 BASE_URL | `frontend/src/api/file.js` |
| B2 | MinIO bucket 默认 private，图片 URL 匿名访问返回 403 | 启动时自动设置 bucket public-read 策略（仅 `s3:GetObject`） | `backend/go-service/pkg/storage/minio.go` |
| B3 | MsgImage flex 布局塌陷，`<uni-image>` 宽度为 0 | `.grid-single/2col/3col` 增加显式宽高和 `display:block` | `frontend/src/components/msg/MsgImage.vue` |
| B4 | H5 下 `URL.createObjectURL(file)` 丢失文件名，显示为 `file-1776xxx` | 前端发送消息时优先使用用户选择的 `file.name` | `frontend/src/pages/{chat,group}/conversation.vue` |

**验证结果**：Playwright MCP 全流程通过（图片/文件上传、缩略图渲染、大图预览、管理端消息详情）。详见 `test-report-phase2d-bugfix.md`。

---

## 🛠️ 2026-04-20 工程化脚本改造（开发者体验）

引入社区通用的 **setup / start 职责分离** 模式（参考 Rails `bin/setup` + `bin/dev`、Django 模板、Next.js 企业模板）：

| 脚本 | 定位 | 使用频率 |
|---|---|---|
| `scripts/dev-setup.sh` | **首次环境初始化**：Docker 检查 + 容器拉起 + 重试式健康检查（pg/redis/minio）| 首次 clone / 重建卷 |
| `scripts/start.sh` | **日常启动**：Docker 中间件 + Go 后端 + 前台 + 管理端全量拉起，端口占用自动跳过 | 每天多次 |
| `scripts/stop.sh` | **日常停止**：优雅终止应用层（先 TERM 后 KILL 兜底），默认保留容器，支持 `--all` 全关 | 每天多次 |
| `scripts/status.sh` | **状态查看**：三应用端口 + 三容器状态一览 | 排障随时 |

核心特性：
- **端口占用自动跳过**：`lsof` 检测，已占用则 WARN 跳过，不重复启动
- **PID + 日志分离**：PID 写入 `.run/*.pid`，stdout 写入 `.run/logs/*.log`（已加入 `.gitignore`）
- **MinIO 健康等待**：`dev-setup.sh` 新增 `/minio/health/live` 重试检查，补齐原脚本遗漏
- **单项启停**：`./scripts/start.sh backend|frontend|admin|docker`

涉及文件：
- 新增 `scripts/start.sh`、`scripts/stop.sh`、`scripts/status.sh`
- 改造 `scripts/dev-setup.sh`（新增 MinIO 检查 + 头部使用时机注释）
- 更新 `README.md`（置顶一键脚本章节，阐述首次/日常使用顺序）
- 更新 `.gitignore`（忽略 `.run/`）

---

## 一、Phase 2b Task 完成状态

| Task | 描述 | 状态 | 备注 |
|------|------|------|------|
| Task 0 | IM Model + 数据库迁移 + 常量 | ✅ 完成 | 3 张表 + init.sql + AutoMigrate |
| Task 1 | WS 事件路由表机制 | ✅ 完成 | Hub.RegisterEvent/DispatchEvent |
| Task 2 | IM DAO 层 | ✅ 完成 | ConversationDAO + MessageDAO |
| Task 3 | IM Service 核心业务 + DTO | ✅ 完成 | 9 个业务方法 + 接口注入 |
| Task 4 | WS 事件处理器 + 离线推送 | ✅ 完成 | 4 个事件 + OfflinePusher |
| Task 5 | REST Controller + Router + Wire | ✅ 完成 | 7 个 REST API + 完整 Wire 集成 |
| Task 6 | 前台 Store + API + WS 事件 | ✅ 完成 | chat.js Store + API + TabBar badge |
| Task 7 | 会话列表页 + 聊天对话页 | ✅ 完成 | 2 个核心页面 |
| Task 8 | 设置页 + 搜索页 + 联系人改造 | ✅ 完成 | 2 个辅助页面 + 发消息跳转 |
| Task 9 | 文档更新 + 代码审查 | ✅ 完成 | 进度/架构文档同步 |
| UI 改造 | ui-ux-pro-max 规范改造 | ✅ 完成 | uni-icons 替换 emoji + 设计规范文件 |
| 代码审查修复 | 后端 7 项修复 | ✅ 完成 | P0×2 + P1×3 + 推送补全×2 |
| 用户测试修复 | 8 项 Bug 修复 | ✅ 完成 | 好友申请/接受、在线状态、UI 布局等 |

### 代码审查修复详情

| # | 优先级 | 修复内容 |
|---|--------|----------|
| Fix 1 | P0 | ClearHistory 改为个人视图操作（ClearBeforeMsgID），不再删除双方消息 |
| Fix 2 | P0 | Redis 未读数负数保护（Lua 脚本原子递减，下限为 0） |
| Fix 3 | P1 | GetConversationList N+1 查询优化（LEFT JOIN 一次获取 peerID） |
| Fix 4 | P1 | 消息搜索改用 GIN 全文索引（to_tsvector/plainto_tsquery 替代 LIKE） |
| Fix 5 | P1 | 撤回消息后更新会话预览（last_msg_content = "XX 撤回了一条消息"） |
| Fix 6 | - | im.message.new 推送补充 sender_name、sender_avatar |
| Fix 7 | - | im.message.recalled 推送补充 sender_id |

### 用户测试修复详情

| # | 修复内容 |
|---|----------|
| Fix 8 | 好友申请拒绝后重新申请失败 — FriendshipDAO 新增 ReactivateRejectedRequest 方法 |
| Fix 9 | 好友接受申请失败（反向记录 UNIQUE 冲突）— AcceptRequest 先查后改，避免重复插入 |
| Fix 10 | Redis 在线状态残留 — OnlineService 启动时清理旧在线数据（cleanStaleOnlineData） |
| Fix 11 | WS 断开时在线状态未清理 — 修正 onDisconnect 判断条件（closedByHub && isOnline） |
| Fix 12 | 前端 WS 连接未全局初始化 — App.vue onLaunch/onShow + login.vue 登录后建立连接 |
| Fix 13 | 后台管理端好友关系页用户 A 列名称错误 — 修正字段绑定 row.user_username |
| Fix 14 | 前台好友在线状态初始值缺失 — ContactService 注入 OnlineChecker，GetFriendList 返回 is_online |
| Fix 15 | 聊天页消息过多时输入框被挤出 — scroll-view 添加 height:0 + min-height:0 约束 |

---

## 二、Phase 2b 新增功能

### 即时通讯（IM）
- **消息收发**：WebSocket 全双工通讯，im.message.send → ACK + 推送
- **三态确认**：sending → sent/ACK → failed
- **消息撤回**：2 分钟内可撤回，推送 im.message.recalled
- **正在输入**：im.typing 事件，3 秒超时自动清除
- **离线消息**：WebSocket 重连后服务端主动推送未读会话摘要

### 会话管理
- **自动创建**：首次发消息时自动创建单聊会话
- **会话列表**：置顶优先 → 最后消息时间降序，LEFT JOIN 一次获取 peerID（N+1 优化）
- **会话操作**：置顶/取消、软删除（不影响对方）、清空聊天记录（个人视图 ClearBeforeMsgID）
- **未读管理**：DB unread_count + Redis STRING 全局未读数（Lua 脚本负数保护），TabBar badge 显示

### WebSocket 事件路由表
- **Hub.RegisterEvent**：业务模块注册事件处理器
- **Hub.DispatchEvent**：消息分发到匹配的处理器
- **事件清单**：im.message.send / im.message.recall / im.conversation.read / im.typing

### REST API（7 个）
| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /api/v1/im/conversations | 会话列表 |
| GET | /api/v1/im/messages | 历史消息（游标分页） |
| PUT | /api/v1/im/conversations/:id/pin | 置顶/取消 |
| DELETE | /api/v1/im/conversations/:id | 删除会话 |
| DELETE | /api/v1/im/conversations/:id/messages | 清空记录 |
| GET | /api/v1/im/messages/search | 全局搜索 |
| GET | /api/v1/im/unread | 全局未读数 |

### 前端页面（4 个）
- `pages/chat/index.vue` — 会话列表（TabBar 页面）
- `pages/chat/conversation.vue` — 聊天对话页
- `pages/chat/settings.vue` — 聊天设置页
- `pages/chat/search.vue` — 消息搜索页

### 数据库表（3 张）
- `im_conversations` — 会话表（含冗余 last_msg_* 字段）
- `im_conversation_members` — 会话成员表（个人视图：置顶/未读/软删除）
- `im_messages` — 消息表（游标分页索引 + GIN 全文搜索索引）

---

## 三、Phase 2a 完成总结

| Task | 描述 | 状态 | 备注 |
|------|------|------|------|
| Task 0-12 | WebSocket + 联系人 + 管理端 | ✅ 全部完成 | 13 个 Task + 8 项 Bug 修复 |

- WebSocket 实时通讯（Hub + Client + PubSub）
- 联系人管理 17 个 API
- 在线状态管理（Redis SET + TTL）
- 管理端扩展（在线监控 + 好友管理）

---

## 四、Phase 1 完成总结

| Task | 描述 | 状态 |
|------|------|------|
| Task 1-11 | 基础设施 + 认证 + 用户管理 | ✅ 全部完成 |

- Go 后端 15+ API、JWT 有状态认证、RBAC 角色权限
- 前台 uni-app 登录/注册/TabBar/个人中心
- 管理端 Vue 3 登录/仪表盘/用户列表/详情
- Docker Compose 一键启动

---

## 五、关键技术决策记录

### 后端（Go）
1. **框架组合**：Gin + GORM + Wire + Zap + Viper
2. **JWT 策略**：有状态 JWT，Token 按 clientType 隔离存储在 Redis
3. **WebSocket**：`gorilla/websocket` + Redis Pub/Sub 跨实例路由
4. **WS 事件路由**：Hub.eventHandlers map[string]EventHandler + RegisterEvent/DispatchEvent
5. **IM 跨模块**：FriendChecker + UserInfoGetter 接口注入（contact → im）
6. **IM 推送**：OfflineMessagePusher 接口注入（im → ws）
7. **在线状态**：混合方案（Redis SET + STRING TTL + Pub/Sub 推送）
8. **角色等级**：`auth_roles.level`（1=超管, 10=管理员, 100=普通用户）

### 前台用户端（frontend/）
1. **框架**：uni-app 3.0（Vue 3.4.21）
2. **状态管理**：Pinia 2.1.7 + pinia-plugin-persistedstate@3
3. **WebSocket**：`uni.connectSocket`（小程序）/ `WebSocket`（H5）
4. **IM Store**：chat.js（会话列表 + 消息缓存 + 三态确认 + 全局未读）
5. **设计系统**：ui-ux-pro-max 规范，MASTER.md + 页面覆盖规范
6. **图标方案**：@dcloudio/uni-ui uni-icons（easycom 自动引入，跨平台兼容）
7. **预处理器**：sass（uni-icons SCSS 依赖）

### 后台管理端（admin/）
1. **框架**：Vue 3.5+ + Vite 7.x + Element Plus
2. **HTTP 客户端**：Axios
3. **存储隔离**：localStorage key 前缀 `admin_`

---

## 六、目录结构概览

```
EchoChat/
├── backend/go-service/
│   ├── app/
│   │   ├── admin/               # 管理端
│   │   ├── auth/                # 认证模块
│   │   ├── contact/             # [Phase 2a] 联系人模块
│   │   ├── im/                  # [Phase 2b] 即时通讯模块
│   │   │   ├── controller/      # REST API 控制器
│   │   │   ├── dao/             # 数据访问（ConversationDAO + MessageDAO）
│   │   │   ├── handler/         # WS 事件处理器 + 离线推送
│   │   │   ├── model/           # 数据库模型
│   │   │   ├── service/         # 核心业务 + 接口定义
│   │   │   ├── router.go
│   │   │   └── provider.go
│   │   ├── ws/                  # [Phase 2a] WebSocket 模块
│   │   ├── constants/           # 含 im.go 常量
│   │   ├── dto/                 # 含 im_dto.go
│   │   └── provider/
│   ├── pkg/
│   │   ├── ws/                  # WS 核心（Hub 含事件路由表）
│   │   ├── db/ logs/ middleware/ utils/
│   └── router/router.go
├── frontend/                    # 前台（uni-app）
│   └── src/
│       ├── api/{auth,contact,user,im,group,file}.js
│       ├── constants/group.js    # [Phase 2c] 群聊角色/状态常量
│       ├── services/websocket.js
│       ├── store/{user,websocket,contact,chat,group}.js
│       ├── pages/chat/          # [Phase 2b] 5 个页面
│       │   ├── index.vue        # 会话列表（含群聊 Tab）
│       │   ├── conversation.vue # 单聊对话
│       │   ├── read-detail.vue  # [Phase 2c] 已读详情
│       │   ├── settings.vue     # 聊天设置
│       │   └── search.vue       # 消息搜索
│       ├── pages/group/         # [Phase 2c] 7 个页面
│       │   ├── conversation.vue # 群聊对话（含 @选择器 + 已读计数 + 禁言提示）
│       │   ├── create.vue       # 创建群聊
│       │   ├── settings.vue     # 群设置
│       │   ├── members.vue      # 成员管理
│       │   ├── invite.vue       # 邀请入群
│       │   ├── join-requests.vue # 入群审批
│       │   └── search.vue       # 搜索群聊
│       ├── pages/contact/       # [Phase 2a] 6 个页面
│       ├── components/msg/     # [Phase 2d] 消息类型组件
│       │   ├── MsgText.vue     # 文本消息
│       │   ├── MsgImage.vue    # 图片消息（网格+预览）
│       │   ├── MsgVoice.vue    # 语音消息（播放+波形）
│       │   └── MsgFile.vue     # 文件消息（卡片+下载）
│       ├── components/chat/    # [Phase 2d] 聊天辅助组件
│       │   ├── MorePanel.vue   # "+"展开面板
│       │   └── VoiceRecorder.vue # 语音录制
│       └── components/CustomTabBar.vue（含 badge）
├── admin/                       # 管理端（Vue 3 + Element Plus）
│   └── src/views/
│       ├── message/            # [Phase 2d] 消息管理
│       │   ├── list.vue        # 消息列表（多条件筛选+操作）
│       │   └── stats.vue       # 消息统计（ECharts 仪表板）
├── deploy/
├── design-system/
└── docs/
    ├── api/
    ├── plans/
    ├── progress/CURRENT_STATUS.md
    └── conventions/
```

---

## 七、开发测试指南

### 服务管理命令

> 建议开 4 个终端窗口，分别运行各服务。终止方式统一为 `Ctrl + C` 或 `kill` 命令。

#### 基础设施（Docker Compose）

```bash
# 一键启动全部基础设施（PostgreSQL + Redis + MinIO）
cd deploy && docker compose -f docker-compose.dev.yml up -d postgres redis minio

# 查看容器状态
cd deploy && docker compose -f docker-compose.dev.yml ps

# 一键停止全部基础设施
cd deploy && docker compose -f docker-compose.dev.yml stop
```

| 服务 | 地址 | 单独启动 | 单独停止 |
|------|------|----------|----------|
| PostgreSQL | localhost:5432 | `docker compose -f docker-compose.dev.yml up -d postgres` | `docker compose -f docker-compose.dev.yml stop postgres` |
| Redis | localhost:6379 | `docker compose -f docker-compose.dev.yml up -d redis` | `docker compose -f docker-compose.dev.yml stop redis` |
| MinIO API | localhost:9000 | `docker compose -f docker-compose.dev.yml up -d minio` | `docker compose -f docker-compose.dev.yml stop minio` |
| MinIO 控制台 | localhost:9001 | （同上） | （同上） |

#### 应用服务

| 服务 | 地址 | 启动命令 | 终止命令 |
|------|------|----------|----------|
| Go 后端 | http://localhost:8085 | `cd backend/go-service && go run cmd/server/main.go` | `Ctrl+C` 或 `kill $(lsof -ti :8085)` |
| 前台用户端 (H5) | http://localhost:5173 | `cd frontend && npm run dev:h5` | `Ctrl+C` 或 `kill $(lsof -ti :5173)` |
| 后台管理端 | http://localhost:3100 | `cd admin && npm run dev` | `Ctrl+C` 或 `kill $(lsof -ti :3100)` |

#### Go 后端快速重启（一行命令）

```bash
kill $(lsof -ti :8085) 2>/dev/null; sleep 2; cd backend/go-service && go run cmd/server/main.go
```

### 测试账号

| 账号 | 密码 | 角色 | 用途 |
|------|------|------|------|
| `super_admin` | `admin123456` | super_admin | 系统预置唯一超管 |
| `admin_test` | `admin123456` | user + admin | 管理端登录推荐 |
| `testuser1` | `test123456` | user + admin | 前台登录测试 |
| `testuser` | `test123456` | user | 前台登录测试 |

### Phase 2b 可测试功能

- **会话列表**：发消息自动创建会话 → 列表排序 → 置顶 → 长按删除
- **聊天**：发送文本 → 三态确认 → 撤回（2分钟内）→ 正在输入提示
- **离线消息**：断开重连 → 自动推送未读摘要 → TabBar badge 更新
- **消息搜索**：全局关键词搜索 → 跳转到对应会话
- **联系人入口**：好友详情页 → 发消息 → 跳转聊天页

---

## 八、Phase 2c — 群聊与已读回执

> **状态：** ✅ 已完成
> **设计文档：** `docs/plans/2026-03-04-phase2c-design.md`
> **实施计划：** `docs/plans/2026-03-04-phase2c-implementation.plan.md`
> **分支：** `feature/phase2c-group-read-receipt`

### 功能范围

| 模块 | 内容 |
|------|------|
| 群聊管理 | 建群/加入/退出/解散/搜索/三级角色/禁言/全体禁言/群公告/群昵称/免打扰 |
| 群消息 | 复用 im.message.* 事件 + @某人/@所有人 + 管理员撤回（无时限）+ 系统消息 |
| 已读回执 | 单聊会话级（last_read_msg_id）+ 群聊消息级（im_message_reads 表）+ 实时推送 |
| MinIO | Docker 容器 + Go SDK + 通用上传 API（群头像） |
| 管理端 | 群列表/群详情/解散群 |
| 前端 | 9 个新页面 + 群聊 Store + 会话列表 Tab 改造 |

### Task 完成状态

| Task | 描述 | 状态 |
|------|------|------|
| Task 0 | MinIO Docker + SDK + 通用上传 API | ✅ 完成 |
| Task 1 | 数据库迁移 + Model + 常量 | ✅ 完成 |
| Task 2 | Group DAO 层 | ✅ 完成 |
| Task 3 | Group Service 业务逻辑 + WS 推送 | ✅ 完成 |
| Task 4 | Group Controller + Router + Wire | ✅ 完成 |
| Task 5 | IM Service 扩展（群消息/@提醒/管理员撤回） | ✅ 完成 |
| Task 6 | 已读回执后端（单聊 + 群聊） | ✅ 完成 |
| Task 7 | 代码审查修复（Critical 4 项 + Important 4 项） | ✅ 完成 |
| Task 8 | 前端已读回执 UI（单聊标记 + 群聊计数 + 详情页） | ✅ 完成 |
| Task 9 | 前端群聊 Store + API 封装 + WS 事件监听 | ✅ 完成 |
| Task 10 | 群聊核心页面（Tab 切换 + 群聊对话页 + 创建群聊页） | ✅ 完成 |
| Task 11 | 群聊管理页面（群设置 + 成员管理 + 邀请入群） | ✅ 完成 |
| Task 12 | 群聊辅助功能（入群审批 + 搜索群聊） | ✅ 完成 |
| Task 13 | 管理端群组管理 + 文档更新 | ✅ 完成 |
| 代码审查修复 | 14 项修复（Critical×5 + Important×4 + Minor×2 + Suggestion×3） | ✅ 完成 |
| Playwright 测试 + 用户反馈修复 | 浏览器端到端测试 + 21 项修复（搜索/UI/交互/功能增强） | ✅ 完成 |

### 用户测试修复详情（Phase 2c）

| # | 修复内容 |
|---|----------|
| Fix T1 | create.vue 创建群后跳转使用 `result.id`（原 `result.group_id` 字段不存在） |
| Fix T2 | conversation/settings/members/join-requests 用 `user_nickname` 替代 `username`（对齐 DTO） |
| Fix T3 | 管理端群组详情弹窗 UI 全面重设计（卡片+头像+分区布局+成员mini头像） |
| Fix T4 | 群搜索结果已在群内的显示「已加入」标签，不再显示「申请加入」按钮 |
| Fix T5 | group_dao.go `SearchGroups` 改用 ILIKE 模糊匹配（原 to_tsvector 不支持混合词搜索） |
| Fix T6 | conversation.vue 增加 groupId=0 时从 chatStore 回退查找 group_id |
| Fix T7 | create.vue 最低选择人数从 2 改为 1（支持 2 人群聊） |
| Fix T8 | create.vue 支持搜索非好友用户并加入群聊（全站用户搜索 + 非好友标签） |
| Fix T9 | invite.vue 支持搜索非好友用户并邀请入群（同 create.vue 改造） |
| Fix T10 | admin/list.vue 成员表用 `username` 替代 `user_nickname`（对齐 admin DTO） |
| Fix T11 | 群聊已读回执优化：无人已读时显示「0人已读」（含点击跳转已读详情功能） |
| Fix T12 | 新好友聊天页 conversationId=0 时「加载更多」点击报错（hasMore 增加 ID 校验） |
| Fix T13 | 联系人 TabBar 添加好友申请未读数 badge（与消息 Tab 一致的交互体验） |
| Fix T14 | App.vue 启动时预加载好友申请数，确保 badge 立即可见 |
| Fix T15 | 单聊已读回执：后端 MarkRead 缺失 `im.message.read.ack` 推送（补全对方已读通知链路） |
| Fix T16 | 群成员列表页为所有角色添加身份标识（群主/管理员/成员） |
| Fix T17 | 全局修复 `e?.data?.message` → `e?.message`（8 个页面 18 处，确保后端错误信息正确展示） |
| Fix T18 | members.vue 管理操作弹窗改为自定义组件（头像+角色+图标操作列表），替代 uni.showActionSheet；三个点按钮从 @longpress 改为 @tap |
| Fix T19 | 已读详情页展示群内昵称：后端 DTO 新增 group_nickname 字段 + DAO 批量查询群昵称 + 前端主显群昵称/副显真实昵称 |
| Fix T20 | 消息免打扰 API 补全：后端缺失 DAO/Service/Controller/Router 完整链路（PUT /api/v1/im/conversations/:id/dnd） |
| Fix T21 | 联系人 Tab 页切换回来后数据不刷新：onMounted → 增加 onShow 生命周期钩子自动重新获取好友列表和待处理申请数 |

### 代码审查修复详情（Phase 2c）

| # | 优先级 | 修复内容 |
|---|--------|----------|
| Fix C1 | Critical | conversation.vue 角色类型不匹配：字符串改为 GROUP_ROLE 数字常量 |
| Fix C2 | Critical | settings.vue 群公告字段名 announcement 改为 notice（对齐后端 DTO） |
| Fix C3 | Critical | chat.js sendMessage 未传递 at_user_ids 到 WS payload |
| Fix C4 | Critical | create.vue 创建成功后导航错误：改为 /pages/group/conversation + 正确参数 |
| Fix C5 | Critical | admin/provider.go Wire Set 未注册 GroupManageService/Controller |
| Fix I1 | Important | store/group.js searchGroups 添加 append 参数解决分页加载竞态 |
| Fix I2 | Important | settings.vue 改为 fetchMembers() 刷新数据，不直接修改 computed 引用 |
| Fix I3 | Important | 新增 constants/group.js 前端角色常量定义，消除魔数 |
| Fix I4 | Important | group_manage_service.go 列表查询 N+1 优化：批量查询用户名和成员数 |
| Fix M1 | Minor | file.js JSON.parse 添加 try/catch 异常保护 |
| Fix M2 | Minor | conversation.vue isSelf 移除冗余临时状态条件 |
| Fix S1 | Suggestion | 群聊对话页新增禁言状态检测和输入栏禁用提示 |
| Fix S2 | Suggestion | join-requests.vue 注册 WS group.join.request 事件实时刷新 |
| Fix S3 | Suggestion | read-detail.vue 获取失败时添加 uni.showToast 用户提示 |

### Phase 2c 新增内容

#### 后端新增模块
- **file/** — 文件上传（MinIO SDK + 通用上传 API）
- **group/** — 群聊管理（Controller + Service + DAO + Model + Router）
  - 18 个群管理 REST API + 11 个 WS 群事件推送
- **im/ 扩展** — 群消息发送/撤回 + @提醒 + 单聊/群聊已读回执
- **admin/ 扩展** — 群组列表 + 群组详情 + 解散群聊

#### 前端新增页面（9 个）
| 页面 | 路径 | 功能 |
|------|------|------|
| 群聊对话页 | `pages/group/conversation.vue` | 群消息收发 + @选择器 + 已读计数 |
| 创建群聊页 | `pages/group/create.vue` | 好友/非好友多选 + 全站用户搜索 + 群名称输入 |
| 群设置页 | `pages/group/settings.vue` | 群信息修改 + 成员概览 + 退出/解散 |
| 群成员页 | `pages/group/members.vue` | 成员列表 + 全角色标识（群主/管理员/成员）+ 自定义操作弹窗 + 角色管理 + 禁言操作 |
| 邀请入群页 | `pages/group/invite.vue` | 好友/非好友多选 + 全站用户搜索 + 排除已在群内成员 |
| 入群审批页 | `pages/group/join-requests.vue` | 申请列表 + 通过/拒绝操作 |
| 搜索群聊页 | `pages/group/search.vue` | 关键词搜索 + 申请加入 + 已加入状态显示 |
| 已读详情页 | `pages/chat/read-detail.vue` | 已读/未读成员列表（群聊消息级）+ 群昵称优先展示 + 真实昵称副行 |
| 会话列表改造 | `pages/chat/index.vue` | Tab 切换（全部/单聊/群聊）+ @标记 + 免打扰标识 |

#### 管理端新增
| 页面 | 路径 | 功能 |
|------|------|------|
| 群组列表 | `views/group/list.vue` | 搜索 + 分页 + 详情弹窗 + 解散群聊 |

#### 数据库新增/变更
- `im_groups` — 群信息表（新增）
- `im_group_join_requests` — 入群申请表（新增）
- `im_message_reads` — 群消息已读表（新增）
- `im_conversation_members` — 扩展字段（role, nickname, is_muted, is_do_not_disturb, joined_at, at_me_count）
- `im_messages` — 扩展字段（at_user_ids BIGINT[]）

---

## 九、Phase 2d — 消息类型扩展

> **状态：** ✅ 已完成
> **设计文档：** `docs/plans/2026-03-04-phase2d-design.md`
> **实施计划：** `docs/plans/2026-03-04-phase2d-implementation.plan.md`
> **分支：** `feature/phase2d-message-types`

### 功能范围

| 模块 | 内容 |
|------|------|
| 文件上传增强 | 50MB 上传上限、图片缩略图（200px JPEG）、语音校验（mp3/wav/aac/m4a, 最长 60s） |
| 富媒体消息 | 图片（多图网格 + 大图预览）、语音（录制+播放+波形）、文件（卡片+下载+预览） |
| IM Service | extra JSON 存储、会话列表预览文案（[图片x3]/[语音 12"]/[文件] xxx.pdf） |
| 前端组件 | MsgText/MsgImage/MsgVoice/MsgFile + MorePanel + VoiceRecorder |
| 管理端 | 消息列表（多条件筛选+撤回+删除+详情）+ 消息统计仪表板（ECharts 四图表） |

### Task 完成状态

| Task | 描述 | 状态 |
|------|------|------|
| Task 1 | 文件上传服务增强（50MB + 缩略图 + 语音校验） | ✅ 完成 |
| Task 2 | 消息 extra 结构定义 + DTO 扩展 + 常量启用 | ✅ 完成 |
| Task 3 | IM Service 适配富媒体消息 | ✅ 完成 |
| Task 4 | 管理端消息 DAO + Service + Controller | ✅ 完成 |
| Task 5 | 消息组件体系 + conversation 页改造 | ✅ 完成 |
| Task 6 | 图片消息完整流程 | ✅ 完成 |
| Task 7 | 语音消息完整流程 | ✅ 完成 |
| Task 8 | 文件消息完整流程 | ✅ 完成 |
| Task 9 | 输入栏改造 | ✅ 完成 |
| Task 10 | 管理端消息列表页 | ✅ 完成 |
| Task 11 | 管理端消息统计页 | ✅ 完成 |
| Task 12 | 管理端路由 + Store + API | ✅ 完成 |
| Task 13 | 群聊适配 + 编译验证 | ✅ 完成 |
| Task 14 | 代码审查 + 文档更新 | ✅ 完成 |
| 代码审查修复 | C1: im_groups 表名修复 + C2: 管理端撤回 WS 推送补全 | ✅ 完成 |

### 后端新增/修改

#### 文件上传 API
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/v1/upload/image | 图片上传（含缩略图生成） |
| POST | /api/v1/upload/voice | 语音上传（含时长校验） |

#### 管理端消息 API
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/admin/messages | 消息列表（分页+多条件筛选） |
| GET | /api/v1/admin/messages/:id | 消息详情 |
| DELETE | /api/v1/admin/messages/:id | 删除消息（软删除） |
| PUT | /api/v1/admin/messages/:id/recall | 撤回消息（+WS 推送） |
| GET | /api/v1/admin/messages/stats | 消息统计 |

### 前端新增组件

| 组件 | 路径 | 功能 |
|------|------|------|
| MsgText.vue | components/msg/ | 文本消息渲染 |
| MsgImage.vue | components/msg/ | 图片网格 + 大图预览 |
| MsgVoice.vue | components/msg/ | 语音播放 + 波形 + 时长 |
| MsgFile.vue | components/msg/ | 文件卡片 + 下载 + 预览 |
| MorePanel.vue | components/chat/ | "+"展开面板 |
| VoiceRecorder.vue | components/chat/ | 长按录音 |

### 管理端新增页面

| 页面 | 路径 | 功能 |
|------|------|------|
| 消息列表 | views/message/list.vue | 多条件筛选 + 操作 + 详情弹窗 |
| 消息统计 | views/message/stats.vue | 趋势折线图 + 类型饼图 + 活跃排行 |

### Playwright 端到端测试（2026-04-17）

| 测试项 | 结果 | 说明 |
|------|------|------|
| 前台单聊输入栏改造 | ✅ 通过 | 语音切换 + MorePanel 展开正常 |
| 前台群聊输入栏改造 | ✅ 通过 | 与单聊一致的改造结构 |
| 管理端消息列表展示 | ✅ 通过 | 表格、分页、操作按钮正常 |
| 管理端筛选功能 | ✅ 通过 | 关键词/类型/发送者ID/重置全部正常 |
| 管理端撤回/删除操作 | ✅ 通过 | 确认弹窗 + 状态变更 + 按钮联动正常 |
| 管理端消息详情弹窗 | ✅ 通过 | 发送者信息 + 元数据 + 内容展示正常 |
| 管理端消息统计图表 | ✅ 通过 | ECharts 线图/饼图/条形图渲染正常 |

> 详细报告：`test-report-phase2d.md`

### 留待后续阶段

- 群头像上传 UI 完善
- 消息转发功能
- 视频消息支持
