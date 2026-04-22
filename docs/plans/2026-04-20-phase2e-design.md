# Phase 2e 设计文档：会议与通知系统

> **状态：** 🚧 进行中（2e-1 ✅ 已完成，2e-2 📋 **设计阶段完成待进入编码**，2e-3 📋 待开发）
> **分支：** `feature/phase2e-meeting-notification`（2e-1）；`feature/phase2e-2-meeting-mvp`（2e-2，基于 `origin/feature/phase2c-group-read-receipt`）
> **前置依赖：** Phase 2a（联系人 + WS）、Phase 2b（即时通讯）、Phase 2c（群聊+已读）、Phase 2d（消息类型扩展）全部完成
> **最后更新：** 2026-04-24（Phase 2e-2 Task 0-14 ✅ 已完成：PoC / media-server 骨架 / REST / DDL / Go 骨架 / 会议 REST / WS 信令 / Go↔Node HTTP / 生命周期 / 前端 mediasoup / 会议前后端页面 / 会议聊天 / meeting_invite 卡片 / docker-compose 双态；剩 Task 15 UI 打磨 + Task 16 E2E 收官）
> **子阶段专用文档：**
> - 2e-1：`docs/plans/2026-04-20-phase2e-1-design.md` + `docs/plans/2026-04-20-phase2e-1-implementation.plan.md`（✅ 已完成，含 TabBar「我的」未读红点优化）
> - 2e-2：`docs/plans/2026-04-21-phase2e-2-design.md`（16 章节）+ `docs/plans/2026-04-21-phase2e-2-implementation.plan.md`（17 个 Task，约 17 人日）

---

## 一、设计目标

基于 Phase 2a-2d 建立的 WebSocket + 消息 + 联系人 + 群聊基础设施，实现 MVP 第一期收官的两大核心能力：

1. **统一通知系统**：消除散落在好友/群聊/会议各模块的通知死角，提供"提醒 + 历史"双通道
2. **多人音视频会议**：基于 mediasoup SFU 架构，支持即时会议（MVP）→ 预约会议 + 邀请（增强）

**核心交付物（按子阶段）：**
- **Phase 2e-1 通知系统**（3-4 人日）✅ 已完成：统一通知中心 + 11 种通知类型预留 + 跨模块 Pusher 接口 + TabBar「我的」未读红点
- **Phase 2e-2 会议 MVP**（约 17 人日）📋 设计阶段完成：mediasoup Node 媒体服务 + 即时会议（≤8 人）+ 密码/邀请链接/通知邀请三合一 + 设备预览页 + 主持人四件套 + 会议内聊天 + 双态部署（本机/公网 coturn）+ 桌面/手机响应式
- **Phase 2e-3 会议增强**（7-10 人日）：预约会议 + 定时提醒 + 等候室/锁定会议 + 设备预览高级参数（降噪/回声/虚拟背景）

**不包含（明确推迟）：** 见 [§九 后续规划清单](#九后续规划清单必须留档)

---

## 二、阶段拆分与路线图

```
┌──────────────────────┐   ┌──────────────────────┐   ┌──────────────────────┐
│  Phase 2e-1          │   │  Phase 2e-2          │   │  Phase 2e-3          │
│  通知系统            │─▶│  会议 MVP            │─▶│  会议增强            │
│  3-4 人日            │   │  10-14 人日          │   │  7-10 人日           │
│                      │   │                      │   │                      │
│ ✓ 好友/群聊事件通知  │   │ ✓ mediasoup Node     │   │ ✓ 预约会议+定时提醒  │
│ ✓ 系统广播（后端）   │   │ ✓ 即时会议（≤8人）   │   │ ✓ 会议邀请（复用 2e-1│
│ ✓ meeting_invite /   │   │ ✓ 音视频+主持人控制  │   │   通知类型）         │
│   meeting_reminder   │   │ ✓ 会议号/密码        │   │ ✓ 入会前预览         │
│   类型预留           │   │                      │   │                      │
└──────────────────────┘   └──────────────────────┘   └──────────────────────┘
```

**拆分理由：**
- **2e-1 风险最低、收益最高**：纯业务逻辑扩展，复用 Phase 2a-2c 的 WS 基础设施；能立即解决"好友申请无感知"等体验死角
- **2e-2 是风险核心**：引入全新技术栈（Node.js + mediasoup + WebRTC），需独立周期聚焦
- **2e-3 是收尾增强**：依赖 2e-1（邀请/提醒走通知通道）和 2e-2（会议能运行），做在最后

---

## 三、Phase 2e-1 详细设计（通知系统）

### 3.1 需求决策记录

| 决策项 | 选择 | 理由 |
|---|---|---|
| 推送样式 | 双通道（持久化入库 + 底部 mini-toast） | 不遗漏 + 不打扰，参照微信 |
| 主入口位置 | 「我的」Tab 顶部铃铛图标 + 数字徽标 | 不占用底部 Tab 位（已满 4 个） |
| 列表组织 | 顶部 Tab 分类：全部 / 好友 / 群聊 / 会议 / 系统 | 结构清晰，便于筛选 |
| 保留期限 | 30 天（每日定时任务清理已读通知） | 平衡存储与体验 |
| 历史追溯 | 不追溯，上线当天起新事件才入通知中心 | 无数据迁移风险 |
| 多端已读同步 | **暂不支持**（WS Hub 单端连接架构，同一用户仅最新连接有效） | 架构约束，已推迟到 Phase 2f/二期 |
| 点击行为 | Deep-link 按类型分发；群/会议邀请支持内联操作按钮 | 减少跳转步骤 |
| Toast 点击 | 无交互（避免误触），仅 2 秒自动消失 | 防止误操作 |
| 通知分类开关 | 不做（统一打开） | 简化 MVP，有需要再加 |

### 3.2 通知类型枚举（11 种，含 2e-2/2e-3 预留）

| type | 触发场景 | 触发模块 | 落地 Phase | Deep-Link 跳转 |
|---|---|---|---|---|
| `friend_request` | 收到好友申请 | contact | 2e-1 | `pages/contact/request` |
| `friend_accepted` | 好友申请被接受 | contact | 2e-1 | `pages/contact/detail?id=<actor_id>` |
| `friend_rejected` | 好友申请被拒绝 | contact | 2e-1 | 无跳转（告知型） |
| `group_invite` | 被邀请加入群聊 | group | 2e-1 | **内联接受/拒绝** |
| `group_join_request` | 收到入群申请（群管理员）| group | 2e-1 | `pages/group/join-requests?groupId=<target_id>` |
| `group_join_approved` | 入群申请被批准 | group | 2e-1 | 直接进入群会话 |
| `group_join_rejected` | 入群申请被拒绝 | group | 2e-1 | 无跳转（告知型） |
| `group_kicked` | 被踢出群聊 | group | 2e-1 | 无跳转（告知型） |
| `group_role_changed` | 被设/撤管理员、群主转让 | group | 2e-1 | 群详情页 |
| `system_broadcast` | 系统广播 | notify（admin 触发）| 2e-1 仅后端 API | 通知详情页 |
| `meeting_invite` | 会议邀请 | meeting | **2e-2 对接** | **内联加入/稍后** |
| `meeting_reminder` | 预约会议开始前 N 分钟 | meeting | **2e-3 对接** | **内联加入** |

### 3.3 数据库设计（新增 1 张表）

```sql
CREATE TABLE notify_notifications (
    id          BIGSERIAL    PRIMARY KEY,
    user_id     BIGINT       NOT NULL,
    type        VARCHAR(40)  NOT NULL,
    title       VARCHAR(200) NOT NULL,
    content     TEXT,
    extra       JSONB,
    actor_id    BIGINT,
    target_type VARCHAR(40),
    target_id   BIGINT,
    is_read     BOOLEAN      NOT NULL DEFAULT FALSE,
    read_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notify_user_unread ON notify_notifications(user_id, is_read, created_at DESC);
CREATE INDEX idx_notify_user_type   ON notify_notifications(user_id, type, created_at DESC);
```

**字段说明：**
- `extra`：按类型存附加信息，例如 group_invite 存 `{group_id, group_name, group_avatar, inviter_name}`、meeting_invite 存 `{room_code, room_title, host_name, scheduled_at}`
- `actor_id`：触发者（如好友申请发起人）
- `target_type` + `target_id`：关联对象（friend_request / group / meeting / join_request）

**清理任务**（每日 02:00 执行）：
```sql
DELETE FROM notify_notifications WHERE created_at < NOW() - INTERVAL '30 days' AND is_read = true;
```

### 3.4 后端 API（4 个）

```
GET  /api/v1/notifications                  列表（query: type, is_read, before_id, limit）
GET  /api/v1/notifications/unread-count     未读数统计（按 type 分组）
PUT  /api/v1/notifications/:id/read         标记单条已读
PUT  /api/v1/notifications/read-all         全部已读（可按 type 过滤）
```

**响应示例** — `GET /api/v1/notifications?limit=20`：
```json
{
  "code": 0, "message": "success",
  "data": {
    "list": [
      {
        "id": 1001, "type": "group_invite",
        "title": "张三邀请你加入群聊「产品团队」",
        "content": "",
        "extra": { "group_id": 5, "group_name": "产品团队", "inviter_name": "张三" },
        "actor_id": 7, "target_type": "group", "target_id": 5,
        "is_read": false, "created_at": "2026-04-20T10:00:00Z"
      }
    ],
    "has_more": false
  }
}
```

### 3.5 WebSocket 事件（2 个，**单端连接架构**）

| 事件 | 方向 | Payload | 说明 |
|---|---|---|---|
| `notify.new` | S→C | 完整通知对象 | 新通知到达，前端入 store + 更新徽标 |
| `notify.unread.total` | S→C | `{ total: 10, by_category: { friend: 3, group: 5, ... } }` | 连接建立/断线重连时补偿推送（权威值覆盖本地） |

> **说明**：由于现有 `ws.Hub` 为单端连接（同一用户仅保留最新连接），**不再设计** `notify.read.ack` 跨设备广播。多端已读同步作为独立技术债推迟到 Phase 2f（详见 §九）。

### 3.6 跨模块 Pusher 接口（沿用 Phase 2a 接口注入标准）

```go
// app/notify/service/pusher.go
type Pusher interface {
    Push(ctx context.Context, userID int64, req *PushRequest) error
}

type PushRequest struct {
    Type       string                 // 通知类型
    Title      string
    Content    string
    Extra      map[string]interface{}
    ActorID    int64
    TargetType string
    TargetID   int64
}
```

**使用方示例**（contact 模块好友申请）：
```go
// contact/service/contact_service.go
s.notifyPusher.Push(ctx, receiverID, &notify.PushRequest{
    Type: "friend_request",
    Title: fmt.Sprintf("%s 请求添加你为好友", applicant.Nickname),
    Content: applicationMessage,
    ActorID: applicantID,
    TargetType: "friend_request",
    TargetID: requestID,
})
```

### 3.7 模块结构

```
backend/go-service/app/notify/
├── constants/notify_types.go              # 11 种 type 枚举常量
├── model/notification.go                  # GORM 模型
├── dao/notification_dao.go                # CRUD
├── service/
│   ├── notify_service.go                  # 业务逻辑：创建/标已读/清理
│   └── pusher.go                          # Pusher 接口 + Impl（持久化 + WS 推送）
├── controller/notification_controller.go  # HTTP 接口
└── provider/wire.go                       # Wire 依赖注入

backend/go-service/app/dto/notify_dto.go   # DTO
```

### 3.8 前端页面

| 路径 | 变更 | 说明 |
|---|---|---|
| `store/notify.js` | **新建** | Pinia Store：state/fetch/markRead/initWs |
| `pages/notify/index.vue` | **新建** | 列表页 + 顶部 5 个分类 Tab + 下拉刷新 + 上拉加载 |
| `pages/profile/index.vue` | 修改 | 顶部添加「🔔 通知」入口 + 数字徽标 |
| `components/notify/NotifyItem.vue` | **新建** | 通知卡片（按类型渲染不同 UI，支持内联操作按钮） |

### 3.9 Pinia Store 设计要点

```javascript
// frontend/src/store/notify.js
export const useNotifyStore = defineStore('notify', () => {
  const notifications = ref([])          // 当前已加载的通知列表
  const unreadCount = ref(0)             // 总未读数（TabBar 徽标）
  const unreadByType = ref({})           // 分类型未读数
  const hasMore = ref(true)
  const currentFilter = ref('all')       // all / friend / group / meeting / system

  // WS 监听：notify.new / notify.unread.total（单端架构，无 notify.read.ack）
  const initWsListeners = () => { ... }

  const fetchNotifications = async (filter, beforeId) => { ... }
  const markRead = async (id) => { ... }
  const markAllRead = async (type) => { ... }
})
```

---

## 四、Phase 2e-2 会议 MVP 范围锁定（摘要 + 引用专用设计）

> **📘 详细设计已独立出文档：**
> - [Phase 2e-2 专用设计文档](./2026-04-21-phase2e-2-design.md) — 16 章节完整设计（架构 / 数据模型 / API / WS 信令 / UI/UX / 风险 / 验收）
> - [Phase 2e-2 实施计划](./2026-04-21-phase2e-2-implementation.plan.md) — 17 个 Task 共约 17 人日
>
> 本节仅保留摘要，作为总路线图的衔接锚点。任何细节调整请以上述两份专用文档为准（本文不再跟随更新）。

### 4.1 范围摘要（与专用设计 §2.2 一致）

- ✅ 即时会议（`XXX-XXX-XXX` 会议号）+ 可选密码（bcrypt）+ 邀请链接 + **通知中心 `meeting_invite`**
- ✅ ≤ 8 人音视频（mediasoup SFU + simulcast 三档）
- ✅ **入会前设备预览页**（摄像头/麦克风/扬声器选择 + 本地预览 + 音量检测）
- ✅ 主持人四件套：静音他人 / 移除成员 / **转让主持人** / 结束会议
- ✅ 生命周期：host 掉线 2 分钟宽限 + 自动转让（最早加入者）+ 空房 5 分钟 TTL
- ✅ **会议内文字聊天**（独立 `meeting_chats` 表，24 小时后清理）
- ✅ **桌面 + 手机双端响应式**（3×3 / 2×2 / 单列+抽屉）
- ✅ **双态部署**：本机 Docker Compose + 公网 `announcedIp` + coturn（`--profile public`）
- ❌ 预约会议 / 提醒 / 等候室 / 锁定 → Phase 2e-3
- ❌ 屏幕共享 / 录制 / 虚拟背景 → 第二期
- ❌ 管理端会议列表 / 详情 / 强制关闭 → Phase 2f

### 4.2 技术选型（锁定）

| 组件 | 选型 | 备注 |
|---|---|---|
| 媒体服务目录 | `media-server/` 根级子项目（与 `backend/` / `frontend/` / `admin/` 并列） | TypeScript + Fastify + mediasoup v3 |
| 客户端库 | `mediasoup-client` JS SDK | `markRaw` 包裹避免 Pinia 响应式代理 |
| 信令通道 | **复用现有 WebSocket Hub** | 新增 `MeetingSignalDispatcher` 接口注入 |
| Go ↔ Node | HTTP REST（容器内网，`X-Internal-Token` 鉴权） | 9 个 API：Router/Transport/Producer/Consumer 生命周期 |
| 数据库表 | `meeting_rooms` + `meeting_participants` + `meeting_chats` | 在总设计基础上修订：`password_hash`（bcrypt） + `ended_reason` + `meeting_chats` 新表 |
| Redis 键 | `echo:meeting:room:{code}` / `members:{code}` / `transport:{code}:{user_id}` / `invite:{token}` / `host_grace:{code}` | 后两个为新增 |

### 4.3 WS 信令事件（11 个，详见专用设计 §6.3）

房间组 3 个 + 成员组 4 个 + 媒体组 5 个，命名统一 `meeting.*` 前缀。

### 4.4 架构变化关键点（衔接 Phase 2e-1）

- **Go 主控 + Node 无状态包装**：所有权威状态在 Go，Node 仅做 mediasoup HTTP 封装
- **跨模块通信模式延续**：`meeting.service.NotifyPusher` 接口 ← 由 `notify.service.NotifyService` 实现，Wire 注入（与 Phase 2a 的 `ws.FriendIDsGetter` / Phase 2e-1 的 `contact.service.NotifyPusher` 完全同构）
- **单端 WS 连接架构**：继续沿用，不因会议而改造；多端改造仍保留给 Phase 2f

---

## 五、Phase 2e-3 会议增强范围锁定

> 注：原计划在 2e-3 的「会议邀请（`meeting_invite`）」与「入会前设备预览」已上移到 **Phase 2e-2**（见 [Phase 2e-2 专用设计 §2.2.1](./2026-04-21-phase2e-2-design.md) 与 §十）。Phase 2e-3 聚焦「预约 / 提醒 / 等候室 / 锁定」。

- 预约会议（`meeting_rooms.type=2`）+ 前端预约表单
- 定时器：到预约时间前 N 分钟触发 `meeting_reminder` 通知
- 等候室（waiting room）+ 锁定会议（lock）
- 设备预览高级参数：降噪 / 回声消除 / 虚拟背景（若浏览器支持）

---

## 六、Pusher 调用点汇总（Phase 2e-1 必须改动的现有代码）

| 文件 | 变更类型 | 说明 |
|---|---|---|
| `app/contact/service/contact_service.go` | 新增 Pusher 注入 + 4 处调用 | 申请/接受/拒绝 3 个事件 |
| `app/group/service/group_service.go` | 新增 Pusher 注入 + 6 处调用 | 邀请/入群申请/审批/踢人/角色变更 |
| `app/provider/wire.go` | 注册 notify 模块依赖 | Wire 依赖注入 |
| `frontend/src/store/contact.js` | 删除冗余的 `notify.friend.request` 处理（由 notify store 接管）| 避免重复提示 |
| `frontend/src/store/group.js` | 删除 `_onJoinRequest` toast（由 notify store 统一）| 避免重复提示 |
| `frontend/src/components/CustomTabBar.vue` | 保持不变 | 联系人 Tab 徽标仍显示"待处理"数，与通知中心徽标并存 |

**重要兼容性约束**：
- **联系人 Tab 徽标**（pendingCount）**保留不变**，继续承担"待办事项"入口
- **通知中心徽标**仅计"未读通知"，两套红点语义不同，互不影响

---

## 七、风险与应对

| 风险 | 等级 | 应对 |
|---|---|---|
| Pusher 调用失败导致业务阻塞 | 中 | 采用"异步推送"：Pusher 调用失败仅记日志，不阻塞业务主流程 |
| 通知表膨胀 | 低 | 30 天清理任务 + user+is_read 索引 + 按用户分表留作未来优化 |
| ~~多端已读同步消息风暴~~ | — | ~~不适用~~：当前 ws.Hub 单端连接架构已不做多端同步，该风险天然规避 |
| mediasoup（2e-2）技术深度 | 高 | 单独做技术 Spike，必要时先搭 PoC 验证 |
| Docker Compose 增加 Node 服务后的资源占用 | 低 | 设置资源限额，文档化本机运行要求 |

---

## 八、验收标准（Phase 2e-1）✅ 已完成

- [x] 11 种通知类型中的 10 种（除 `meeting_invite`/`meeting_reminder`）都能正确触发并落库
- [x] 用户 A 给 B 发好友申请 → B 端 WS 实时收到 `notify.new`，通知中心出现记录，「我的」Tab 红点 +1
- [x] ~~多设备登录同一账号同步~~ → **架构限制不支持**（单端 WS Hub），已在 §3.1/§3.5 修订；仅当前连接设备可感知实时更新，其他设备依赖下次重连时的 `notify.unread.total` 补偿
- [x] 30 天清理任务能正确运行，已读过期通知被删除
- [x] 通知中心列表按 5 个分类 Tab 筛选正确
- [x] 群邀请通知卡片支持内联"接受/拒绝"按钮并触发正确业务逻辑
- [x] 验证清单：`test-report-phase2e-1-notification.md`；Playwright 自动化推迟到 CI 建设阶段（Phase 2f 统一接入）

---

## 九、后续规划清单（必须留档）

### 9.1 推迟到 Phase 2f（MVP 收尾 + 管理端扩展）

| 功能 | 原计划阶段 | 说明 |
|---|---|---|
| **WS Hub 多端连接支持改造** | 2e-1 | 当前 `ws.Hub` 的 `clients map[int64]*Client` 仅支持单连接，新登录会踢掉旧设备。需改为 `map[int64]map[deviceID]*Client` 并适配 `PublishToUser` / `ws.OnlineService` / Pusher 等下游消费者 |
| 多端已读同步（notify.read.ack） | 2e-1 | WS Hub 多端就绪后，再补 `notify.read.ack` 事件广播给同一用户的其他设备 |
| 管理端会议列表/详情/强制关闭 | 2e | `/api/v1/admin/meetings*` 已在总设计文档定义，仅缺实现 |
| 管理端会议统计仪表板 | 2e | `/api/v1/admin/meetings/stats` |
| 管理端通知广播发布 UI | 2e-1 | 后端 API 已就绪，仅缺前端表单页 |
| 管理端用户会议记录 | 2e | `/api/v1/admin/users/:id/meetings` |
| 管理端操作日志页面 | 2e | 数据表 `admin_operation_logs` 已设计 |
| 管理端仪表板总览 | 2e | `/api/v1/admin/dashboard` |
| 系统配置管理 | 2e | `/api/v1/admin/system/config` |
| 通知分类开关设置 | 2e-1 | 用户级通知偏好（push/不push） |
| Playwright E2E 自动化 CI | 2e-1 | 当前仅 `test-report-*.md` 手动验证，CI 接入需评估整套 e2e 基础设施 |

### 9.2 推迟到第二期

| 功能 | 说明 |
|---|---|
| 屏幕共享 | mediasoup Producer 扩展为 screen 类型 |
| 会议录制与回放 | 需 mediasoup 录制插件 + 对象存储（MinIO 已就绪） |
| 虚拟背景 / 背景模糊 | 客户端 WebRTC 滤镜 |
| 微信授权登录 | OAuth + UnionID |
| 互动直播（主播/观众/弹幕） | 独立直播流架构 |
| 消息撤回时间延长 / 管理员无时限撤回的审计日志 | —— |
| 视频消息（type=4）| Phase 2d 已显式推迟 |
| 表情包 / 自定义贴纸 | Phase 2d 已显式推迟 |
| 消息转发 / 合并转发 / 引用回复 | Phase 2d 已显式推迟 |

### 9.3 推迟到第三期

- 微服务拆分（auth / im / meeting / notify 拆独立服务）
- Kubernetes 部署编排
- 跨服务器会议（多 mediasoup Worker 集群 + Router Pipe）
- AI 辅助：语音转文字、会议纪要、智能摘要

---

## 十、文档同步与变更记录

| 日期 | 变更 |
|---|---|
| 2026-04-20 | Phase 2e 规划完成：拆分为 2e-1/2e-2/2e-3 三个子阶段；本文档落盘 |
| 2026-04-20 | Phase 2e-1 实施完成：落实 notify 模块（后端 11 种类型 / 5 REST + 2 WS / 30 天清理）+ 前端通知中心；§3.1/§3.5/§八/§九 同步修订「单端 WS 连接」约束与推迟项 |
| 2026-04-24 | Phase 2e-2 推进 Task 0-14 全部完成：mediasoup PoC / media-server 9 REST / 3 表 DDL + DAO / Go 12 REST + WS 13 事件 / HTTPMediaOrchestrator / 生命周期状态机（host 宽限 + 自动转让 + 空房 TTL）/ 前端 mediasoup-client + Pinia store / 会议主页面（Hub/Create/Join/Preview/Room）+ 核心组件 / 会议内聊天 / meeting_invite 通知卡片（含过期态 + deep-link）/ docker-compose 双态扩展（media-server + coturn public profile）+ 三份 .env 模板 + deploy-public.sh + 部署指南。剩 Task 15 UI 打磨 + Task 16 E2E 收官 |
