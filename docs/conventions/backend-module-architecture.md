# Go 后端模块架构规范

> **适用范围**：EchoChat Go 后端（`backend/go-service/`）
> **创建日期**：2026-03-04
> **最后更新**：2026-03-04（基于全量代码审查，以实际代码为准）
> **关联文档**：`docs/conventions/frontend-backend-integration.md`

**核心原则：本文档所有代码模板均直接摘自项目现有代码，编写新模块时必须严格遵循，禁止自创新模式。**

---

## 一、模块分层架构

每个业务模块采用四层架构，目录结构如下：

```
app/{module_name}/
├── controller/
│   └── {module}_controller.go   # HTTP 请求处理
├── service/
│   └── {module}_service.go      # 业务逻辑
├── dao/
│   └── {module}_dao.go          # 数据访问
├── model/
│   └── {module}.go              # 数据模型
├── handler/
│   └── {module}_handler.go      # [可选] WS 事件处理器
├── router.go                    # 路由注册
└── provider.go                  # Wire ProviderSet
```

---

## 二、层间调用规则

```
Controller → Service → DAO → 数据库
                 ↓
           其他模块接口（通过注入的 interface）
                 ↓
           PubSub / Redis
```

- Controller **只处理 HTTP 关注点**：参数绑定、调用 Service、返回响应
- Service **只处理业务逻辑**：权限校验、业务规则、事务编排、推送通知
- DAO **只处理数据存取**：GORM 查询、批量操作、不含业务逻辑
- **禁止跨层调用**：Controller 不能直接调用 DAO，DAO 不能调用 Service

---

## 三、日志 API（以实际代码为准，严禁使用不存在的 API）

项目 logs 包（`pkg/logs`）**只有以下 5 个公开日志方法**：

```go
logs.Debug(ctx, funcName, message, ...zap.Field)
logs.Info(ctx, funcName, message, ...zap.Field)
logs.Warn(ctx, funcName, message, ...zap.Field)
logs.Error(ctx, funcName, message, ...zap.Field)
logs.Fatal(ctx, funcName, message, ...zap.Field)
```

辅助方法：`logs.Init()`, `logs.Sync()`, `logs.GetTraceID()`, `logs.MaskEmail()`

**不存在的 API（严禁调用）**：`LogFunctionEntry`、`LogFunctionExit`、`LogSuccess`、`LogFailure` 等均不存在。

**funcName 命名规则**：`"{层级}.{文件名}.{方法名}"`
- DAO 层：`"dao.conversation_dao.FindPrivateConversation"`
- Service 层：`"service.im_service.SendMessage"`
- Controller 层：`"controller.auth_controller.Register"`

---

## 四、Controller 层代码风格

项目中存在两种 Controller 风格，新模块应根据所属类型选择对应风格：

### 4.1 前台业务模块 Controller（contact/im 模块风格）

**适用于**：contact、im、group、file 等前台用户端业务模块

**特征**：接收器 `ctl`，不记日志，方法级 `handleError`

```go
// 摘自 app/contact/controller/contact_controller.go（实际代码）
package controller

import (
    "strconv"
    "github.com/echochat/backend/app/contact/service"
    "github.com/echochat/backend/app/dto"
    "github.com/echochat/backend/pkg/middleware"
    "github.com/echochat/backend/pkg/utils"
    "github.com/gin-gonic/gin"
)

type ContactController struct {
    contactService *service.ContactService
}

func NewContactController(contactService *service.ContactService) *ContactController {
    return &ContactController{contactService: contactService}
}

// GetFriendList 获取好友列表
// GET /api/v1/contacts?group_id=xx
func (ctl *ContactController) GetFriendList(c *gin.Context) {
    ctx := c.Request.Context()
    userID, ok := middleware.GetCurrentUserID(c)
    if !ok {
        utils.ResponseUnauthorized(c, "无法获取当前用户信息")
        return
    }
    // ... 参数解析 ...
    friends, err := ctl.contactService.GetFriendList(ctx, userID, groupID)
    if err != nil {
        ctl.handleError(c, err, "获取好友列表失败")
        return
    }
    utils.ResponseOK(c, friends)
}

// handleError 统一业务错误映射
func (ctl *ContactController) handleError(c *gin.Context, err error, fallbackMsg ...string) {
    switch err {
    case service.ErrSelfRequest:
        utils.ResponseBadRequest(c, err.Error())
    case service.ErrAlreadyFriend:
        utils.ResponseBadRequest(c, err.Error())
    // ... 覆盖所有已知业务错误 ...
    default:
        msg := "服务器内部错误"
        if len(fallbackMsg) > 0 && fallbackMsg[0] != "" {
            msg = fallbackMsg[0]
        }
        utils.ResponseError(c, msg)
    }
}
```

**关键要点**：
- 接收器变量名：`ctl`
- **不导入** `logs` 和 `zap`，不记日志
- 没有 `funcName` 变量
- 用 `ctx := c.Request.Context()` 获取上下文
- 用 `middleware.GetCurrentUserID(c)` 获取当前用户
- `handleError` 是**方法**（不是包级函数），签名 `(c *gin.Context, err error, fallbackMsg ...string)`
- 响应统一用 `utils.ResponseOK/ResponseBadRequest/ResponseError` 等
- 参数绑定：JSON 用 `c.ShouldBindJSON(&req)`，Query 用 `c.ShouldBindQuery(&req)`

### 4.2 auth/admin 模块 Controller 风格

**适用于**：auth、admin 模块（涉及安全审计和管理操作，需要更详细的日志）

**特征**：接收器 `ctrl`/`ctl`，有 funcName+logs+zap，包级函数 `handleAuthError` 或内联错误处理

```go
// 摘自 app/auth/controller/auth_controller.go（实际代码）
func (ctrl *AuthController) Register(c *gin.Context) {
    funcName := "controller.auth_controller.Register"
    ctx := c.Request.Context()

    var req dto.RegisterRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        logs.Warn(ctx, funcName, "参数校验失败", zap.Error(err))
        utils.ResponseBadRequest(c, "参数校验失败: "+err.Error())
        return
    }

    logs.Info(ctx, funcName, "注册请求",
        zap.String("username", req.Username),
        zap.String("email", logs.MaskEmail(req.Email)),
    )

    resp, err := ctrl.authService.Register(ctx, &req)
    if err != nil {
        handleAuthError(c, err, "注册失败")
        return
    }
    utils.ResponseOK(c, resp)
}
```

---

## 五、Service 层代码风格（统一）

所有模块的 Service 层风格一致。

```go
// 摘自 app/im/service/im_service.go + app/contact/service/contact_service.go（实际代码）
package service

import (
    "context"
    "errors"
    "github.com/echochat/backend/app/xxx/dao"
    "github.com/echochat/backend/app/dto"
    "github.com/echochat/backend/pkg/logs"
    "go.uber.org/zap"
)

// 错误变量在包顶部定义
var (
    ErrNotFriend    = errors.New("对方不是你的好友")
    ErrEmptyContent = errors.New("消息内容不能为空")
    // ...
)

type IMService struct {
    convDAO       *dao.ConversationDAO
    msgDAO        *dao.MessageDAO
    // ...
}

func NewIMService(...) *IMService {
    return &IMService{...}
}

// SendMessage 发送消息
func (s *IMService) SendMessage(ctx context.Context, senderID int64, req *dto.SendMessageRequest) (*dto.MessageDTO, error) {
    funcName := "service.im_service.SendMessage"
    logs.Info(ctx, funcName, "发送消息",
        zap.Int64("sender_id", senderID),
        zap.Int64("conversation_id", req.ConversationID))

    // 业务逻辑...
    if err != nil {
        logs.Error(ctx, funcName, "操作失败", zap.Error(err))
        return nil, err
    }
    return result, nil
}
```

**关键要点**：
- 接收器变量名：`s`
- 每个公开方法开头声明 `funcName` 并记一次 Info/Debug 日志
- 错误时记 `logs.Error` 日志
- 错误定义为包级 `var ErrXxx = errors.New("中文描述")`
- 跨模块依赖通过 interface 注入（定义在 Service 包内）

---

## 六、DAO 层代码风格（统一）

```go
// 摘自 app/im/dao/conversation_dao.go（实际代码）
type ConversationDAO struct {
    db *gorm.DB
}

func NewConversationDAO(db *gorm.DB) *ConversationDAO {
    return &ConversationDAO{db: db}
}

func (d *ConversationDAO) FindPrivateConversation(ctx context.Context, userID, targetUserID int64) (*model.Conversation, error) {
    funcName := "dao.conversation_dao.FindPrivateConversation"
    logs.Debug(ctx, funcName, "查找单聊会话",
        zap.Int64("user_id", userID), zap.Int64("target_user_id", targetUserID))

    var conv model.Conversation
    err := d.db.WithContext(ctx).
        Raw(`SELECT ...`, userID, targetUserID).
        Scan(&conv).Error

    if err != nil {
        logs.Error(ctx, funcName, "查找单聊会话失败", zap.Error(err))
        return nil, err
    }
    if conv.ID == 0 {
        return nil, nil
    }
    return &conv, nil
}
```

**关键要点**：
- 接收器变量名：`d`
- 所有 DB 操作使用 `d.db.WithContext(ctx)`
- 查询方法开头记 `logs.Debug`，写操作记 `logs.Info`
- 错误时记 `logs.Error`

---

## 七、Router 代码风格

```go
// 摘自 app/contact/router.go（实际代码）
package contact

import (
    "github.com/echochat/backend/app/contact/controller"
    "github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, ctrl *controller.ContactController, jwtAuth gin.HandlerFunc) {
    authed := r.Group("/api/v1")
    authed.Use(jwtAuth)
    {
        authed.GET("/contacts", ctrl.GetFriendList)
        authed.POST("/contacts/request", ctrl.SendFriendRequest)
        // ...
    }
}
```

**关键要点**：
- 函数签名：`RegisterRoutes(r *gin.Engine, ctrl *controller.XxxController, jwtAuth gin.HandlerFunc)`
- 参数名用 `r`（不是 `engine`）
- 路由组用 `r.Group(...)` + `.Use(jwtAuth)`

---

## 八、Provider 代码风格

```go
// 摘自 app/contact/provider.go（简洁风格）
package contact

import (
    "github.com/echochat/backend/app/contact/controller"
    "github.com/echochat/backend/app/contact/dao"
    "github.com/echochat/backend/app/contact/service"
    "github.com/google/wire"
)

var ContactSet = wire.NewSet(
    dao.NewFriendshipDAO,
    dao.NewFriendGroupDAO,
    service.NewContactService,
    controller.NewContactController,
)
```

**ProviderSet 命名**：`{ModuleName}Set`（如 `AuthSet`、`ContactSet`、`IMSet`、`FileSet`、`GroupSet`）

---

## 九、常量定义风格

```go
// 摘自 app/constants/im.go（实际代码）
package constants

const (
    ConversationTypePrivate = 1
    ConversationTypeGroup   = 2
)

var ConversationTypeMap = map[int]string{
    ConversationTypePrivate: "单聊",
    ConversationTypeGroup:   "群聊",
}
```

**关键要点**：
- 常量命名 camelCase：`GroupStatusNormal`（不是 `GROUP_STATUS_NORMAL`）
- 每组常量配套一个 `XxxMap` 中文映射
- 文件按模块拆分：`im.go`、`contact.go`、`group.go`

---

## 十、DTO 定义风格

```go
// 摘自 app/dto/im_dto.go（实际代码）
package dto

type SendMessageRequest struct {
    ConversationID int64  `json:"conversation_id"`
    TargetUserID   int64  `json:"target_user_id"`
    Type           int    `json:"type"`
    Content        string `json:"content"`
    ClientMsgID    string `json:"client_msg_id"`
}

type MessageDTO struct {
    ID             int64  `json:"id"`
    ConversationID int64  `json:"conversation_id"`
    // ...
}
```

**关键要点**：
- 按模块分文件：`im_dto.go`、`contact_dto.go`、`admin_dto.go`、`group_dto.go`
- Request 用 `json` tag（POST body）或 `form` tag（GET query）
- Response 用 `json` tag + `omitempty` 可选
- 时间字段在 DTO 中用 `string` 类型（格式化后的 `"2006-01-02 15:04:05"`）

---

## 十一、Model 定义风格

```go
// 摘自 app/im/model/conversation.go（实际代码）
package model

import "time"

type Conversation struct {
    ID            int64      `json:"id" gorm:"primaryKey;autoIncrement"`
    Type          int        `json:"type" gorm:"not null;default:1"`
    CreatorID     int64      `json:"creator_id" gorm:"not null"`
    LastMessageID *int64     `json:"last_message_id"`
    CreatedAt     time.Time  `json:"created_at" gorm:"not null;autoCreateTime;type:timestamp(0)"`
    UpdatedAt     time.Time  `json:"updated_at" gorm:"not null;autoUpdateTime;type:timestamp(0)"`
}

func (Conversation) TableName() string {
    return "im_conversations"
}
```

**关键要点**：
- 必须有 `TableName()` 方法
- 时间字段用 `time.Time`，GORM tag 含 `type:timestamp(0)`
- 可选字段用指针类型（`*int64`、`*time.Time`、`*string`）
- 每个字段必须有注释说明用途

---

## 十二、跨模块接口注入模式

模块间通信通过 **interface injection** 实现，禁止直接 import 其他模块包。

### 标准流程

```
步骤1: 在消费方 Service 中定义 interface
步骤2: 在提供方 DAO 中实现该接口
步骤3: 在 app/provider/wire.go 中用 wire.Bind 绑定
步骤4: 更新 wire_gen.go
```

### 已有接口注入清单

| 接口 | 定义方 | 实现方 | 用途 |
|------|--------|--------|------|
| FriendIDsGetter | ws/handler | contact/dao | 获取用户好友 ID 列表 |
| FriendChecker | im/service | contact/dao | 检查是否为好友 |
| UserInfoGetter | im/service | contact/dao | 获取用户信息 |
| OnlineChecker | contact/service | ws/service | 检查在线状态 |
| TokenValidator | ws/handler | auth/service | WS 连接 Token 校验 |

---

## 十三、批量查询优化

获取关联信息时，**必须使用批量查询 + Map 映射，严禁 N+1 查询**。

---

## 十四、系统消息规范（Phase 2c+）

系统消息写入 `im_messages` 表，`type=10`（`MessageTypeSystem`），`sender_id=0`（表示系统），`content` 使用**纯文本格式**。前端居中显示、灰色小字体、无头像、无气泡。

---

## 十五、依赖管理

- 添加新依赖前必须检查 `go.mod` 中的 Go 版本（当前 `go 1.23.12`）
- 选择与当前 Go 版本兼容的包版本，禁止触发 Go 工具链自动升级
- 优先复用 `go.mod` 中已有的间接依赖
