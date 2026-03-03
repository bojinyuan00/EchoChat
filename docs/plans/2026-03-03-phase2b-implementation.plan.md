# Phase 2b 即时通讯消息系统（单聊）实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 基于 Phase 2a WebSocket 基础设施，实现单聊即时通讯核心功能

**Architecture:** 新建 `app/im/` 模块（MVC 分层），扩展 `pkg/ws/` 事件路由，前端新增 chat Store + 4 个页面

**Tech Stack:** Go (Gin + GORM + Wire) / Redis / PostgreSQL / Vue 3 (uni-app + Pinia) / WebSocket

**Design Doc:** `docs/plans/2026-03-03-phase2b-design.md`

---

## Task 0: 设计文档 + 新分支 + 数据库表迁移

**Files:**
- 已完成: `docs/plans/2026-03-03-phase2b-design.md`
- Modify: `backend/go-service/cmd/server/main.go`（AutoMigrate 新表）

**Step 1: 创建新分支**

```bash
cd /Users/bojinyuan/Documents/workspace/webrtc/EchoChat
git checkout -b feature/phase2b-instant-messaging
```

**Step 2: 创建 IM 模型文件**

Create: `backend/go-service/app/im/model/conversation.go`

```go
package model

import "time"

type Conversation struct {
    ID               int64      `json:"id" gorm:"primaryKey;autoIncrement"`
    Type             int        `json:"type" gorm:"not null;default:1;comment:1=单聊 2=群聊"`
    CreatorID        int64      `json:"creator_id" gorm:"not null"`
    LastMessageID    *int64     `json:"last_message_id"`
    LastMsgContent   string     `json:"last_msg_content" gorm:"type:text;default:''"`
    LastMsgTime      *time.Time `json:"last_msg_time"`
    LastMsgSenderID  *int64     `json:"last_msg_sender_id"`
    CreatedAt        time.Time  `json:"created_at"`
    UpdatedAt        time.Time  `json:"updated_at"`
}

func (Conversation) TableName() string { return "im_conversations" }
```

Create: `backend/go-service/app/im/model/conversation_member.go`

```go
package model

import "time"

type ConversationMember struct {
    ID              int64     `json:"id" gorm:"primaryKey;autoIncrement"`
    ConversationID  int64     `json:"conversation_id" gorm:"not null;uniqueIndex:idx_conv_user"`
    UserID          int64     `json:"user_id" gorm:"not null;uniqueIndex:idx_conv_user"`
    IsPinned        bool      `json:"is_pinned" gorm:"default:false"`
    IsDeleted       bool      `json:"is_deleted" gorm:"default:false"`
    UnreadCount     int       `json:"unread_count" gorm:"default:0"`
    LastReadMsgID   int64     `json:"last_read_msg_id" gorm:"default:0"`
    CreatedAt       time.Time `json:"created_at"`
    UpdatedAt       time.Time `json:"updated_at"`
}

func (ConversationMember) TableName() string { return "im_conversation_members" }
```

Create: `backend/go-service/app/im/model/message.go`

```go
package model

import "time"

type Message struct {
    ID              int64     `json:"id" gorm:"primaryKey;autoIncrement"`
    ConversationID  int64     `json:"conversation_id" gorm:"not null;index:idx_msg_conv_time;index:idx_msg_conv_id"`
    SenderID        int64     `json:"sender_id" gorm:"not null"`
    Type            int       `json:"type" gorm:"not null;default:1;comment:1=文本"`
    Content         string    `json:"content" gorm:"type:text;not null"`
    Extra           *string   `json:"extra" gorm:"type:jsonb"`
    Status          int       `json:"status" gorm:"not null;default:1;comment:1=正常 2=已撤回 3=已删除"`
    ClientMsgID     string    `json:"client_msg_id" gorm:"size:64;default:''"`
    CreatedAt       time.Time `json:"created_at" gorm:"index:idx_msg_conv_time"`
}

func (Message) TableName() string { return "im_messages" }
```

**Step 3: 在 main.go 添加 AutoMigrate**

在 `cmd/server/main.go` 中（Wire 初始化之后），新增 IM 表 AutoMigrate：

```go
// 在 app, err := provider.InitializeApp(cfg) 之后添加
import imModel "github.com/echochat/backend/app/im/model"

if err := app.DB.AutoMigrate(
    &imModel.Conversation{},
    &imModel.ConversationMember{},
    &imModel.Message{},
); err != nil {
    logs.Fatal(ctx, "main", "IM 表迁移失败", zap.Error(err))
}
```

**Step 4: 创建 Redis 索引 SQL（手动执行）**

全文搜索索引需手动执行（GORM AutoMigrate 不支持 GIN 索引）：

```sql
CREATE INDEX IF NOT EXISTS idx_im_messages_content_search 
ON im_messages USING gin(to_tsvector('simple', content));
```

**Step 5: 验证编译**

```bash
cd backend/go-service && go build ./...
```

Expected: 编译成功

**Step 6: Commit**

```bash
git add -A && git commit -m "feat(im): Task 0 - IM 模型定义 + 数据库表迁移"
```

---

## Task 1: WS 事件路由表机制

**Files:**
- Modify: `backend/go-service/pkg/ws/hub.go`（添加事件路由表）
- Modify: `backend/go-service/pkg/ws/message.go`（添加 MessageHandler 类型导出）
- Modify: `backend/go-service/app/ws/handler.go`（onMessage 优先查路由表）

**Step 1: 在 hub.go 添加事件路由表**

Hub 新增字段和方法：

```go
// EventHandler WS 事件处理函数签名
type EventHandler func(client *Client, msg *Message)

// Hub 新增字段
type Hub struct {
    // ...现有字段
    eventHandlers map[string]EventHandler
    ehMu          sync.RWMutex  // 保护 eventHandlers
}

// NewHub 修改：初始化 eventHandlers
func NewHub() *Hub {
    return &Hub{
        clients:       make(map[int64]*Client),
        register:      make(chan *Client, 256),
        unregister:    make(chan *Client, 256),
        stopCh:        make(chan struct{}),
        eventHandlers: make(map[string]EventHandler),
    }
}

// RegisterEvent 注册事件处理器（启动时调用，非热路径）
func (h *Hub) RegisterEvent(event string, handler EventHandler) {
    h.ehMu.Lock()
    defer h.ehMu.Unlock()
    h.eventHandlers[event] = handler
}

// DispatchEvent 分发事件到注册的处理器
// 返回 true 表示已处理，false 表示无匹配处理器
func (h *Hub) DispatchEvent(client *Client, msg *Message) bool {
    h.ehMu.RLock()
    handler, ok := h.eventHandlers[msg.Event]
    h.ehMu.RUnlock()
    if !ok {
        return false
    }
    handler(client, msg)
    return true
}
```

**Step 2: 修改 handler.go 的 onMessage**

在 `createReadHandler` 的 switch 之前，先尝试路由表分发：

```go
func (h *Handler) createReadHandler(userID int64) ws.MessageHandler {
    return func(client *ws.Client, msg *ws.Message) {
        // 先尝试事件路由表
        if h.hub.DispatchEvent(client, msg) {
            return
        }

        // 内置事件 fallback
        switch msg.Event {
        case "heartbeat":
            // ...现有逻辑
        default:
            // ...现有逻辑
        }
    }
}
```

**Step 3: 验证编译**

```bash
cd backend/go-service && go build ./...
```

**Step 4: Commit**

```bash
git add -A && git commit -m "feat(ws): Task 1 - Hub 事件路由表机制"
```

---

## Task 2: IM DAO 层

**Files:**
- Create: `backend/go-service/app/im/dao/conversation_dao.go`
- Create: `backend/go-service/app/im/dao/message_dao.go`

**Step 1: 创建 ConversationDAO**

Create: `backend/go-service/app/im/dao/conversation_dao.go`

核心方法：
- `FindPrivateConversation(ctx, userID, targetUserID)` — 查找两人的单聊会话
- `CreateWithMembers(ctx, conv, memberIDs)` — 事务创建会话 + 成员
- `GetUserConversations(ctx, userID)` — 获取用户会话列表（JOIN members + conversations）
- `GetMember(ctx, conversationID, userID)` — 获取成员信息
- `UpdateMember(ctx, member)` — 更新成员信息（置顶、未读数等）
- `SoftDeleteMember(ctx, conversationID, userID)` — 软删除会话
- `UpdateLastMessage(ctx, conversationID, msgID, content, senderID, time)` — 更新最后消息
- `IncrementUnread(ctx, conversationID, userID)` — 未读数 +1
- `ClearUnread(ctx, conversationID, userID)` — 清零未读
- `GetUnreadConversations(ctx, userID)` — 获取有未读的会话列表

**Step 2: 创建 MessageDAO**

Create: `backend/go-service/app/im/dao/message_dao.go`

核心方法：
- `Create(ctx, message)` — 写入消息
- `GetByID(ctx, id)` — 查询单条消息
- `GetByConversation(ctx, conversationID, beforeID, limit)` — 游标分页查消息
- `UpdateStatus(ctx, id, status)` — 更新消息状态（撤回）
- `DeleteByConversation(ctx, conversationID)` — 清空会话消息
- `SearchMessages(ctx, userID, keyword, limit)` — 全局搜索（JOIN members 确保权限）
- `FindByClientMsgID(ctx, conversationID, clientMsgID)` — 幂等查询

**Step 3: 验证编译**

```bash
cd backend/go-service && go build ./...
```

**Step 4: Commit**

```bash
git add -A && git commit -m "feat(im): Task 2 - IM DAO 层（会话 + 消息）"
```

---

## Task 3: IM Service 核心业务

**Files:**
- Create: `backend/go-service/app/im/service/im_service.go`
- Create: `backend/go-service/app/im/service/interfaces.go`（接口定义）

**Step 1: 创建接口定义文件**

Create: `backend/go-service/app/im/service/interfaces.go`

```go
package service

import (
    "context"
    authModel "github.com/echochat/backend/app/auth/model"
)

// FriendChecker 好友关系校验接口（contact.FriendshipDAO 隐式实现）
type FriendChecker interface {
    IsFriend(ctx context.Context, userID, targetID int64) (bool, error)
}

// UserInfoGetter 用户信息批量查询接口（contact.FriendshipDAO 隐式实现）
type UserInfoGetter interface {
    GetUsersByIDs(ctx context.Context, userIDs []int64) ([]authModel.User, error)
}
```

**Step 2: 创建 IMService**

Create: `backend/go-service/app/im/service/im_service.go`

IMService 注入：`ConversationDAO`、`MessageDAO`、`*ws.PubSub`、`FriendChecker`、`UserInfoGetter`

核心方法（参考设计文档第七节）：
- `SendMessage(ctx, senderID, targetUserID, content, msgType, clientMsgID)` — 核心链路：校验好友 → 查找/创建会话 → 写消息 → 更新 last_message → 增加未读 → 推送
- `RecallMessage(ctx, userID, messageID, conversationID)` — 校验 2 分钟 + 更新状态 + 推送
- `GetConversations(ctx, userID)` — 会话列表（含对方用户信息）
- `GetMessages(ctx, userID, conversationID, beforeID, limit)` — 历史消息 + 发送者信息
- `PinConversation(ctx, userID, conversationID, isPinned)` — 置顶
- `DeleteConversation(ctx, userID, conversationID)` — 软删除
- `ClearMessages(ctx, userID, conversationID)` — 清空
- `MarkAsRead(ctx, userID, conversationID)` — 标记已读 + 清 Redis
- `SearchMessages(ctx, userID, keyword)` — 全局搜索
- `PushOfflineMessages(ctx, userID)` — 查未读会话 → 推送 im.conversation.unread
- `FindOrCreateConversation(ctx, userID, targetUserID)` — 内部方法

DTO 定义放在 `app/dto/im_dto.go`（新建）：
- `ConversationItem` — 会话列表项
- `MessageItem` — 消息项
- `SearchResult` — 搜索结果（按会话分组）

**Step 3: 创建 IM DTO**

Create: `backend/go-service/app/dto/im_dto.go`

```go
package dto

type ConversationItem struct {
    ID              int64  `json:"id"`
    Type            int    `json:"type"`
    PeerUserID      int64  `json:"peer_user_id"`
    PeerNickname    string `json:"peer_nickname"`
    PeerAvatar      string `json:"peer_avatar"`
    PeerUsername    string `json:"peer_username"`
    LastMsgContent  string `json:"last_msg_content"`
    LastMsgTime     string `json:"last_msg_time"`
    LastMsgSenderID int64  `json:"last_msg_sender_id"`
    UnreadCount     int    `json:"unread_count"`
    IsPinned        bool   `json:"is_pinned"`
}

type MessageItem struct {
    ID           int64  `json:"id"`
    SenderID     int64  `json:"sender_id"`
    SenderName   string `json:"sender_name"`
    SenderAvatar string `json:"sender_avatar"`
    Type         int    `json:"type"`
    Content      string `json:"content"`
    Status       int    `json:"status"`
    ClientMsgID  string `json:"client_msg_id"`
    CreatedAt    string `json:"created_at"`
}

type MessageSearchGroup struct {
    ConversationID int64         `json:"conversation_id"`
    PeerNickname   string        `json:"peer_nickname"`
    PeerAvatar     string        `json:"peer_avatar"`
    Messages       []MessageItem `json:"messages"`
}
```

**Step 4: 添加 IM 常量**

Create: `backend/go-service/app/constants/im_constants.go`

```go
package constants

const (
    ConversationTypePrivate = 1
    ConversationTypeGroup   = 2

    MessageTypeText  = 1

    MessageStatusNormal   = 1
    MessageStatusRecalled = 2
    MessageStatusDeleted  = 3

    MessageRecallTimeLimit = 2 * 60  // 撤回时限（秒）
)
```

**Step 5: 验证编译**

```bash
cd backend/go-service && go build ./...
```

**Step 6: Commit**

```bash
git add -A && git commit -m "feat(im): Task 3 - IM Service + DTO + 常量定义"
```

---

## Task 4: IM WS 事件处理器 + 离线消息推送

**Files:**
- Create: `backend/go-service/app/im/handler/message_handler.go`
- Modify: `backend/go-service/app/ws/handler.go`（离线消息推送接口 + 连接回调）
- Modify: `backend/go-service/app/ws/online_service.go`（添加 OfflineMessagePusher 接口）

**Step 1: 创建 IM 事件处理器**

Create: `backend/go-service/app/im/handler/message_handler.go`

```go
package handler

// MessageHandler IM 模块的 WebSocket 事件处理器
// 负责处理：im.message.send、im.message.recall、im.typing.start/stop、im.conversation.sync

type MessageHandler struct {
    imService *service.IMService
    hub       *ws.Hub
}

// RegisterEvents 向 Hub 注册 IM 事件处理器
func (h *MessageHandler) RegisterEvents(hub *ws.Hub) {
    hub.RegisterEvent("im.message.send", h.HandleSendMessage)
    hub.RegisterEvent("im.message.recall", h.HandleRecallMessage)
    hub.RegisterEvent("im.typing.start", h.HandleTypingStart)
    hub.RegisterEvent("im.typing.stop", h.HandleTypingStop)
    hub.RegisterEvent("im.conversation.sync", h.HandleConversationSync)
}
```

每个 Handler 方法：解析 msg.Data → 调用 IMService → 返回 ACK Response

**Step 2: 添加离线消息推送接口**

在 `app/ws/online_service.go` 中定义接口：

```go
// OfflineMessagePusher 离线消息推送接口（由 IM Service 实现）
type OfflineMessagePusher interface {
    PushOfflineMessages(ctx context.Context, userID int64) error
}
```

Handler 新增 `offlinePusher OfflineMessagePusher` 字段，在 `Upgrade` 中 WS 连接成功后调用。

**Step 3: 修改 Handler.Upgrade 添加离线推送调用**

在 `handler.go` Upgrade 方法的 `h.onlineService.UserOnline()` 之后，添加：

```go
if h.offlinePusher != nil {
    go h.offlinePusher.PushOfflineMessages(c.Request.Context(), claims.UserID)
}
```

**Step 4: 验证编译**

```bash
cd backend/go-service && go build ./...
```

**Step 5: Commit**

```bash
git add -A && git commit -m "feat(im): Task 4 - IM WS 事件处理器 + 离线消息推送"
```

---

## Task 5: IM REST Controller + Router + Wire 集成

**Files:**
- Create: `backend/go-service/app/im/controller/im_controller.go`
- Create: `backend/go-service/app/im/router.go`
- Create: `backend/go-service/app/im/provider.go`
- Modify: `backend/go-service/app/provider/provider.go`（App 新增 IM 字段）
- Modify: `backend/go-service/app/provider/wire.go`（添加 IMSet）
- Regenerate: `backend/go-service/app/provider/wire_gen.go`
- Modify: `backend/go-service/router/router.go`（注册 IM 路由）

**Step 1: 创建 IM Controller**

Create: `backend/go-service/app/im/controller/im_controller.go`

```go
package controller

type IMController struct {
    imService *service.IMService
}

func NewIMController(svc *service.IMService) *IMController {
    return &IMController{imService: svc}
}
```

实现 7 个 REST handler：
- `GetConversations` — GET /conversations
- `GetMessages` — GET /conversations/:id/messages
- `PinConversation` — PUT /conversations/:id/pin
- `DeleteConversation` — DELETE /conversations/:id
- `ClearMessages` — DELETE /conversations/:id/messages
- `MarkAsRead` — PUT /conversations/:id/read
- `SearchMessages` — GET /messages/search

**Step 2: 创建 IM Router**

Create: `backend/go-service/app/im/router.go`

```go
package im

func RegisterRoutes(r *gin.Engine, ctrl *controller.IMController, jwtAuth gin.HandlerFunc) {
    authed := r.Group("/api/v1/im")
    authed.Use(jwtAuth)
    {
        authed.GET("/conversations", ctrl.GetConversations)
        authed.GET("/conversations/:id/messages", ctrl.GetMessages)
        authed.PUT("/conversations/:id/pin", ctrl.PinConversation)
        authed.DELETE("/conversations/:id", ctrl.DeleteConversation)
        authed.DELETE("/conversations/:id/messages", ctrl.ClearMessages)
        authed.PUT("/conversations/:id/read", ctrl.MarkAsRead)
        authed.GET("/messages/search", ctrl.SearchMessages)
    }
}
```

**Step 3: 创建 IM Provider**

Create: `backend/go-service/app/im/provider.go`

```go
package im

var IMSet = wire.NewSet(
    dao.NewConversationDAO,
    dao.NewMessageDAO,
    service.NewIMService,
    controller.NewIMController,
    handler.NewMessageHandler,
)
```

**Step 4: 修改 provider.go — App 新增字段**

在 `App` struct 和 `NewApp` 中添加：
- `IMController *imController.IMController`
- `IMHandler *imHandler.MessageHandler`

**Step 5: 修改 wire.go 和重新生成 wire_gen.go**

添加 `IMSet` 到 Wire injector，添加接口绑定：
- `wire.Bind(new(imService.FriendChecker), new(*friendshipDAO))`
- `wire.Bind(new(imService.UserInfoGetter), new(*friendshipDAO))`
- `wire.Bind(new(wsApp.OfflineMessagePusher), new(*imService.IMService))`

```bash
cd backend/go-service && wire ./app/provider/
```

**Step 6: 修改 router.go**

在 `router/router.go` 的 Setup 函数中添加：

```go
import imApp "github.com/echochat/backend/app/im"

imApp.RegisterRoutes(engine, app.IMController, jwtAuth)
```

并在 Router.Setup 中注册 IM 事件处理器：

```go
app.IMHandler.RegisterEvents(app.Hub)
```

**Step 7: 验证编译**

```bash
cd backend/go-service && go build ./...
```

**Step 8: 启动后端验证**

```bash
cd backend/go-service && go run cmd/server/main.go
```

Expected: 服务启动成功，im_conversations / im_conversation_members / im_messages 表自动创建

**Step 9: Commit**

```bash
git add -A && git commit -m "feat(im): Task 5 - REST Controller + Router + Wire 完整集成"
```

---

## Task 6: 前台 chat Store + API 封装 + WS 事件集成

**Files:**
- Create: `frontend/src/api/chat.js`
- Create: `frontend/src/store/chat.js`
- Modify: `frontend/src/services/websocket.js`（如需添加 send 到 store）
- Modify: `frontend/src/components/CustomTabBar.vue`（未读 badge）

**Step 1: 创建 chat API**

Create: `frontend/src/api/chat.js`

封装 7 个 REST API：
- `getConversations()` — GET /api/v1/im/conversations
- `getMessages(conversationId, beforeId, limit)` — GET /api/v1/im/conversations/:id/messages
- `pinConversation(conversationId, isPinned)` — PUT /api/v1/im/conversations/:id/pin
- `deleteConversation(conversationId)` — DELETE /api/v1/im/conversations/:id
- `clearMessages(conversationId)` — DELETE /api/v1/im/conversations/:id/messages
- `markAsRead(conversationId)` — PUT /api/v1/im/conversations/:id/read
- `searchMessages(keyword)` — GET /api/v1/im/messages/search

**Step 2: 创建 chat Store**

Create: `frontend/src/store/chat.js`

Pinia Store，包含：
- 状态：conversations, currentConversation, messages, totalUnread, typingUsers, pendingMessages
- Actions：loadConversations, sendMessage, loadHistory, markAsRead, recallMessage, pinConversation, deleteConversation, clearMessages, searchMessages
- WS 事件注册：initWsListeners()
  - `im.message.new` → 更新会话列表 + 消息列表 + 未读数
  - `im.message.send.ack` → 更新 pending 消息状态为已发送
  - `im.message.recalled` → 标记消息为已撤回
  - `im.typing.notify` → 更新 typingUsers
  - `im.conversation.unread` → 初始化会话未读数

**Step 3: 修改 CustomTabBar 添加未读 badge**

修改 `frontend/src/components/CustomTabBar.vue`：
- 引入 `useChatStore`
- 消息 Tab 显示 `chatStore.totalUnread` badge（红色圆点/数字）

**Step 4: 验证编译**

```bash
cd frontend && npm run build:h5 2>&1 | head -20
```

Expected: 编译成功

**Step 5: Commit**

```bash
git add -A && git commit -m "feat(frontend): Task 6 - chat Store + API + WS 事件 + TabBar badge"
```

---

## Task 7: 前台会话列表页 + 聊天页

**前置操作：** 使用 ui-ux-pro-max 技能包进行页面设计

**Files:**
- Modify: `frontend/src/pages/chat/index.vue`（改造为会话列表）
- Create: `frontend/src/pages/chat/conversation.vue`（聊天页）
- Modify: `frontend/src/pages.json`（添加 conversation 页面路由）

**Step 1: 读取 ui-ux-pro-max 技能包**

```bash
npx openskills read ui-ux-pro-max
```

使用脚本搜索聊天/消息页面的设计参考。

**Step 2: 注册新页面路由**

在 `pages.json` 的 pages 数组中添加：

```json
{
    "path": "pages/chat/conversation",
    "style": {
        "navigationBarTitleText": "聊天",
        "navigationStyle": "custom"
    }
}
```

**Step 3: 实现会话列表页 (chat/index.vue)**

主要功能：
- 搜索栏（点击跳转 search 页）
- 会话列表：v-for conversations，展示头像/昵称/最后消息/时间/未读badge
- 排序：置顶在前，再按 lastMsgTime 降序
- 长按操作菜单（uni.showActionSheet）：置顶、删除、清空
- 点击跳转：`uni.navigateTo({ url: '/pages/chat/conversation?conversationId=X&userId=Y' })`
- 空状态："暂无消息"
- 使用 `CustomTabBar :current="0"`

**Step 4: 实现聊天页 (chat/conversation.vue)**

主要功能：
- 自定义顶部导航：对方昵称、"正在输入…"、设置图标
- 消息列表（scroll-view，scroll-into-view 自动滚底）
- 上拉加载历史（loadHistory with beforeId）
- 消息气泡：
  - 自己的消息右对齐（蓝色背景）
  - 对方消息左对齐（灰色背景）
  - 已撤回消息灰色提示
- 消息状态：发送中(spinner)、已发送(✓)、失败(重试图标)
- 长按消息 → 撤回（仅自己的、2分钟内）
- 底部输入区：textarea + 发送按钮
- 输入时触发 im.typing.start（节流 3s），停止输入 3s 后发 im.typing.stop
- onLoad：通过 userId 参数加载/进入会话
- onUnload：清理 typing 状态

**Step 5: 验证编译**

```bash
cd frontend && npm run build:h5 2>&1 | head -20
```

**Step 6: Commit**

```bash
git add -A && git commit -m "feat(frontend): Task 7 - 会话列表页 + 聊天页（ui-ux-pro-max）"
```

---

## Task 8: 前台会话设置页 + 消息搜索页 + 联系人模块改造

**前置操作：** 使用 ui-ux-pro-max 技能包

**Files:**
- Create: `frontend/src/pages/chat/settings.vue`
- Create: `frontend/src/pages/chat/search.vue`
- Modify: `frontend/src/pages.json`（添加 settings/search 路由）
- Modify: `frontend/src/pages/contact/detail.vue`（sendMessage 改为跳转）
- Modify: `frontend/src/pages/contact/index.vue`（列表项点击进入聊天）

**Step 1: 注册新页面路由**

在 `pages.json` 添加：

```json
{ "path": "pages/chat/settings", "style": { "navigationBarTitleText": "聊天设置" } },
{ "path": "pages/chat/search", "style": { "navigationBarTitleText": "搜索消息", "navigationStyle": "custom" } }
```

**Step 2: 实现会话设置页 (chat/settings.vue)**

- 对方信息卡片（头像 + 昵称 + 用户名）
- 置顶聊天 Switch
- 清空聊天记录按钮（uni.showModal 二次确认）
- 删除会话按钮（uni.showModal 二次确认 → navigateBack）

**Step 3: 实现消息搜索页 (chat/search.vue)**

- 搜索输入框（防抖 300ms）
- 搜索结果按会话分组（v-for groups → 会话名 + 消息列表）
- 点击消息跳转到聊天页

**Step 4: 改造好友详情页 (contact/detail.vue)**

```javascript
const sendMessage = () => {
  uni.navigateTo({ url: `/pages/chat/conversation?userId=${userId.value}` })
}
```

**Step 5: 改造好友列表页 (contact/index.vue)**

好友列表项添加点击跳转聊天的快捷入口（或双击/右侧图标）。

**Step 6: 验证编译**

```bash
cd frontend && npm run build:h5 2>&1 | head -20
```

**Step 7: Commit**

```bash
git add -A && git commit -m "feat(frontend): Task 8 - 设置页 + 搜索页 + 联系人联动"
```

---

## Task 9: 集成测试 + 文档更新 + 代码审查

**Files:**
- Modify: `docs/progress/CURRENT_STATUS.md`
- Modify: `docs/plans/2026-03-03-phase2b-design.md`（状态更新）
- Modify: `docs/plans/2026-02-27-echochat-system-design.md`
- Modify: `docs/architecture/system-architecture.md`（如存在）
- Modify: `.cursor/rules/project-context.mdc`
- Create: `docs/api/frontend/im.md`（IM API 文档）
- Create: `docs/api/websocket-im.md`（WS IM 事件文档）

**Step 1: 三端编译验证**

```bash
cd backend/go-service && go build ./...
cd frontend && npm run build:h5
cd admin && npm run build
```

Expected: 全部编译成功

**Step 2: 启动后端 + 前端功能测试**

```bash
cd backend/go-service && go run cmd/server/main.go &
cd frontend && npm run dev:h5
```

使用 Playwright MCP 验证核心流程：
1. 用 testuser1 登录 → 联系人 → 点击好友 → 发消息跳转聊天页
2. 发送一条消息 → 收到 ACK
3. 用另一个账号登录 → 会话列表显示新消息 → 进入聊天 → 看到消息
4. 撤回消息 → 对方看到撤回提示
5. 消息搜索 → 输入关键词 → 显示结果

**Step 3: 更新项目文档**

按项目规则（文档自动同步规则）更新：
- `CURRENT_STATUS.md`：Phase 2b Task 状态表
- `project-context.mdc`：当前进度更新
- 总体设计文档：API 列表更新
- 新增 IM API 文档
- 新增 WS IM 事件文档

**Step 4: 代码审查**

使用 `code-reviewer` 子代理进行结构化审查。

**Step 5: Commit**

```bash
git add -A && git commit -m "docs: Task 9 - Phase 2b 集成测试通过 + 文档同步"
```

---

## 执行说明

本计划共 **10 个 Task**，建议执行方式：

1. **Task 0-5（后端）**：按顺序执行，每个 Task 完成后编译验证 + 提交
2. **Task 6-8（前端）**：依赖后端 API 就绪，前端页面使用 ui-ux-pro-max 设计
3. **Task 9（集成）**：全端验证 + 文档更新

预计工作量：后端 6 个 Task + 前端 3 个 Task + 集成 1 个 Task
