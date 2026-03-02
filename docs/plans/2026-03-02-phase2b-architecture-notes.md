# Phase 2b 架构建议备忘录：即时通讯消息系统

> **创建日期：** 2026-03-02
> **状态：** 📋 待设计（供 Phase 2b 设计阶段参考）
> **前置依赖：** Phase 2a 全部完成（WebSocket + 联系人管理）

---

## 一、Phase 2a 经验总结与架构教训

### 1.1 接口注入模式（Interface Injection）

Phase 2a 后期修复中，`ws.OnlineService` 需要获取好友列表来推送上下线通知，但不应直接依赖 `contact` 包。通过定义 `FriendIDsGetter` 接口，由 `FriendshipDAO` 隐式实现，在 Wire 中注入——这是跨模块通信的标准模式。

**Phase 2b 建议：** IM 模块同样需要获取好友/联系人数据（如验证单聊权限、获取会话成员信息），必须遵循相同的接口注入模式，避免 `im` 包直接 import `contact` 包。

需要预定义的接口：

```go
// im 模块可能需要的外部依赖接口
type FriendChecker interface {
    IsFriend(ctx context.Context, userID, targetID int64) (bool, error)
}

type UserInfoGetter interface {
    GetUsersByIDs(ctx context.Context, userIDs []int64) ([]UserBasicInfo, error)
}
```

### 1.2 批量查询模式

Phase 2a 的 `GetRecommendFriends` 最初对每个候选人单独查询用户信息，性能极差。修复后使用 `GetUsersByIDs` 一次批量查询 + Map 映射。

**Phase 2b 建议：** IM 消息列表中需要展示发送者信息（头像、昵称），必须使用批量查询模式：
- 收集一页消息的所有 `sender_id`
- 去重后一次 IN 查询获取用户信息
- 构建 `map[int64]UserBasicInfo` 映射
- 按消息顺序填充发送者信息

### 1.3 Admin 端查询模式

`OnlineManageService` 最初只返回 `[]int64`，缺少用户名导致管理端无法展示。修复后注入 `*gorm.DB` 查询用户表补充信息。

**Phase 2b 建议：** 管理端的消息管理/会话管理 API 从一开始就设计为返回完整信息的 DTO，不要只返回 ID。

---

## 二、IM 模块架构建议

### 2.1 模块结构

```
app/im/
├── controller/
│   └── im_controller.go         # REST API 控制器
├── service/
│   └── im_service.go            # 业务逻辑（会话管理、消息收发）
├── dao/
│   ├── conversation_dao.go      # 会话 DAO
│   └── message_dao.go           # 消息 DAO
├── model/
│   ├── conversation.go          # im_conversations 模型
│   ├── conversation_member.go   # im_conversation_members 模型
│   └── message.go               # im_messages 模型
├── handler/
│   └── message_handler.go       # WebSocket 消息事件处理器
├── router.go                    # 路由注册
└── provider.go                  # Wire Provider Set
```

### 2.2 消息收发链路设计

```
发送者 Client
    │
    ├─ REST: POST /api/v1/im/conversations/:id/messages（HTTP 发消息）
    │  或
    ├─ WS: im.message.send（WebSocket 发消息）
    │
    ▼
IM Service
    │── 权限校验（是否为会话成员）
    │── 消息写入 PostgreSQL (im_messages)
    │── 更新会话 last_message 信息
    │── 更新 Redis 未读计数 (HINCRBY echo:im:unread:{member_id} conv_id 1)
    │
    ▼
PubSub.PublishToUser（复用 Phase 2a 的 Pub/Sub 基础设施）
    │── 推送给会话内所有在线成员
    │── 事件: im.message.new
    │
    ▼
接收者 Client（实时收到消息推送）
```

### 2.3 依赖注入设计

```go
type IMService struct {
    conversationDAO *dao.ConversationDAO
    messageDAO      *dao.MessageDAO
    pubsub          *ws.PubSub          // 复用 Phase 2a 的 Pub/Sub
    friendChecker   FriendChecker       // 接口注入，contact.FriendshipDAO 实现
    userInfoGetter  UserInfoGetter      // 接口注入，批量获取用户信息
}
```

Wire 注入链：
- `FriendshipDAO` → 隐式实现 `FriendChecker`（已有 `IsFriend` 方法）
- `FriendshipDAO` → 隐式实现 `UserInfoGetter`（已有 `GetUsersByIDs` 方法）

### 2.4 WebSocket 事件处理器

Phase 2a 的 WebSocket Handler 只处理心跳和连接管理。Phase 2b 需要增加消息事件路由：

```go
// 建议在 app/ws/handler.go 中增加事件分发机制
// 或在 app/im/handler/ 下实现 IM 事件处理器，注册到 Hub

// 需要处理的 WS 事件
// im.message.send    → 发送消息
// im.message.read    → 标记已读
// im.typing.start    → 正在输入
// im.typing.stop     → 停止输入
```

**建议方案：** 在 `pkg/ws/hub.go` 或 `app/ws/handler.go` 中实现事件路由表（`map[string]EventHandler`），各模块注册自己的事件处理器，避免 handler.go 膨胀成巨型文件。

---

## 三、数据库注意事项

### 3.1 单聊会话去重

单聊会话应保证两个用户之间只有一个会话。建议方案：
- 创建单聊时，用两个 user_id 的较小值和较大值组合查询是否已存在
- 或增加唯一约束 + 规范化存储（小 ID 在前）

### 3.2 消息分页查询

`im_messages` 将是数据量最大的表。已有索引 `idx_im_messages_conv_time (conversation_id, created_at DESC)`。

分页建议：
- 使用游标分页（`WHERE created_at < ? AND conversation_id = ? ORDER BY created_at DESC LIMIT ?`）而非 OFFSET 分页
- 前端传 `before_msg_id` 或 `before_time` 参数

### 3.3 未读消息计数

Redis HASH `echo:im:unread:{user_id}` → `{conv_id: count}`
- 收到新消息：`HINCRBY echo:im:unread:{user_id} {conv_id} 1`
- 标记已读：`HDEL echo:im:unread:{user_id} {conv_id}`
- 获取总未读数：`HVALS echo:im:unread:{user_id}` 求和

---

## 四、DTO 设计建议

```go
// 会话列表项（首页会话列表展示用）
type ConversationItem struct {
    ID              int64  `json:"id"`
    Type            int    `json:"type"`               // 1=单聊, 2=群聊
    Name            string `json:"name"`               // 群聊名称 / 对方昵称
    Avatar          string `json:"avatar"`             // 群头像 / 对方头像
    LastMessage     string `json:"last_message"`       // 最后一条消息预览
    LastMessageTime string `json:"last_message_time"`  // 最后消息时间
    UnreadCount     int    `json:"unread_count"`       // 未读消息数
    IsPinned        bool   `json:"is_pinned"`          // 是否置顶
}

// 消息项
type MessageItem struct {
    ID             int64       `json:"id"`
    SenderID       int64       `json:"sender_id"`
    SenderName     string      `json:"sender_name"`     // 批量查询填充
    SenderAvatar   string      `json:"sender_avatar"`   // 批量查询填充
    Type           int         `json:"type"`
    Content        string      `json:"content"`
    Extra          interface{} `json:"extra,omitempty"`
    Status         int         `json:"status"`
    CreatedAt      string      `json:"created_at"`
}
```

---

## 五、前端注意事项

### 5.1 WebSocket 事件集成

前台 `services/websocket.js` 已支持事件监听（`on/off`），Phase 2b 需要注册新事件：
- `im.message.new` → 更新会话列表 + 未读数 + 当前聊天窗口
- `im.typing.start/stop` → 显示"正在输入..."

### 5.2 Store 设计

```
store/
├── chat.js              # 会话列表、当前会话、消息缓存（新增）
└── ...existing stores
```

消息缓存策略：
- 每个会话缓存最新 N 条消息在内存中
- 向上翻页时 REST 拉取历史消息
- 新消息通过 WS 推送自动追加

---

## 六、管理端扩展建议

Phase 2b 管理端可暂不实现消息管理，但如果实现，参考 Phase 2a 的模式：
- `admin/service/im_manage_service.go`
- 注入 `*gorm.DB` 查询 im_messages + auth_users
- 返回包含发送者用户名的完整 DTO（不要只返回 ID）

---

## 七、Phase 2b 建议的 Task 拆分思路

1. 设计文档 + 数据库迁移（im_conversations, im_conversation_members, im_messages）
2. IM Model + DAO 层
3. IM Service 层（含接口注入设计）
4. IM Controller + 路由 + Wire 集成
5. WebSocket 事件路由机制 + IM 事件处理器
6. 前台会话列表页 + 聊天页
7. 前台 chat Store + API 封装
8. 管理端扩展（可选）
9. 集成测试 + 文档更新
