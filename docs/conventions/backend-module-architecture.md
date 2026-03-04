# Go 后端模块架构规范

> **适用范围**：EchoChat Go 后端（`backend/go-service/`）
> **创建日期**：2026-03-04
> **最后更新**：2026-03-04（Phase 2c 设计阶段整理）
> **关联文档**：`docs/conventions/frontend-backend-integration.md`

---

## 一、模块分层架构

每个业务模块采用四层架构，目录结构如下：

```
app/{module_name}/
├── controller/
│   └── {module}_controller.go   # HTTP 请求处理（参数绑定/校验 → 调用 Service → 响应）
├── service/
│   └── {module}_service.go      # 业务逻辑（事务协调、权限校验、多 DAO 编排）
├── dao/
│   └── {module}_dao.go          # 数据访问（GORM 操作，不含业务逻辑）
├── model/
│   └── {module}.go              # 数据模型（GORM 结构体，对应数据库表）
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

## 三、跨模块接口注入模式

模块间通信通过 **interface injection** 实现，禁止直接 import 其他模块包。

### 3.1 标准流程

```
步骤1: 在消费方 Service 中定义 interface（如 im/service → GroupMemberChecker）
步骤2: 在提供方 DAO 中实现该接口（如 group/dao.GroupDAO）
步骤3: 在 app/provider/wire.go 中用 wire.Bind 绑定接口和实现
步骤4: 重新生成 wire_gen.go
```

### 3.2 接口命名约定

| 接口类型 | 命名模式 | 示例 |
|---------|---------|------|
| 数据查询 | `{Entity}{Action}er` | `FriendIDsGetter`, `GroupInfoGetter` |
| 状态检查 | `{Entity}{State}Checker` | `GroupMemberChecker`, `OnlineChecker` |
| 操作执行 | `{Entity}{Action}er` | `OfflineMessagePusher` |

### 3.3 已有接口注入清单

| 接口 | 定义方 | 实现方 | 用途 |
|------|--------|--------|------|
| FriendIDsGetter | ws/handler | contact/dao | 获取用户好友 ID 列表 |
| FriendChecker | im/service | contact/dao | 检查是否为好友 |
| UserInfoGetter | im/service | auth/dao | 获取用户信息 |
| OnlineChecker | contact/service | ws/service | 检查在线状态 |
| OfflineMessagePusher | im/service | ws/handler | 离线消息推送 |
| GroupMemberChecker | im/service | group/dao | 检查群成员身份（Phase 2c） |
| GroupInfoGetter | im/service | group/dao | 获取群信息（Phase 2c） |
| GroupRoleChecker | im/service | group/dao | 检查用户群角色（Phase 2c） |

---

## 四、Wire 依赖注入规范

- 每个模块在 `provider.go` 中导出 `ProviderSet`（`wire.NewSet(...)`）
- 接口绑定统一在 `app/provider/wire.go` 中声明
- 修改 wire.go 后必须重新运行 `wire gen ./app/provider/` 生成 wire_gen.go
- Wire 有过手动 patch 历史（Phase 2b），修改后需检查 wire_gen.go 一致性

---

## 五、日志记录标准

所有 DAO 和 Service 的公开方法必须记录入口和出口日志：

```go
func (d *SomeDAO) SomeMethod(ctx context.Context, param int64) (result *Model, err error) {
    funcName := "dao.some_dao.SomeMethod"

    logs.LogFunctionEntry(ctx, funcName, map[string]interface{}{
        "param": param,
    })

    defer func() {
        logs.LogFunctionExit(ctx, funcName, result, err)
    }()

    // 业务逻辑
    return result, err
}
```

**funcName 命名规则：** `{层级}.{模块}_{文件}.{方法名}`
- DAO 层：`dao.group_dao.CreateGroup`
- Service 层：`service.group_service.CreateGroup`
- Controller 层：`controller.group_controller.CreateGroup`

---

## 六、Controller 错误处理标准

```go
func (ctrl *Controller) HandleAction(c *gin.Context) {
    funcName := "controller.module.HandleAction"

    // 参数绑定
    var req dto.SomeRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        utils.ResponseBadRequest(c, "参数校验失败: "+err.Error())
        return
    }

    // 调用 Service
    result, err := ctrl.service.DoAction(c.Request.Context(), &req)
    if err != nil {
        ctrl.handleError(c, funcName, err)
        return
    }

    utils.ResponseOK(c, "操作成功", result)
}

// handleError 必须覆盖所有已知业务错误
func (ctrl *Controller) handleError(c *gin.Context, funcName string, err error) {
    switch err {
    case service.ErrNotFound:
        utils.ResponseNotFound(c, err.Error())
    case service.ErrPermission:
        utils.ResponseForbidden(c, err.Error())
    // ... 覆盖所有已知错误
    default:
        logs.Error(c.Request.Context(), funcName, "操作失败", zap.Error(err))
        utils.ResponseError(c, "操作失败")
    }
}
```

**关键要求：**
- `handleError` 必须覆盖所有已知业务错误，不能用 `default` 笼统处理
- 错误 message 使用中文，面向用户
- 未知错误必须记录日志

---

## 七、批量查询优化

获取关联信息时，**必须使用批量查询 + Map 映射，严禁 N+1 查询**：

```go
// ✅ 正确：批量查询
userIDs := extractUserIDs(members)
users, _ := userDAO.GetByIDs(ctx, userIDs)
userMap := make(map[int64]*User)
for _, u := range users {
    userMap[u.ID] = u
}
for _, m := range members {
    m.Nickname = userMap[m.UserID].Nickname
}

// ❌ 错误：N+1 查询
for _, m := range members {
    user, _ := userDAO.GetByID(ctx, m.UserID)
    m.Nickname = user.Nickname
}
```

---

## 八、系统消息规范（Phase 2c+）

### 8.1 系统消息定义

系统消息是由服务端自动生成的提示类消息，写入 `im_messages` 表，type=10（`MessageTypeSystem`）。

### 8.2 内容格式

系统消息的 `content` 字段使用**纯文本格式**，不使用 JSON 结构。

### 8.3 前端渲染

系统消息在聊天页面中**居中显示，灰色小字体，无头像，无气泡**。

### 8.4 sender_id

系统消息的 `sender_id` 设为 0（表示系统），前端根据 sender_id=0 和 type=10 判断为系统消息。

---

## 九、WS 事件推送模式

### 9.1 S→C 推送（Service 层触发）

```go
pubsub.PublishToUser(ctx, targetUserID, &ws.Message{
    Event: "group.member.join",
    Data:  map[string]interface{}{...},
})

pubsub.PublishToUsers(ctx, memberIDs, &ws.Message{
    Event: "im.message.new",
    Data:  messageDTO,
})
```

### 9.2 C→S 事件处理（Hub 事件路由表注册）

```go
hub.RegisterEvent("im.message.send", handler.HandleSendMessage)
hub.RegisterEvent("im.message.read", handler.HandleReadMessage)
```

---

## 十、前端 Store 与 API 封装规范

### 10.1 API 封装标准

文件位置：`frontend/src/api/{module}.js`

```javascript
import request from '@/utils/request'

// 获取群详情
export function getGroupDetail(groupId) {
  return request.get(`/groups/${groupId}`)
}

// 创建群聊
export function createGroup(data) {
  return request.post('/groups', data)
}
```

**命名规则：**
- GET 请求：`get{Entity}` / `get{Entity}List`
- POST 请求：`create{Entity}` / `{action}{Entity}`
- PUT 请求：`update{Entity}` / `set{Entity}{Field}`
- DELETE 请求：`delete{Entity}` / `remove{Entity}`

### 10.2 Pinia Store 标准结构

文件位置：`frontend/src/store/{module}.js`

```javascript
import { defineStore } from 'pinia'

export const useGroupStore = defineStore('group', {
  state: () => ({
    conversations: [],      // 群会话列表
    currentGroup: null,      // 当前群详情
    messages: {},            // {conversationId: Message[]}
    members: {},             // {groupId: Member[]}
  }),

  getters: {
    unreadTotal: (state) => { ... },
  },

  actions: {
    // 初始化 WS 事件监听（在 App.vue onLaunch 中调用）
    initWsListeners() { ... },

    // 加载群会话列表
    async loadConversations() { ... },

    // 发送群消息
    async sendMessage(conversationId, content, atUserIds) { ... },
  }
})
```

**Store 设计原则：**
1. **单一职责**：每个模块一个 Store，不混合不同模块数据
2. **WS 监听统一初始化**：所有 Store 的 WS 事件监听在 `App.vue` 的 `_initGlobalWS` 中统一调用
3. **消息缓存**：按 conversationId 键值对缓存，避免重复请求
4. **乐观更新**：发送消息时先本地插入，再等待服务端 ack 确认
