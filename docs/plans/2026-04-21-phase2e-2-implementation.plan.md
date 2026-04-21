# Phase 2e-2 实施计划：会议 MVP（多人音视频）

> **状态：** 📋 待执行（设计文档定稿后进入代码开发）
> **设计文档：** [Phase 2e-2 设计文档](./2026-04-21-phase2e-2-design.md)
> **上级路线图：** [Phase 2e 整体路线图](./2026-04-20-phase2e-design.md)
> **分支：** `feature/phase2e-2-meeting-mvp`
> **预估总工时：** **约 17 人日**（17 个 Task，含 PoC 与 UI 打磨）
> **最后更新：** 2026-04-21（实施计划首版落盘）

---

## 一、范围锁定

### 本期交付（与设计文档 §2.2.1 一一对应）

- 即时会议创建 / 加入 / 离开 / 结束 / 列表 / 详情
- 会议号 `XXX-XXX-XXX` + 可选密码（bcrypt）+ 邀请链接 + 通知中心邀请（`meeting_invite`）
- 入会前设备预览页（设备选择 + 本地预览 + 音量检测）
- ≤ 8 人音视频（mediasoup SFU + simulcast 3 档）
- 主持人四件套：静音他人 / 移除成员 / 转让主持人 / 结束会议
- 会议生命周期：host 掉线 2 分钟宽限 + host 转让（最早加入者）+ 空房 5 分钟 TTL
- 会议内文字聊天（独立 `meeting_chats` 表，24 小时后清理）
- 响应式布局：桌面 3×3 / 平板 2×2 / 手机 单列+抽屉
- 双态部署：本机 Docker Compose + 公网 `announcedIp` + coturn `--profile public`

### 显式推迟（与设计文档 §2.2.2 一一对应）

预约会议 / 提醒 / 等候室 / 锁定 / 屏幕共享 / 录制 / 虚拟背景 / 联合主持人 / 管理端会议管理 / 多 Worker 集群。

---

## 二、Task 依赖拓扑

```mermaid
flowchart LR
    T0[Task 0 mediasoup PoC Spike] --> T1[Task 1 media-server 骨架]
    T1 --> T2[Task 2 Node 9 个 REST API]
    T0 --> T3[Task 3 数据库 DDL]
    T3 --> T4[Task 4 Go meeting 模块骨架]
    T4 --> T5[Task 5 会议 REST 接口]
    T4 --> T6[Task 6 WS 信令处理器]
    T2 --> T7[Task 7 Go-Node HTTP Client]
    T6 --> T7
    T5 --> T8[Task 8 生命周期状态机]
    T2 --> T9[Task 9 前端 mediasoup-client + Store]
    T5 --> T10[Task 10 前端预览/创建/加入页]
    T9 --> T11[Task 11 会议室主页]
    T11 --> T12[Task 12 会议内聊天面板]
    T5 --> T13[Task 13 meeting_invite 通知对接]
    T11 --> T14[Task 14 docker-compose + 双态配置]
    T11 --> T15[Task 15 UI/UX 打磨]
    T15 --> T16[Task 16 E2E + 代码审查 + 文档同步]
    T12 --> T16
    T13 --> T16
    T14 --> T16
    T8 --> T16
```

**关键路径**：T0 → T1 → T2 → T9 → T11 → T15 → T16（≈ 9 人日）
**并行机会**：T3/T4 可与 T1/T2 并行；T10/T13 可与 T11 并行；T14 可与 T15 并行。

---

## 三、文件与变更清单概览

### 3.1 新建子项目

- `media-server/` —— 根级子项目（与 `backend/` / `frontend/` / `admin/` 并列）
  - `src/app.ts` / `src/config.ts`
  - `src/mediasoup/{worker.ts, router.ts, codecs.ts}`
  - `src/routes/{router,transport,producer,consumer}.route.ts`
  - `src/services/{router,transport,producer,consumer}.service.ts`
  - `src/middlewares/internal-auth.ts`
  - `src/utils/logger.ts`
  - `tests/*.spec.ts`
  - `package.json` / `tsconfig.json` / `Dockerfile` / `.env.example`

### 3.2 后端新增（`backend/go-service/app/meeting/`）

| 文件 | 作用 |
|---|---|
| `constants/{meeting_status,meeting_role,ws_events}.go` | 常量 |
| `model/{meeting_room,meeting_participant,meeting_chat}.go` | GORM 模型 |
| `dao/{meeting_room_dao,meeting_participant_dao,meeting_chat_dao}.go` | DAO |
| `service/meeting_service.go` | 房间 CRUD + 生命周期 |
| `service/meeting_signal_service.go` | WS 信令处理 |
| `service/meeting_chat_service.go` | 会议内聊天 |
| `service/node_client.go` | Go → Node HTTP 客户端 |
| `service/interfaces.go` | 对外注入接口（`NotifyPusher` / `UserInfoResolver`） |
| `controller/{meeting_controller,meeting_chat_controller}.go` | REST Controller |
| `controller/meeting_ws_handler.go` | WS 事件处理（注册到 ws.Hub） |
| `router/meeting_router.go` | 路由注册 |
| `task/meeting_cleanup_task.go` | 定时任务：空房 TTL 兜底 + chat 24h 清理 |
| `provider/{provider,wire_gen}.go` | Wire |

### 3.3 后端改造

| 文件 | 改动 |
|---|---|
| `app/provider/wire.go` | 注册 `MeetingSet`；绑定 `meetingService.NotifyPusher = notifyService`、`meetingService.UserInfoResolver = userService`、`ws.Hub.MeetingSignalDispatcher = meetingSignalService` |
| `app/provider/provider.go` | `App` 新增 `MeetingController` / `MeetingChatController` / `MeetingCleanupTask` |
| `cmd/server/main.go` | 启动 `app.MeetingCleanupTask.Start()` + `defer Stop()` |
| `router/router.go` | 注册 meeting 路由 |
| `app/ws/hub.go` / `app/ws/handler.go` | 新增 `MeetingSignalDispatcher` 接口注入；WS 断线时触发 `meeting.OnUserDisconnect` |
| `deploy/docker/postgres/init.sql` | 追加 `meeting_rooms` / `meeting_participants` / `meeting_chats` DDL + 索引 |
| `app/dto/meeting_dto.go` | 请求 / 响应 DTO |

### 3.4 前端新增（`frontend/src/`）

| 文件 | 作用 |
|---|---|
| `api/meeting.js` | REST API 封装 |
| `constants/meeting.js` | 会议类型 / 角色 / WS 事件常量 |
| `store/meeting.js` | Pinia Store |
| `utils/mediasoup-client.js` | Device/Transport/Producer/Consumer 封装 |
| `pages/meeting/{index,create,join,preview,room}.vue` | 5 个页面 |
| `components/meeting/{VideoTile,VideoGrid,MeetingToolbar,MemberPanel,ChatPanel,InviteDialog,DevicePreview,NetworkBadge}.vue` | 8 个组件 |

### 3.5 前端改造

| 文件 | 改动 |
|---|---|
| `pages.json` | 新增 5 条 page 注册 |
| `utils/ws.js` | 增加 meeting 事件分发 |
| `components/NotifyItem.vue` | `meeting_invite` 卡片补齐「立即加入 / 稍后」 |
| `store/notify.js` | `meeting_invite` 点击跳转 `/pages/meeting/preview?code=xxx` |

### 3.6 Docker / 脚本

| 文件 | 改动 |
|---|---|
| `deploy/docker-compose.dev.yml` | 新增 `media-server` 服务；新增 `coturn` 服务（`profiles: [public]`） |
| `scripts/start.sh` | 增加 `media` 子命令（启动 media-server）与 `full` 子命令（全部服务） |
| `scripts/stop.sh` | 对应 stop |
| `scripts/status.sh` | 增加 media-server 进程/容器检测 |
| `scripts/deploy-public.sh`（新建） | 公网部署快捷脚本（校验 announcedIp、端口开放） |
| `.env.example` / `.env.local.example` | 新增 `MEDIASOUP_*` / `MEDIA_INTERNAL_TOKEN` / `TURN_*` |

---

## 四、Task 明细

### Task 0：mediasoup PoC Spike

- **目标**：在正式动工前用最小代码跑通"2 个浏览器 ↔ Node ↔ mediasoup ↔ 互相看见视频"，验证技术栈可用性与关键参数
- **依赖**：无（起点）
- **主要产出**：
  - `media-server/poc/`（临时目录，PoC 结束后删除或保留为 tests/examples）
  - Node 侧 Fastify + mediasoup 单 Router + 2 Transport + 2 Producer + 2 Consumer 跑通
  - 浏览器侧极简 HTML + mediasoup-client + 原生 WebSocket（不走 uni-app）
  - 压测记录：1 Worker 下 4/6/8 人会议的 CPU / 带宽 baseline
- **检查点**：
  - 本机 Chrome + Firefox 两个窗口可互相看见对方摄像头视频
  - 记录关键坑：`announcedIp` 在 localhost 的处理、DTLS 握手超时的排查方式
  - 输出 `media-server/docs/poc-notes.md`（含启动步骤 + 压测截图）
- **工作量**：**1 人日**
- **风险缓解**：若 PoC 过程中发现 mediasoup 学习曲线超预期，评估是否改用 `livekit-server`（本 Task 为决策关卡）

### Task 1：media-server 项目骨架

- **目标**：创建正式 `media-server/` 目录，包含 TS 配置、Fastify 入口、日志、中间件、Dockerfile
- **依赖**：T0
- **主要产出**：
  - `media-server/package.json`（固定版本号，不做动态解析）
  - `media-server/tsconfig.json`、`.eslintrc`、`.prettierrc`
  - `media-server/src/app.ts`（Fastify 实例 + `/healthz` 端点）
  - `media-server/src/config.ts`（zod 校验 + 环境变量加载）
  - `media-server/src/mediasoup/worker.ts`（单 Worker 启动 + die 自动重启）
  - `media-server/src/middlewares/internal-auth.ts`（`X-Internal-Token`）
  - `media-server/src/utils/logger.ts`（pino）
  - `media-server/Dockerfile`（多阶段构建）
  - `media-server/.env.example`
- **检查点**：
  - `pnpm install && pnpm dev` 本地可启动，`curl /healthz` 返回 `{ok: true, workerPid}`
  - Docker 镜像构建成功，运行时打印 Worker PID
- **工作量**：**0.5 人日**

### Task 2：Node 侧 9 个 REST API

- **目标**：实现设计文档 §7.2 的 9 个接口，mediasoup 资源完整生命周期
- **依赖**：T1
- **主要产出**：
  - `src/routes/router.route.ts`（POST/DELETE）
  - `src/routes/transport.route.ts`（POST / POST connect）
  - `src/routes/producer.route.ts`（POST / DELETE）
  - `src/routes/consumer.route.ts`（POST / POST resume / DELETE）
  - `src/services/*.service.ts`（对应的资源管理类，使用 `Map<id, Resource>`）
  - `tests/router.spec.ts` / `tests/transport.spec.ts`（vitest，覆盖创建+释放）
- **检查点**：
  - 所有接口走 zod 请求 Schema 校验，非法参数返回 400 带字段错误
  - 缺 `X-Internal-Token` 返回 401
  - 资源释放：DELETE router 后，内部 `routerMap.size === 0` 且 mediasoup `router.closed === true`
  - 单元测试覆盖率 ≥ 60%
- **工作量**：**2 人日**

### Task 3：数据库 DDL + 模型 + DAO

- **目标**：PostgreSQL 3 张表落地 + Go 侧 model/dao 完整实现
- **依赖**：T0
- **主要产出**：
  - `deploy/docker/postgres/init.sql` 追加 3 张表 + 索引 + COMMENT（对齐设计 §5.1）
  - `app/meeting/model/{meeting_room,meeting_participant,meeting_chat}.go`
  - `app/meeting/dao/*.go`（含 CRUD + 事务 + 按 code/user 查询 + 软删除等）
  - `app/meeting/constants/meeting_status.go`（`StatusPending/Active/Ended` = 0/1/2）
  - `app/meeting/constants/meeting_role.go`（`RoleParticipant/Host/CoHost` = 0/1/2，MVP 仅用 0/1）
- **检查点**：
  - `docker compose up postgres -d` 后表结构正确；可手动 `INSERT` 测试数据
  - DAO 单元测试：创建房间 + 参与者加入 + 主持人转让（事务）+ 列表查询
  - 外键约束 `ON DELETE CASCADE` 正常工作（删除房间自动清理参与者/聊天）
- **工作量**：**0.5 人日**

### Task 4：Go 侧 meeting 模块骨架 + Wire

- **目标**：controller/service/router/provider 空壳搭建 + Wire 绑定完成
- **依赖**：T3
- **主要产出**：
  - `app/meeting/controller/*.go`（空 handler + 路由注册）
  - `app/meeting/service/*.go`（空方法签名）
  - `app/meeting/service/interfaces.go`（`NotifyPusher` / `UserInfoResolver` 接口）
  - `app/meeting/router/meeting_router.go`
  - `app/meeting/provider/provider.go` + `wire_gen.go`
  - `app/provider/wire.go` 注册 `MeetingSet` + interface 绑定
  - `app/provider/provider.go` `App` 结构体新增字段
- **检查点**：
  - `wire ./app/provider` 生成成功，编译通过
  - `go run cmd/server/main.go` 启动无报错
  - 路由打印包含 `/api/v1/meeting/rooms` 等前缀
- **工作量**：**0.5 人日**

### Task 5：会议 REST 接口（创建/加入/离开/结束/列表/详情 + 邀请链接兑换 + 邀请）

- **目标**：填充 Task 4 骨架中的业务逻辑，完整实现设计 §6.2 的 12 个接口
- **依赖**：T4
- **主要产出**：
  - 会议号生成：`GenerateRoomCode()` 随机 9 位数字，冲突重试（< 3 次）
  - 密码 bcrypt 存取（`service.setPassword / verifyPassword`）
  - 加入会议时校验：会议存在、未结束、容量未满、密码正确、用户未在其他会议
  - 离开 / 结束：参与者表 `left_at` 填入 + duration 计算 + 触发 mediasoup 资源清理
  - 邀请链接 Token：`GenerateInviteToken() -> Redis SET EX 600`，`RedeemInviteToken(token)` 校验并删除
  - `POST /rooms/:code/invite` 接受 `{invitee_ids, group_ids}`，生成 Token 后调 `NotifyPusher`
  - DTO：`app/dto/meeting_dto.go` 完整定义请求/响应结构
  - API 文档：`docs/api/frontend/meeting.md` 新建（全部 12 接口 + 示例）
- **检查点**：
  - Postman 手测 12 接口全部 2xx；错误场景返回正确错误码（`meeting_not_found` / `meeting_full` / `password_incorrect` 等）
  - 容量限制：第 9 人加入返回 `meeting_full`
  - 密码连续错误 5 次锁定 10 分钟
- **工作量**：**1.5 人日**

### Task 6：WS 信令 11 事件处理器

- **目标**：实现设计 §6.3 的 11 个 WS 事件，完整对接到 `ws.Hub`
- **依赖**：T4
- **主要产出**：
  - `app/meeting/controller/meeting_ws_handler.go`：注册事件回调到 Hub
  - `app/meeting/service/meeting_signal_service.go`：3 组事件（房间 / 成员 / 媒体）的业务逻辑
  - `app/ws/hub.go` 接口扩展：`MeetingSignalDispatcher` 接口注入 + `DispatchMeeting(event, payload)` 方法
  - `app/meeting/constants/ws_events.go`：11 个事件名常量
  - 权限校验：所有事件 handler 入口调用 `assertIsParticipant` / `assertIsHost`
  - 广播：`Hub.BroadcastToMeeting(roomCode, event, payload, excludeUserID)` 辅助方法
- **检查点**：
  - 通过 `wscat` 或临时前端脚本连入 WS，逐个事件手测
  - 未授权事件（非参与者发 `meeting.member.state.changed`）被拒绝
  - 事件广播覆盖正确（excludeUserID 生效）
- **工作量**：**1.5 人日**

### Task 7：Go → Node HTTP Client 封装

- **目标**：实现设计 §6.6 的 `NodeClient` 接口，挂接到 WS 信令流程
- **依赖**：T2 + T6
- **主要产出**：
  - `app/meeting/service/node_client.go`：9 个方法完整实现
  - 配置：`config.NodeServiceURL` / `config.NodeInternalToken`（从 yaml + 环境变量）
  - 超时：HTTP Client 5 秒超时；关闭类操作 2 秒超时
  - 重试：关闭类操作失败重试 2 次（指数退避 200ms/500ms）
  - 日志：每次调用记录 `funcName + room_code + duration + status_code`
  - 错误映射：Node 5xx → 业务错误码 `media_server_error`；404 → `media_resource_not_found`；超时 → `media_timeout`
- **检查点**：
  - 单元测试：使用 `httptest.NewServer` 模拟 Node，覆盖成功/失败/超时三类
  - 集成测试：Go + Node 真实连通，创建 Router → Transport → Producer → 销毁链路
- **工作量**：**0.5 人日**

### Task 8：会议生命周期状态机（host 宽限期 + 自动转让 + 空房 TTL）

- **目标**：实现设计 §6.5 的状态机完整逻辑
- **依赖**：T5
- **主要产出**：
  - `app/meeting/service/meeting_service.go` 补充：
    - `OnHostDisconnect(ctx, roomCode)` → 写 `echo:meeting:host_grace:{code}` EX 120s
    - `OnHostReconnect(ctx, roomCode, userID)` → 清除宽限期键
    - `HandleHostGraceExpired(ctx, roomCode)` → 转让主持（挑最早加入者）或销毁会议
    - `OnAllMembersLeft(ctx, roomCode)` → 设置房间 Redis key TTL 300s
    - `OnRoomTTLExpired(ctx, roomCode)` → `status=2, ended_reason=empty_ttl`，清理 Node 资源
  - `app/meeting/task/meeting_cleanup_task.go`：每 30 秒扫描 Redis TTL + DB 状态，兜底清理
  - 事务包裹主持人转让（见设计 §11.3）
- **检查点**：
  - 手测：host 关闭浏览器 → 2 分钟内重连，身份保留 → 超过 2 分钟自动转让给另一成员
  - 手测：全员退出 → 5 分钟后 DB `status=2`、Redis key 清空
  - 单元测试覆盖：转让事务回滚、并发转让竞态
- **工作量**：**0.5 人日**

### Task 9：前端 mediasoup-client 集成 + Pinia Store

- **目标**：实现设计 §8.2 / §8.4 的 Store 状态与 mediasoup-client 生命周期封装
- **依赖**：T2
- **主要产出**：
  - `frontend/src/utils/mediasoup-client.js`：
    - `createDevice(rtpCapabilities)` → `mediasoupClient.Device`
    - `createSendTransport` / `createRecvTransport` 包装 + 事件桥接到 WS
    - `produceAudio(track)` / `produceVideo(track, encodings)`
    - `consume(producerId)` 全流程
    - 所有实例通过 `markRaw` 包裹
  - `frontend/src/store/meeting.js`：
    - state 完整定义（见设计 §8.2）
    - 10+ actions 与 WS `_on*` 事件响应
  - `frontend/src/constants/meeting.js`：事件名 / 类型 / 角色常量
  - `frontend/src/api/meeting.js`：REST 封装
  - `frontend/src/utils/ws.js` 增加 meeting 事件分发桥
- **检查点**：
  - 手动启动前端，在控制台调用 `useMeetingStore().createRoom({title: '测试'})` → 成功入会并推流
  - 两个浏览器标签互相看见视频
  - 关闭标签后另一端 1 秒内收到 `meeting.member.left`
- **工作量**：**1.5 人日**

### Task 10：前端设备预览页 + 创建页 + 加入页

- **目标**：完成 `/pages/meeting/{create,join,preview}.vue` 3 个页面
- **依赖**：T5
- **主要产出**：
  - `MeetingCreate.vue`：标题输入 + 密码 + 开关（入会静音 / 允许聊天） + 「立即开会」按钮
  - `MeetingJoin.vue`：会议号输入（3-3-3 分组） + 密码 + 承接 `?code=xxx` 参数
  - `MeetingPreview.vue`：
    - 左侧视频预览（`<video autoplay muted>` 承载 localStream）
    - 右侧设备列表（`enumerateDevices` 填充）
    - 音量条（AudioContext AnalyserNode 每 100ms 采样）
    - 显示名称确认 + 「加入会议」按钮
  - 浏览器兼容性检测：不支持 `getUserMedia` → 友好降级
- **检查点**：
  - Chrome / Firefox / Safari 手测设备选择 + 预览 + 音量条正常
  - 手机浏览器（Safari iOS / Chrome Android）布局无错位
  - 权限被拒绝时给出清晰引导
- **工作量**：**1 人日**

### Task 11：前端会议室主页（视频网格 + 工具栏 + 成员面板 + 响应式）

- **目标**：实现 `/pages/meeting/room.vue` + 6 个主要组件
- **依赖**：T9
- **主要产出**：
  - `MeetingRoom.vue`：主容器 + 网络质量监测 + WS 监听挂载
  - `VideoGrid.vue`：响应式网格（见设计 §8.5）+ 按成员数动态布局
  - `VideoTile.vue`：视频块（含静音图标、说话者流光占位）
  - `MeetingToolbar.vue`：5 按钮（麦 / 摄 / 成员 / 聊天 / 挂断）+ 邀请入口
  - `MemberPanel.vue`：成员列表 + host 菜单（静音 / 移除 / 转让）
  - `InviteDialog.vue`：复制链接 + 复制会议号 + 选择联系人（复用 Phase 2a 组件）
  - `NetworkBadge.vue`：3 档网络质量占位（具体动效 Task 15 打磨）
- **检查点**：
  - 4 人会议：UI 网格切换 1→2→3→4 人布局正常
  - host 操作：静音他人 / 移除成员 / 转让主持 / 结束会议 全部成功
  - 手机浏览器：单列主画面 + 缩略抽屉工作正常
- **工作量**：**2 人日**

### Task 12：会议内聊天面板

- **目标**：实现 `ChatPanel.vue` + 后端 `meeting_chats` 相关接口
- **依赖**：T11
- **主要产出**：
  - 后端：`meeting_chat_controller.go` + `meeting_chat_service.go`（2 接口：POST 发送、GET 历史分页）
  - WS 事件：`meeting.chat.new`（广播给房间内所有成员）
  - 前端：`ChatPanel.vue`（消息列表 + 发送框 + 滚动到底部）
  - Store 内 `chat: {messages, hasMore}` 字段维护
  - 清理任务：`meeting_cleanup_task` 每天扫描 `status=2 AND ended_at < NOW() - INTERVAL '24 hours'`，`DELETE FROM meeting_chats WHERE room_id IN (...)`
- **检查点**：
  - 两个用户互发 10 条消息，顺序正确
  - 会议结束 24 小时后聊天记录被清理（可手动 `UPDATE ended_at = NOW() - INTERVAL '25 hours'` 触发）
- **工作量**：**0.5 人日**

### Task 13：`meeting_invite` 通知对接

- **目标**：打通 `POST /rooms/:code/invite` → 通知中心 → 前端卡片跳转链路
- **依赖**：T5
- **主要产出**：
  - 后端：`Invite()` service 内调用 `notifyPusher.Push(ctx, receiverID, PushRequest{Type: "meeting_invite", Extra: MeetingInviteExtra{...}})`
  - 通知 `extra` 结构补充（设计 §10.1）
  - 前端 `NotifyItem.vue`：`meeting_invite` 卡片底部渲染「立即加入 / 稍后」按钮
  - 前端路由跳转：「立即加入」→ `/pages/meeting/preview?code=xxx`；「稍后」→ 标已读
  - 邀请链接复用：若 `expired_at < now()` 灰显按钮提示"邀请已过期"
- **检查点**：
  - 用户 A 创建会议 → 邀请用户 B → 用户 B 通知中心铃铛出现未读 → TabBar「我的」红点亮起 → 点击卡片「立即加入」→ 成功入会
  - 过期通知点击按钮 → 显示过期提示，不跳转
- **工作量**：**0.5 人日**

### Task 14：docker-compose 扩展 + 环境变量双态开关

- **目标**：本机和公网双态部署脚本 + 文档
- **依赖**：T11
- **主要产出**：
  - `deploy/docker-compose.dev.yml` 新增 `media-server` 服务 + `coturn`（`profiles: [public]`）
  - `.env.example` / `.env.local.example` / `.env.public.example` 三份配置
  - `scripts/start.sh` 扩展 `media` / `full` 子命令
  - `scripts/stop.sh` / `scripts/status.sh` 同步
  - `scripts/deploy-public.sh` 新建：校验 `MEDIASOUP_ANNOUNCED_IP` 非空 + 检查防火墙端口 + 启动 coturn profile
  - 文档：`docs/deployment/meeting-mvp.md` 新建，涵盖本机 + 公网两种流程 + 常见问题
- **检查点**：
  - 本机：`scripts/start.sh full` 全量启动，5 分钟内全部服务就绪
  - 公网（模拟）：`MEDIASOUP_ANNOUNCED_IP=x.x.x.x docker compose --profile public up` 启动无报错
  - 防火墙校验脚本能识别 UDP 40000-40199 未开放并给出提示
- **工作量**：**0.5 人日**

### Task 15：`ui-ux-pro-max` 定制 UI 打磨

- **目标**：调用技能包产出 4 屏 EchoChat 原创风格，落地到代码
- **依赖**：T11
- **主要产出**：
  - 调用：`npx openskills read ui-ux-pro-max` + 4 屏 brief（设计 §9.1）
  - 设计落地：
    - `design-system/echochat/pages/meeting-home.md`
    - `design-system/echochat/pages/meeting-preview.md`
    - `design-system/echochat/pages/meeting-room.md`
    - `design-system/echochat/pages/meeting-invite.md`
  - 代码落地：
    - `VideoTile.vue` 说话者流光轮廓动效
    - 柔性网格：2/3 人布局的非等分样式
    - 自视频浮窗四角吸附（`draggable` + 吸附计算）
    - 静音氛围色（工具栏底色绑定 computed `allMuted`）
    - `NetworkBadge.vue` 3 档波浪动效
- **检查点**：
  - 视觉走查：4 屏与设计产物一致，色彩 / 留白 / 动效符合飞书简洁 + EchoChat 原创要求
  - 动效性能：不影响 60fps（Chrome Performance 面板确认）
  - 代码审查：`ui-ux-pro-max` 产物被真正使用，不留下"代码 vs 设计产物"脱节
- **工作量**：**2 人日**

### Task 16：E2E 验证 + Playwright MCP 回归 + 代码审查 + 文档同步

- **目标**：端到端闭环验证 + 交付收官
- **依赖**：T12 + T13 + T14 + T15 + T8
- **主要产出**：
  - Playwright MCP 场景脚本（保存到 `.playwright-mcp/scenarios/`）：
    - **场景 1**：创建会议 → 通过会议号加入第二个用户 → 互推音视频 → 结束
    - **场景 2**：邀请链接加入（含密码）
    - **场景 3**：通知中心「立即加入」按钮跳转
    - **场景 4**：主持人转让（主动）+ 主持人掉线（被动，2 分钟宽限）
  - `code-reviewer` 子代理运行 → 出具报告，修复 P0/P1
  - 文档同步（设计 §十六 变更记录 + 以下文件）：
    - `docs/progress/CURRENT_STATUS.md`：Phase 2e-2 状态改为 ✅
    - `docs/architecture/system-architecture.md`：新增 meeting 模块
    - `docs/api/README.md` + `docs/api/frontend/meeting.md`
    - `.cursor/rules/project-context.mdc`：当前进度更新
    - `docs/plans/2026-04-20-phase2e-design.md`：Phase 2e-2 段标记 ✅
  - 测试报告：`test-report-phase2e-2-meeting.md` 落盘
- **检查点**：
  - 4 个 E2E 场景全部通过
  - 代码评审 0 个 P0/P1
  - 所有文档链接可达、无 404
  - 清理所有 TODO / FIXME / `console.log` 调试残留
- **工作量**：**1 人日**

---

## 五、工作量汇总

| Task | 名称 | 工时 |
|---|---|---:|
| 0 | mediasoup PoC Spike | 1.0 |
| 1 | media-server 项目骨架 | 0.5 |
| 2 | Node 侧 9 个 REST API | 2.0 |
| 3 | 数据库 DDL + 模型 + DAO | 0.5 |
| 4 | Go 侧 meeting 模块骨架 + Wire | 0.5 |
| 5 | 会议 REST 接口 | 1.5 |
| 6 | WS 信令 11 事件处理器 | 1.5 |
| 7 | Go → Node HTTP Client | 0.5 |
| 8 | 生命周期状态机 | 0.5 |
| 9 | 前端 mediasoup-client + Pinia Store | 1.5 |
| 10 | 前端预览/创建/加入页 | 1.0 |
| 11 | 前端会议室主页 | 2.0 |
| 12 | 会议内聊天面板 | 0.5 |
| 13 | `meeting_invite` 通知对接 | 0.5 |
| 14 | docker-compose + 双态配置 | 0.5 |
| 15 | `ui-ux-pro-max` 定制 UI 打磨 | 2.0 |
| 16 | E2E + 代码审查 + 文档同步 | 1.0 |
| **合计** |  | **17.0** |

**说明**：原 Phase 2e 路线图对 Phase 2e-2 的估算为 10-14 人日。当前 17 人日相较增加 3-7 人日的原因：
- 纳入了**设备预览页**（Task 10 的一部分，约 0.5 人日）
- 纳入了**会议内聊天**（Task 12，0.5 人日）
- 纳入了**移动端响应式布局**（分摊到 Task 11/15，约 1 人日）
- 纳入了**邀请通知对接**与 Phase 2e-1 的闭环（Task 13）
- 纳入了**`ui-ux-pro-max` 定制 UI 打磨**（Task 15，2 人日）
- 预留了 **PoC Spike 与 E2E** 的显式工时（Task 0 / Task 16 共 2 人日）

该增量均是用户在 Plan 模式下的 11 项决策明确要求纳入的，已与设计文档 §2.2 对齐。

---

## 六、工作流与提交节奏

1. **分支**：`feature/phase2e-2-meeting-mvp`（已由上游决策指定）
2. **每个 Task 完成后**：
   - 本地 `go test ./app/meeting/...` / `pnpm test` / `npm run lint` 全绿
   - `code-reviewer` 子代理小审（仅重要 Task 5/7/9/11 跑全量审）
   - 小粒度 commit：`feat(meeting): task-N <简短描述>`
   - **不**自动推远端，由用户显式触发
3. **每周节奏**：Task 0-4 集中第一周（基础设施 + 骨架）；Task 5-13 分散第二周（业务主体）；Task 14-16 第三周（部署 + 打磨 + 收官）
4. **阻塞处理**：
   - PoC（Task 0）如失败，立即暂停，与用户重议选型（livekit-server 替换 mediasoup）
   - 任何 Task 工时超出估算 50% 以上，立刻停下来复盘，必要时拆子任务或推迟非关键部分
5. **Review Gate**：每完成 3-4 个 Task 触发一次 review gate 汇报阶段进展

---

## 七、显式不做（本实施计划范围外）

- 不写 Phase 2e-3 相关代码（预约 / 提醒 / 等候室 / 锁定）
- 不写管理端 meeting 模块代码（推迟到 Phase 2f）
- 不做录制 / 屏幕共享 / 虚拟背景（二期）
- 不做多 Worker / 多 Node 实例 / Router Pipe（三期）
- 不引入 CI 流水线（接入后再由运维任务统一实施）
- 不做付费 TURN 服务接入（MVP 自建 coturn 即可）

---

## 八、关联文档

- [Phase 2e-2 设计文档](./2026-04-21-phase2e-2-design.md)
- [Phase 2e 整体路线图](./2026-04-20-phase2e-design.md)
- [Phase 2e-1 实施计划](./2026-04-20-phase2e-1-implementation.plan.md)（参考格式）
- [系统总设计](./2026-02-27-echochat-system-design.md)
- [项目规则](../../.cursor/rules/project-context.mdc)
- [API 文档索引](../api/README.md)

---

## 九、变更记录

| 日期 | 作者 | 变更 |
|---|---|---|
| 2026-04-21 | Agent | 首版落盘，17 个 Task 共 17 人日，含 PoC / UI 打磨 / E2E |

---

**文档结束**
