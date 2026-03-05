# Phase 2c 实施计划：群聊与已读回执

> **状态：** ✅ 已完成（含代码审查修复 14 项 + 浏览器测试修复 21 项）
> **设计文档：** `docs/plans/2026-03-04-phase2c-design.md`
> **分支：** `feature/phase2c-group-read-receipt`
> **预计 Task 数：** 14 个（全部完成）+ Playwright 端到端测试
> **最后更新：** 2026-03-04

---

## Task 总览

| Task | 阶段 | 描述 | 依赖 | 状态 |
|------|------|------|------|------|
| Task 0 | 基础设施 | MinIO Docker + SDK + 通用上传 API | 无 | ✅ |
| Task 1 | 基础设施 | 数据库迁移 + Model + 常量定义 | 无 | ✅ |
| Task 2 | 群聊后端 | Group DAO 层 | Task 1 | ✅ |
| Task 3 | 群聊后端 | Group Service 业务逻辑 | Task 2 | ✅ |
| Task 4 | 群聊后端 | Group Controller + Router + Wire | Task 3 | ✅ |
| Task 5 | 群聊后端 | WS 群管理事件处理器 | Task 3 | ✅ |
| Task 6 | 群聊后端 | IM Service 扩展（群消息 + @提醒 + 管理员撤回） | Task 2 | ✅ |
| Task 7 | 已读回执 | 已读回执后端（ReadDAO + Service + API + WS 推送） | Task 1 | ✅ |
| Task 8 | 已读回执 | 前端已读回执 UI（单聊标记 + 群聊计数 + 详情页） | Task 7 | ✅ |
| Task 9 | 前端 | 群聊 Store + API 封装 + WS 事件监听 | Task 4, 5 | ✅ |
| Task 10 | 前端 | 群聊核心页面（Tab 改造 + 群对话页 + 创建群页） | Task 9 | ✅ |
| Task 11 | 前端 | 群聊管理页面（群设置 + 成员 + 邀请 + @选择器） | Task 10 | ✅ |
| Task 12 | 前端 | 群聊辅助功能（审批 + 搜索 + 免打扰 + 公告 UI） | Task 11 | ✅ |
| Task 13 | 管理端 | 管理端群聊管理 + 全量文档更新 + 代码审查 | Task 12 | ✅ |

---

## Task 0: MinIO Docker + SDK + 通用上传 API

**目标：** 引入 MinIO 文件存储服务，封装通用上传能力

### 交付物

1. **Docker Compose**
   - `deploy/docker/docker-compose.dev.yml` 添加 minio 服务
   - volumes 持久化 `minio_data`
   - 端口：9000（API）+ 9001（Console）

2. **Go 后端配置**
   - `config/config.yaml` 添加 minio 配置节
   - `config/config.go` 添加 MinioConfig 结构体
   - `pkg/storage/minio.go` — MinIO 客户端初始化（NewMinioClient）

3. **文件模块**
   - `app/file/service/file_service.go` — Upload(ctx, file) → URL
   - `app/file/controller/file_controller.go` — POST /api/v1/upload
   - `app/file/router.go` — 路由注册
   - `app/file/provider.go` — Wire ProviderSet

4. **集成**
   - `app/provider/wire.go` 添加 FileSet
   - `router/router.go` 注册 file 路由

### 验收标准

- `docker compose up -d minio` 正常启动
- `curl -F "file=@test.png" http://localhost:8085/api/v1/upload` 返回 MinIO URL
- MinIO Console (localhost:9001) 可看到上传的文件

---

## Task 1: 数据库迁移 + Model + 常量定义

**目标：** 建立 Phase 2c 所需的数据库表和 Go 模型

### 交付物

1. **新增表 SQL（init.sql 追加）**
   - `im_groups` 表（含所有注释和索引）
   - `im_group_join_requests` 表
   - `im_message_reads` 表

2. **ALTER 现有表**
   - `im_conversation_members` 新增 6 字段：role, nickname, is_muted, is_do_not_disturb, joined_at, at_me_count
   - `im_messages` 新增 1 字段：at_user_ids (BIGINT[])

3. **Go Model 文件**
   - `app/group/model/group.go` — Group 结构体
   - `app/group/model/join_request.go` — JoinRequest 结构体
   - `app/im/model/message_read.go` — MessageRead 结构体
   - `app/im/model/conversation_member.go` — 扩展 ConversationMember 结构体
   - `app/im/model/message.go` — 扩展 Message 结构体（at_user_ids）

4. **常量文件**
   - `app/constants/group.go` — 群角色、群状态、申请状态、系统消息类型

5. **DTO 文件**
   - `app/dto/group_dto.go` — 群聊相关 DTO
   - `app/dto/im_dto.go` — 扩展：已读回执 DTO + ConversationDTO 群聊字段

6. **GORM AutoMigrate**
   - `app/provider/wire_gen.go` 中确保新 Model 注册

### 验收标准

- 数据库迁移成功，`\d+ im_groups` 等表存在
- `go build` 编译通过
- 常量和 DTO 定义完整

---

## Task 2: Group DAO 层

**目标：** 实现群聊数据访问层

### 交付物

`app/group/dao/group_dao.go`

| 方法 | 功能 |
|------|------|
| CreateGroup | 创建 im_groups 记录 |
| GetByID | 根据 ID 查群信息 |
| GetByConversationID | 根据会话 ID 查群信息 |
| UpdateInfo | 更新群名/头像/公告/可搜索性/全体禁言 |
| UpdateOwner | 转让群主 |
| Dissolve | 解散群（status → 2） |
| GetMembers | 获取群成员列表（含用户信息 JOIN auth_users） |
| GetMemberRole | 查询用户在群内的角色 |
| IsMember | 检查用户是否为群成员 |
| AddMembers | 批量添加成员（im_conversation_members） |
| RemoveMember | 移除成员（DELETE im_conversation_members） |
| UpdateMemberRole | 更新成员角色 |
| UpdateMemberMute | 更新成员禁言状态 |
| UpdateMemberNickname | 更新群内昵称 |
| GetMemberCount | 获取群成员数 |
| SearchGroups | 搜索公开群（is_searchable=true，GIN 全文索引） |
| GetUserGroups | 获取用户加入的所有群 |
| CreateJoinRequest | 创建入群申请 |
| GetPendingJoinRequests | 获取待审批申请列表 |
| UpdateJoinRequest | 更新申请状态 |
| HasPendingRequest | 检查是否有待处理的申请 |

### 验收标准

- 所有方法有完整的日志记录（LogFunctionEntry/Exit）
- `go build` 编译通过

---

## Task 3: Group Service 业务逻辑

**目标：** 实现群聊核心业务逻辑

### 交付物

`app/group/service/group_service.go`

| 方法 | 功能 | 关键逻辑 |
|------|------|----------|
| CreateGroup | 创建群聊 | 创建 conversation(type=2) + groups + members，插入系统消息，推送通知 |
| GetGroupInfo | 获取群详情 | 包含成员数 |
| UpdateGroupInfo | 更新群信息 | 权限校验（群主/管理员），推送 group.info.update |
| UpdateNotice | 更新群公告 | 权限校验，插入系统消息，推送 group.notice.update |
| DissolveGroup | 解散群聊 | 仅群主，推送 group.dissolved |
| InviteMembers | 邀请入群 | 权限校验，检查上限，批量添加，系统消息，推送 group.member.join |
| KickMember | 踢人 | 层级权限（群主>管理员>成员），系统消息，推送 group.member.kicked |
| LeaveGroup | 退出群 | 群主不能退出（需先转让），系统消息，推送 group.member.leave |
| TransferOwner | 转让群主 | 仅群主，修改双方角色 |
| SetAdmin | 设置/取消管理员 | 仅群主 |
| MuteMember | 禁言/解除 | 群主/管理员（不能禁言同级或上级） |
| SetAllMuted | 全体禁言 | 群主/管理员 |
| UpdateNickname | 修改群昵称 | 群成员自行修改 |
| SearchGroups | 搜索公开群 | 分页 |
| ApplyJoin | 申请入群 | 创建申请，推送 group.join.request 给群主/管理员 |
| GetJoinRequests | 获取申请列表 | 权限校验 |
| ReviewJoinRequest | 审批申请 | 通过→添加成员+系统消息+推送，拒绝→更新状态 |

**接口依赖：**
- `UserInfoGetter` — 获取用户信息（批量）
- `PubSub` — 推送通知

### 验收标准

- 所有权限校验逻辑完整（群主 > 管理员 > 成员）
- 系统消息正确写入 im_messages (type=10)
- PubSub 推送事件正确

---

## Task 4: Group Controller + Router + Wire

**目标：** 暴露群聊 REST API 并集成到依赖注入

### 交付物

1. **Controller**
   - `app/group/controller/group_controller.go` — 16 个 REST API 处理函数
   - 统一使用 `utils.Response*` 系列响应
   - `handleError` 覆盖所有已知业务错误

2. **Router**
   - `app/group/router.go` — 路由注册
   - 路径前缀 `/api/v1/groups`
   - JWT 中间件保护

3. **Wire 集成**
   - `app/group/provider.go` — GroupSet
   - `app/provider/wire.go` — 添加 GroupSet + 新接口绑定
   - `app/provider/wire_gen.go` — 重新生成
   - `router/router.go` — 注册 group 路由

### 验收标准

- `go build` 编译通过
- 所有 16 个 API 端点可访问（需 JWT Token）
- API 响应格式统一

---

## Task 5: WS 群管理事件处理器

**目标：** 实现群管理相关的 WebSocket 事件推送

### 交付物

`app/group/handler/group_handler.go`

| WS 事件 | 触发时机 | 推送目标 |
|---------|----------|----------|
| group.member.join | 成员加入 | 群所有成员 |
| group.member.leave | 成员退出 | 群所有成员 |
| group.member.kicked | 成员被踢 | 群所有成员 + 被踢者 |
| group.info.update | 群信息变更 | 群所有成员 |
| group.notice.update | 群公告变更 | 群所有成员 |
| group.dissolved | 群解散 | 群所有成员 |
| group.mute.update | 禁言变更 | 群所有成员 |
| group.join.request | 新入群申请 | 群主 + 管理员 |
| group.join.approved | 申请通过 | 申请人 |

**注册方式：** 通过 Hub.RegisterEvent 注册到事件路由表（如有 C→S 事件），S→C 推送通过 PubSub.PublishToUser/PublishToUsers。

### 验收标准

- 群管理操作后相关成员能收到实时通知
- 系统消息在群聊中正确显示

---

## Task 6: IM Service 扩展（群消息 + @提醒 + 管理员撤回）

**目标：** 扩展现有 IMService 以支持群聊消息场景

### 交付物

1. **IMService 方法扩展**

| 方法 | 变更内容 |
|------|----------|
| SendMessage | 增加群聊分支：成员校验、禁言检查、批量未读递增、免打扰跳过 Redis |
| RecallMessage | 增加群聊分支：管理员可撤回他人消息无时限，撤回展示区分操作者 |
| GetConversationList | 增加群聊会话：填充群名/群头像/成员数/免打扰/被@计数 |
| GetHistoryMessages | 群聊适配：返回发送者群昵称（优先于全局昵称） |

2. **新增接口定义**（`app/im/service/im_service.go`）

```go
type GroupMemberChecker interface {
    IsMember(ctx context.Context, conversationID, userID int64) (bool, error)
    GetMemberRole(ctx context.Context, conversationID, userID int64) (int, error)
    IsMuted(ctx context.Context, conversationID, userID int64) (bool, error)
    IsAllMuted(ctx context.Context, conversationID int64) (bool, error)
}

type GroupInfoGetter interface {
    GetByConversationID(ctx context.Context, conversationID int64) (*GroupBasicInfo, error)
    GetMemberCount(ctx context.Context, groupID int64) (int, error)
}
```

3. **SendMessage 扩展**
   - `app/dto/im_dto.go` — SendMessageRequest 新增 at_user_ids 字段
   - `app/im/handler/event_handler.go` — 适配群消息推送（推给所有群成员）
   - im.message.new 推送增加 conv_type、at_user_ids 字段

4. **免打扰逻辑**
   - 免打扰成员：递增会话 unread_count，不递增 Redis 全局未读
   - 会话列表中免打扰会话显示灰色未读数

### 验收标准

- 群消息发送/撤回功能正常
- 禁言用户发消息被拒
- @提醒字段正确存储和推送
- 免打扰用户不增加全局未读数

---

## Task 7: 已读回执后端

**目标：** 实现已读回执完整后端逻辑

### 交付物

1. **ReadDAO**
   - `app/im/dao/read_dao.go`
   - BatchCreate(ctx, messageIDs[], userID) — 批量写入已读记录
   - GetMessageReadUsers(ctx, messageID, page, limit) — 查询已读用户列表
   - GetMessageReadCount(ctx, messageID) — 查询已读计数
   - GetBatchReadCounts(ctx, messageIDs[]) — 批量查询已读计数

2. **IMService 扩展**
   - MarkRead — 重构：单聊走 last_read_msg_id，群聊写 im_message_reads
   - GetMessageReadDetail — 已读详情列表（含用户信息）
   - GetMessageReadCount — 已读/未读计数

3. **REST API**
   - GET /api/v1/im/messages/:id/reads — 已读详情
   - GET /api/v1/im/messages/:id/read-count — 已读计数
   - Controller + Router 更新

4. **WS 事件**
   - im.message.read — 重构：单聊推 read.ack，群聊推 read.count
   - im.message.read.ack — 单聊实时推送
   - im.message.read.count — 群聊计数推送

### 验收标准

- 单聊：打开会话后对方看到 "已读"
- 群聊：打开会话后发送者看到 "X人已读"
- 点击 "X人已读" 可查看已读/未读人员列表

---

## Task 8: 前端已读回执 UI

**目标：** 前端实现已读回执展示（使用 ui-ux-pro-max 设计）

### 交付物

1. **chat Store 扩展**（`store/chat.js`）
   - 新增 readStatus 状态管理
   - WS 监听 im.message.read.ack / im.message.read.count
   - 打开会话时发送 im.message.read

2. **单聊已读标记**（修改 `pages/chat/conversation.vue`）
   - 自己发的消息下方显示 "已读" / "未读"
   - 基于 last_read_msg_id 判断

3. **群聊已读计数**（后续 Task 10 的 group/conversation.vue 中实现基础展示）
   - 消息下方显示 "X人已读"
   - 点击跳转已读详情页

4. **已读详情页**（新增 `pages/chat/read-detail.vue`）
   - Tab 切换：已读 / 未读
   - 用户列表（头像 + 昵称 + 已读时间）
   - 调用 GET /api/v1/im/messages/:id/reads

### 验收标准

- 单聊对话页正确显示 "已读"/"未读"
- 已读详情页正确展示已读/未读人员
- 使用 ui-ux-pro-max 设计规范

---

## Task 9: 前端群聊 Store + API 封装

**目标：** 建立前端群聊数据管理层

### 交付物

1. **API 封装**
   - `api/group.js` — 16 个群聊 REST API 封装
   - `api/file.js` — 文件上传 API 封装

2. **群聊 Store**（`store/group.js`）
   - state: groupConversations, currentGroup, groupMessages, groupMembers
   - actions: loadGroupConversations, sendGroupMessage, loadGroupHistory, ...
   - WS 监听: im.message.new(conv_type=2), group.* 系列事件
   - 群消息缓存策略（同 chat.js 模式）

3. **WS 事件注册**
   - App.vue `_initGlobalWS` 中初始化 groupStore.initWsListeners()
   - 处理 group.member.join/leave/kicked/info.update/notice.update/dissolved 等

### 验收标准

- API 封装完整，方法命名清晰
- Store 能正确管理群聊状态
- WS 群管理事件正确触发 Store 更新

---

## Task 10: 群聊核心页面（ui-ux-pro-max）

**目标：** 实现群聊核心交互页面

### 交付物

1. **会话列表 Tab 改造**（修改 `pages/chat/index.vue`）
   - 顶部增加 Tab 切换：单聊 / 群聊
   - 群聊 Tab 展示群会话列表（群名、群头像、最后消息、未读数）
   - 免打扰群未读数显示为灰色
   - 被@标记："[N条] @了我"

2. **群聊对话页**（新增 `pages/group/conversation.vue`）
   - 消息列表（显示发送者昵称 + 头像）
   - @成员选择器（输入 @ 弹出成员列表）
   - 消息发送（含 at_user_ids）
   - 消息撤回（管理员额外权限）
   - 系统消息特殊展示（居中灰色文字）
   - "X人已读" 显示 + 点击跳转详情

3. **创建群聊页**（新增 `pages/group/create.vue`）
   - 好友列表多选
   - 搜索用户 ID 添加
   - 填写群名称
   - 确认创建

4. **CustomTabBar 适配**
   - TabBar 新增或适配群聊入口（如果需要）

### 验收标准

- Tab 切换流畅，单聊/群聊列表独立
- 群聊对话页消息展示正确
- @选择器交互流畅（输入 @ 弹出选择列表）
- 创建群聊后自动跳转到群聊对话页

---

## Task 11: 群聊管理页面（ui-ux-pro-max）

**目标：** 实现群聊管理相关页面

### 交付物

1. **群设置页**（新增 `pages/group/settings.vue`）
   - 群信息展示（名称/头像/公告/群 ID）
   - 群信息编辑（群主/管理员可修改名称/头像/公告）
   - 成员概览（前 N 个头像 + "查看全部"）
   - 免打扰开关
   - 群昵称设置
   - 退出群聊 / 解散群聊

2. **群成员列表页**（新增 `pages/group/members.vue`）
   - 完整成员列表
   - 角色标识（群主皇冠/管理员盾牌/成员无标识）
   - 管理操作入口（踢人/禁言/设管理员）— 根据当前用户角色显示
   - 搜索成员

3. **邀请入群页**（新增 `pages/group/invite.vue`）
   - 好友列表选择（排除已在群内的）
   - 搜索用户 ID 添加

### 验收标准

- 群设置页信息展示完整
- 角色权限控制正确（群主能看到所有管理入口，普通成员看不到）
- 邀请页排除已有成员

---

## Task 12: 群聊辅助功能（ui-ux-pro-max）

**目标：** 实现群聊辅助功能页面

### 交付物

1. **入群申请审批页**（新增 `pages/group/join-requests.vue`）
   - 待审批申请列表（申请人信息 + 附言 + 时间）
   - 通过/拒绝操作
   - 已处理记录

2. **搜索群聊页**（新增 `pages/group/search.vue`）
   - 关键词搜索公开群
   - 搜索结果：群名、头像、成员数、简介
   - 申请加入（填写申请附言）
   - 已加入的群直接进入

3. **免打扰 UI**
   - 群设置页免打扰开关
   - 会话列表免打扰标识（灰色未读数）

4. **群公告 UI**
   - 群设置页公告展示 + 编辑
   - 群公告系统消息展示

5. **全局消息搜索扩展**（修改 `pages/chat/search.vue`）
   - 搜索结果包含群聊消息
   - 结果标识会话类型（单聊/群聊图标）

### 验收标准

- 入群审批流程完整（申请→通知→审批→入群）
- 群搜索结果准确
- 免打扰功能正常
- 全局搜索包含群聊消息

---

## Task 13: 管理端 + 文档更新 + 代码审查

**目标：** 管理端群聊管理 + 全量文档同步 + 代码审查

### 交付物

1. **管理端后端**
   - `app/admin/service/group_manage_service.go` — 群列表/详情/解散/移除
   - `app/admin/controller/group_manage_controller.go` — 4 个 REST API
   - `app/admin/router.go` — 路由扩展

2. **管理端前端**
   - `admin/src/views/group/list.vue` — 群列表页（搜索/分页/状态筛选）
   - `admin/src/views/group/detail.vue` — 群详情页（成员管理/解散）
   - `admin/src/api/group.js` — API 封装
   - 侧边栏菜单添加群聊管理入口

3. **文档更新**
   - `docs/progress/CURRENT_STATUS.md` — 进度更新
   - `.cursor/rules/project-context.mdc` — 记忆更新
   - `docs/architecture/system-architecture.md` — 架构更新
   - `docs/plans/2026-02-27-echochat-system-design.md` — 总体设计更新
   - `docs/api/frontend/im.md` — IM API 文档更新
   - `docs/api/frontend/group.md` — 新增群聊 API 文档
   - `docs/api/websocket.md` — WS 事件文档更新
   - `docs/api/admin/group.md` — 新增管理端群聊 API 文档
   - `docs/api/README.md` — 导航更新

4. **代码审查**
   - 使用 code-reviewer 子代理进行结构化审查
   - 修复审查发现的问题

### 验收标准

- 管理端群列表/详情页功能正常
- 管理员可解散群/移除成员
- 所有文档与代码保持一致
- 代码审查通过

---

## 实施依赖关系图

```
Task 0 (MinIO)  ──────────────────────────────────────────────┐
Task 1 (DB迁移) ──┬── Task 2 (Group DAO) ──┬── Task 3 (Group Service) ──┬── Task 4 (Controller+Wire)
                  │                        │                           └── Task 5 (WS Handler)
                  │                        └── Task 6 (IM扩展)
                  └── Task 7 (已读回执后端) ── Task 8 (已读回执前端)
                                                                            │
Task 4 + Task 5 ── Task 9 (Store+API) ── Task 10 (核心页面) ── Task 11 (管理页面) ── Task 12 (辅助功能)
                                                                                                    │
Task 0 + Task 12 ── Task 13 (管理端+文档+审查)
```

---

## 开发注意事项

### 代码风格全局一致（最高优先级）

**编写任何新代码前，必须先阅读同层级现有模块的实际代码，严格复制其风格。** 详细规范见 `docs/conventions/backend-module-architecture.md`。

**强制执行流程：**
1. 写代码前 → 用 Read/Grep 读取同类型现有文件
2. 写代码时 → 逐行对照导入、结构体、方法签名、日志调用、错误处理
3. 写代码后 → 与参照文件做差异比对，确认风格完全一致

**各层参照文件（Phase 2c 新模块必读）：**

| 新模块文件 | 参照现有文件 |
|-----------|------------|
| group/controller | `app/contact/controller/contact_controller.go`、`app/im/controller/im_controller.go` |
| group/service | `app/im/service/im_service.go`、`app/contact/service/contact_service.go` |
| group/dao | `app/im/dao/conversation_dao.go`、`app/contact/dao/friendship_dao.go` |
| group/model | `app/im/model/conversation.go`、`app/im/model/message.go` |
| group/router.go | `app/contact/router.go`、`app/im/router.go` |
| group/provider.go | `app/contact/provider.go` |
| constants/group.go | `app/constants/im.go`、`app/constants/contact.go` |
| dto/group_dto.go | `app/dto/im_dto.go`、`app/dto/contact_dto.go` |

**严格禁止事项：**
- 禁止调用不存在的 API（如 `logs.LogFunctionEntry`）
- 禁止引入与当前 Go 版本不兼容的依赖
- 禁止自创新封装模式（如自定义错误处理框架）
- 禁止在 Controller 层记日志（auth/admin 模块除外）
- 禁止使用未经项目验证的第三方库

### Code Review 遗留问题清单（Important/Minor）

以下问题在首轮 Code Review 中被标记为 Important 或 Minor，已在后续 Task 中逐步处理。

#### Important 级别

| # | 模块 | 问题 | 修复优先级 |
|---|------|------|-----------|
| I-1 | group/service | `UpdateGroup` 中 `_ = member` 无效赋值，应改为 `_, _, err :=` | 低（代码清洁） |
| I-2 | group/service | `MuteMember` 使用 `ErrCannotKickHigherRole` 语义不精确，应新增通用错误 | 中 |
| I-3 | group/dao | `SearchGroups` 的 `to_tsquery` 对特殊字符敏感，应改用 `plainto_tsquery` | 高（用户输入安全） |
| I-4 | group/service | `InviteMembers` 循环逐条插入缺少事务包裹和批量优化 | 中（性能） |
| I-5 | group/controller | `handleError` 缺少 `ErrAlreadyMuted/ErrUserMuted/ErrGroupAllMuted` 映射 | 高 |
| I-6 | group/model | `MessageRead` 放在 `group/model` 而非 `im/model`，领域归属可议 | 低（架构决策） |
| I-7 | im/service | `sendGroupMessage` 推送未检查成员免打扰设置 | 中（业务逻辑） |
| I-8 | im/service | 群消息推送逐用户查询 N+1 问题 | 中（性能） |
| I-9 | im/service | WS 群已读事件 `im.message.read` 处理器未注册 | 高（Task 5 实现） |
| I-10 | im/service | 消息推送数据缺少 `conv_type` 字段 | 高（前端适配） |
| I-11 | im/service | `GetMessageReadDetail` 无分页参数 | 中 |
| I-12 | file/service | 文件上传缺少 MIME 类型校验 | 中（安全性） |
| I-13 | file/service | 上传返回 URL 使用内部地址，应返回可配置前缀 | 中 |
| I-14 | pkg/storage | MinIO 初始化无超时控制 | 低 |

#### Minor 级别

| # | 模块 | 问题 |
|---|------|------|
| M-1 | group/dao | `GetMemberCount` / `GetMemberIDs` 缺少 funcName 和日志 |
| M-2 | group/dao | `HasRead` 缺少 funcName 和日志 |
| M-3 | group/provider | 与 IM 模块 Provider 风格略有差异（无 ProvideXxx 包装） |
| M-4 | group/service | `SearchGroups` 逐群查 `GetMemberCount`，建议批量 COUNT |
| M-5 | group/service | `imMember` 内部类型可直接使用 `imModel.ConversationMember` |
| M-6 | group/controller | `SetAllMuted` 使用匿名结构体，建议改用 DTO |

### 其他注意事项

1. **接口注入模式**：新模块间通信必须走 interface injection，禁止直接 import
2. **批量查询优化**：群成员信息获取使用批量查询 + Map 映射，避免 N+1
3. **前端设计规范**：所有前端页面使用 ui-ux-pro-max 技能包设计
4. **系统消息**：群管理操作产生的系统消息统一使用 type=10（MessageTypeSystem），内容格式化
5. **权限层级**：群主(2) > 管理员(1) > 成员(0)，操作时必须校验层级
6. **Wire 依赖**：Phase 2b 中 Wire 有过手动 patch 历史，注意检查 wire_gen.go 一致性
7. **依赖管理**：当前 Go 版本 1.23.12，添加新依赖必须选择兼容版本

---

## Playwright 浏览器测试修复记录

> 在代码审查修复完成后，使用 Playwright MCP 进行端到端浏览器测试，共发现并修复 21 项问题。

| # | 类型 | 修复内容 | 涉及文件 |
|---|------|----------|----------|
| T1 | Bug | create.vue 创建群后跳转使用 `result.id`（原 `result.group_id` 不存在） | `frontend/src/pages/group/create.vue` |
| T2 | Bug | 群聊页面成员显示使用 `user_nickname` 替代 `username`（对齐 GroupMemberDTO） | `conversation/settings/members/join-requests.vue` |
| T3 | UI | 管理端群详情弹窗重新设计（卡片头像+分区布局+动态配色） | `admin/src/views/group/list.vue` |
| T4 | UX | 群搜索结果已加入的群显示「已加入」标签，不显示「申请加入」 | `frontend/src/pages/group/search.vue` |
| T5 | Bug | `SearchGroups` 从 `to_tsvector` 改为 `ILIKE`（修复混合中英文搜索） | `backend/go-service/app/group/dao/group_dao.go` |
| T6 | Bug | conversation.vue 增加 groupId=0 时从 chatStore 回退查找 | `frontend/src/pages/group/conversation.vue` |
| T7 | UX | 创建群最低人数从 2 改为 1（支持 2 人群聊） | `frontend/src/pages/group/create.vue` |
| T8 | 功能 | 创建群聊支持搜索非好友用户（全站搜索 + 非好友标签 + selectedMap） | `frontend/src/pages/group/create.vue` |
| T9 | 功能 | 邀请入群支持搜索非好友用户（同 create.vue 改造模式） | `frontend/src/pages/group/invite.vue` |
| T10 | Bug | 管理端成员表用 `username` 替代 `user_nickname`（对齐 admin DTO） | `admin/src/views/group/list.vue` |
| T11 | UX | 群聊已读回执：无人已读时显示「0人已读」而非隐藏标签 | `frontend/src/pages/group/conversation.vue` |
| T12 | Bug | 新好友聊天页 conversationId=0 时「加载更多」点击 400 报错 | `frontend/src/pages/chat/conversation.vue` + `frontend/src/store/chat.js` |
| T13 | UX | 联系人 TabBar 添加好友申请未读数 badge（与消息 Tab 一致） | `frontend/src/components/CustomTabBar.vue` |
| T14 | 功能 | App.vue 启动时预加载好友申请数，确保 badge 立即可见 | `frontend/src/App.vue` |
| T15 | Bug | 单聊已读回执：后端 MarkRead 缺失 `im.message.read.ack` 推送 | `backend/go-service/app/im/service/im_service.go` |
| T16 | UX | 群成员列表为所有角色添加身份标识（🔑群主/🛡管理员/👤成员） | `frontend/src/pages/group/members.vue` |
| T17 | Bug | 全局修复 `e?.data?.message` → `e?.message`（8 文件 18 处） | `create/search/read-detail/members/join-requests/settings/request.vue` |
| T18 | UI/UX | 群成员管理操作弹窗改为自定义组件 + 三个点按钮从 @longpress 改为 @tap | `frontend/src/pages/group/members.vue` |
| T19 | 功能 | 已读详情页展示群昵称：DTO 新增 group_nickname + DAO 批量查询 + 前端主显群昵称/副显真实昵称 | `group_dto.go` + `conversation_dao.go` + `im_service.go` + `read-detail.vue` |
| T20 | Bug | 消息免打扰后端 API 补全：DAO UpdateMemberDND + Service SetDoNotDisturb + Controller + Router 注册 | `conversation_dao.go` + `im_service.go` + `im_controller.go` + `router.go` |
| T21 | Bug | 联系人 Tab 页切换回来数据不刷新：新增 onShow 生命周期钩子自动重新获取好友列表和待处理申请数 | `pages/contact/index.vue` |
