# Phase 2a 设计文档：WebSocket 实时通讯与联系人管理

> **状态：** 设计确认，开发中
> **分支：** `feature/phase2a-websocket-contacts`
> **实施计划：** `docs/plans/2026-03-02-phase2a-implementation.md`（独立文件）
> **前置依赖：** Phase 1 全部完成（用户认证 + 管理端用户管理）

---

## 一、设计目标

搭建 WebSocket 实时通讯基础设施，实现完整联系人/好友管理功能，为 Phase 2b 即时聊天奠定基础。

**核心交付物：**
- WebSocket 长连接（心跳、断线重连、Redis Pub/Sub 消息总线）
- 联系人完整功能（好友申请/分组/黑名单/搜索/推荐）
- 在线状态管理（混合推拉方案）
- 管理端：在线监控 + 好友关系管理

---

## 二、架构方案

### 2.1 消息总线：Redis Pub/Sub 全量路由

所有跨用户的实时消息经 Redis Pub/Sub 路由，每个用户拥有独立频道。

**消息投递链路：**

```
业务层（Service）
    ↓ PUBLISH echo:ws:channel:{targetUserID}
Redis Pub/Sub
    ↓ SUBSCRIBE
目标实例 Hub
    ↓ send chan
目标 Client WebSocket
```

**方案选定理由：**
- 天然支持多实例扩展（第三期微服务拆分无需重构）
- 消息路由标准化，所有模块统一使用 Pub/Sub
- 当前单实例阶段性能影响微乎其微（延迟增加 ~0.1-0.5ms）

### 2.2 WebSocket 连接管理

| 组件 | 职责 |
|------|------|
| `pkg/ws/hub.go` | Hub 连接管理（注册/注销/按 userID 查找） |
| `pkg/ws/client.go` | 客户端连接封装（readPump + writePump + 心跳） |
| `pkg/ws/message.go` | 统一消息协议（匹配设计文档 5.2 节） |
| `pkg/ws/pubsub.go` | Redis Pub/Sub 封装（Publish/Subscribe/Unsubscribe） |
| `app/ws/handler.go` | WebSocket 升级处理（JWT 认证） |

**连接生命周期：**
1. `GET /ws?token=xxx` → JWT 认证 → gorilla/websocket 升级
2. 创建 Client → 注册 Hub → Redis SADD + SET + SUBSCRIBE
3. 心跳保活 30s（ping/pong）+ Redis TTL 续期
4. 断线 → 注销 Hub → Redis SREM + DEL + UNSUBSCRIBE

**技术选型：** `gorilla/websocket`（社区最成熟，虽已归档但稳定可靠）

### 2.3 消息协议

复用系统设计文档 5.2 节：

```json
// 客户端 → 服务端
{ "event": "contact.request.send", "seq": 1001, "data": {...}, "time": "2026-03-02 10:30:00" }

// 服务端 → 客户端（推送）
{ "event": "notify.friend.request", "seq": 0, "data": {...}, "time": "..." }

// 服务端 → 客户端（ACK）
{ "event": "contact.request.send.ack", "seq": 1001, "code": 0, "message": "ok", "data": {...} }
```

事件命名规范：`{模块}.{对象}.{动作}`

---

## 三、联系人模块设计

### 3.1 数据模型

复用系统设计文档已有表：

- `contact_friendships` — 好友关系（双向存储，status: 0=待确认, 1=已接受, 2=已拒绝, 3=已拉黑）
- `contact_groups` — 好友分组

### 3.2 好友申请流程

```
A 搜索用户 B
    ↓ POST /contacts/request
创建 A→B 记录（status=0）
    ↓ Redis PUBLISH → B
B 收到 WebSocket 推送 notify.friend.request
    ↓ POST /contacts/accept
更新 A→B（status=1）+ 创建 B→A（status=1）
    ↓ Redis PUBLISH → A
A 收到 WebSocket 推送 contact.request.accepted
    → 双向好友关系建立
```

### 3.3 黑名单机制

复用 `contact_friendships.status=3`：
- 拉黑操作：双向删除好友记录 → 新建单向 status=3 记录
- 被拉黑方无法发送好友申请和消息
- 取消拉黑：删除 status=3 记录（不自动恢复好友关系）

### 3.4 REST API

```
# 好友关系
GET    /api/v1/contacts                   好友列表（含在线状态）
POST   /api/v1/contacts/request           发送好友申请
POST   /api/v1/contacts/accept            接受申请
POST   /api/v1/contacts/reject            拒绝申请
DELETE /api/v1/contacts/:id               删除好友
PUT    /api/v1/contacts/:id/remark        设置备注
GET    /api/v1/contacts/requests          待处理申请列表

# 好友分组
GET    /api/v1/contacts/groups            分组列表
POST   /api/v1/contacts/groups            创建分组
PUT    /api/v1/contacts/groups/:id        修改分组
DELETE /api/v1/contacts/groups/:id        删除分组
PUT    /api/v1/contacts/:id/group         移动好友到分组

# 黑名单
POST   /api/v1/contacts/block             拉黑
DELETE /api/v1/contacts/block/:user_id    取消拉黑
GET    /api/v1/contacts/block             黑名单列表

# 搜索与推荐
GET    /api/v1/users/search               搜索用户
GET    /api/v1/contacts/recommend         好友推荐
```

---

## 四、在线状态设计

### 4.1 混合方案

| 场景 | 方式 | 说明 |
|------|------|------|
| 打开联系人页 | REST 拉取 | `GET /api/v1/contacts/online` 批量查询好友在线状态 |
| 好友上/下线 | WS 推送 | 通过 Pub/Sub 推送 `user.status.online`/`offline` |
| 心跳续期 | Redis TTL | `EXPIRE echo:user:status:{userID} 60` |

### 4.2 Redis 键设计（匹配系统设计文档 4.2 节）

```
echo:user:online                    SET    所有在线用户 ID
echo:user:status:{user_id}          STRING 状态 JSON（TTL=60s 心跳续期）
echo:ws:user:{user_id}              STRING 连接实例信息
echo:ws:channel:{user_id}           Pub/Sub 频道（消息投递）
```

---

## 五、管理端扩展

### 5.1 新增功能

- 仪表盘：实时在线用户数统计卡片
- 在线用户列表页
- 好友关系管理页（查看/删除）

### 5.2 管理端 API

```
GET    /api/v1/admin/online/users    在线用户列表
GET    /api/v1/admin/online/count    在线用户数
GET    /api/v1/admin/contacts        所有好友关系（分页）
DELETE /api/v1/admin/contacts/:id    管理员解除好友关系
```

---

## 六、前台用户端变更

### 6.1 WebSocket 客户端

- 单例模式，`ws://host/ws?token=xxx`
- 心跳 30s，断线指数退避重连（1s → 2s → 4s → 8s → 30s max）
- 事件分发：`on(event, callback)` / `off(event, callback)`

### 6.2 新增页面

- 联系人列表（替换占位页）
- 好友申请列表
- 好友详情（备注/分组/删除/拉黑）
- 搜索添加好友
- 好友分组管理
- 黑名单

---

## 七、文档管理策略

Phase 2a 开始执行新的文档拆分规则：

- **每个 Phase 独立设计文档**（本文件）
- **每个 Phase 独立实施计划**
- **API 文档按模块 + 端拆分**（每文件对应一个功能模块）
- **WebSocket 事件协议独立为文档**
- **单文件控制在 300-500 行以内**

```
docs/
├── api/
│   ├── admin/
│   │   ├── user.md            已有
│   │   ├── online.md          新增：在线监控 API
│   │   └── contact.md         新增：好友关系管理 API
│   └── frontend/
│       ├── auth.md            已有
│       ├── contact.md         新增：联系人 API
│       └── websocket.md       新增：WebSocket 事件协议
├── plans/
│   ├── 2026-02-27-echochat-system-design.md    总设计（蓝图参考）
│   ├── 2026-02-27-phase1-foundation-and-auth.md Phase1（已完成）
│   └── 2026-03-02-phase2a-design.md            本文件
└── progress/
    └── CURRENT_STATUS.md
```
