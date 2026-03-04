# EchoChat 项目开发进度

> **最后更新**：2026-03-04（Phase 2c 设计完成，待实施）
> **当前阶段**：Phase 2c 设计完成，准备开始实施
> **当前分支**：`feature/phase2c-group-read-receipt`（已从 phase2b 拉出）
> **实施计划**：`docs/plans/2026-03-04-phase2c-implementation.plan.md`
> **设计文档**：`docs/plans/2026-03-04-phase2c-design.md`

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
│       ├── api/{auth,contact,user,im}.js
│       ├── services/websocket.js
│       ├── store/{user,websocket,contact,chat}.js
│       ├── pages/chat/          # [Phase 2b] 4 个页面
│       │   ├── index.vue        # 会话列表
│       │   ├── conversation.vue # 聊天对话
│       │   ├── settings.vue     # 聊天设置
│       │   └── search.vue       # 消息搜索
│       ├── pages/contact/       # [Phase 2a] 6 个页面
│       └── components/CustomTabBar.vue（含 badge）
├── admin/                       # 管理端（Vue 3 + Element Plus）
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

### 启动命令

```bash
# 1. 启动 PostgreSQL + Redis
cd deploy && docker compose -f docker-compose.dev.yml up -d postgres redis

# 2. 启动 Go 后端（http://localhost:8085）
cd backend/go-service && go run cmd/server/main.go

# 3. 启动管理端（http://localhost:3100）
cd admin && npm run dev

# 4. 启动前台 H5（http://localhost:5173+）
cd frontend && npm run dev:h5
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

## 八、下一阶段：Phase 2c — 群聊与已读回执

> **状态：** 设计完成，待实施
> **设计文档：** `docs/plans/2026-03-04-phase2c-design.md`
> **实施计划：** `docs/plans/2026-03-04-phase2c-implementation.plan.md`
> **分支：** `feature/phase2c-group-read-receipt`（已创建，从 phase2b 拉出）

### 功能范围

| 模块 | 内容 |
|------|------|
| 群聊管理 | 建群/加入/退出/解散/搜索/三级角色/禁言/全体禁言/群公告/群昵称/免打扰 |
| 群消息 | 复用 im.message.* 事件 + @某人/@所有人 + 管理员撤回（无时限）+ 系统消息 |
| 已读回执 | 单聊会话级（last_read_msg_id）+ 群聊消息级（im_message_reads 表）+ 实时推送 |
| MinIO | Docker 容器 + Go SDK + 通用上传 API（群头像） |
| 管理端 | 群列表/群详情/解散群/移除成员 |
| 前端 | 9 个新页面 + 群聊 Store + 会话列表 Tab 改造 |

### Task 概览（14 个）

| Task | 描述 | 状态 |
|------|------|------|
| Task 0 | MinIO Docker + SDK + 通用上传 API | 📋 |
| Task 1 | 数据库迁移 + Model + 常量 | 📋 |
| Task 2 | Group DAO 层 | 📋 |
| Task 3 | Group Service 业务逻辑 | 📋 |
| Task 4 | Group Controller + Router + Wire | 📋 |
| Task 5 | WS 群管理事件处理器 | 📋 |
| Task 6 | IM Service 扩展（群消息/@提醒/管理员撤回） | 📋 |
| Task 7 | 已读回执后端 | 📋 |
| Task 8 | 前端已读回执 UI | 📋 |
| Task 9 | 前端群聊 Store + API + WS 监听 | 📋 |
| Task 10 | 群聊核心页面（Tab + 对话 + 创建） | 📋 |
| Task 11 | 群聊管理页面（设置 + 成员 + 邀请 + @选择器） | 📋 |
| Task 12 | 群聊辅助功能（审批 + 搜索 + 免打扰 + 公告） | 📋 |
| Task 13 | 管理端 + 文档更新 + 代码审查 | 📋 |

### 留待后续阶段

- 消息类型扩展（图片/语音/文件）
- 管理端消息管理功能
