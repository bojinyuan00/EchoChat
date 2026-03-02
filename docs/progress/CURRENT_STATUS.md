# EchoChat 项目开发进度

> **最后更新**：2026-03-02（Phase 2a 代码审查全面修复 + Phase 2b 架构建议）
> **当前阶段**：Phase 2a 已完成，Phase 2b 待设计
> **当前分支**：`feature/phase2a-websocket-contacts`
> **实施计划**：`phase_2a_实施计划_221003ce.plan.md`
> **设计文档**：`docs/plans/2026-03-02-phase2a-design.md`
> **Phase 2b 架构备忘**：`docs/plans/2026-03-02-phase2b-architecture-notes.md`

---

## 一、Phase 2a Task 完成状态

| Task | 描述 | 状态 | 备注 |
|------|------|------|------|
| Task 0 | 设计文档 + 新分支 | ✅ 完成 | 架构设计、Redis Pub/Sub、文档策略 |
| Task 1 | 数据库表结构 | ✅ 完成 | contact_friendships + contact_groups |
| Task 2 | WebSocket 核心模块 | ✅ 完成 | Hub + Client + PubSub + Handler |
| Task 3 | Contact 模型与 DAO | ✅ 完成 | friendship + friend_group DAO |
| Task 4 | Contact Service | ✅ 完成 | 好友申请/分组/黑名单/搜索/推荐 |
| Task 5 | Contact Controller & Router | ✅ 完成 | 17 个 REST API + Wire 集成 |
| Task 6 | 在线状态管理 | ✅ 完成 | Redis SET + TTL 心跳续期 |
| Task 7 | 管理端后端 | ✅ 完成 | 在线监控 + 好友关系管理 API |
| Task 8 | 前台 WS 客户端 + Store + API | ✅ 完成 | websocket.js + contact.js Store/API |
| Task 9 | 前台联系人页面 | ✅ 完成 | 6 个页面（ui-ux-pro-max 规范） |
| Task 10 | 管理端前端 | ✅ 完成 | 在线监控 + 好友管理页面 |
| Task 11 | API 文档编写 | ✅ 完成 | 4 份独立文档 |
| Task 12 | 集成测试 + 文档更新 + 代码审查 | ✅ 完成 | 三端编译通过 |

---

## 二、Phase 2a 新增功能

### WebSocket 实时通讯
- **连接管理**：`gorilla/websocket` + JWT 认证 + 心跳（30s）
- **消息架构**：Redis Pub/Sub 跨实例消息路由
- **Hub**：本地连接管理（注册/注销/按用户发送）
- **Client**：读写泵 + 断线回调 + 缓冲通道

### 联系人管理（17 个 API）
- 好友申请（发送/接受/拒绝）
- 好友列表（按分组筛选 + 在线状态）
- 好友详情（备注/分组移动）
- 好友删除 + 拉黑/取消拉黑
- 好友分组（CRUD + 排序）
- 用户搜索 + 好友推荐（共同好友算法）

### 在线状态管理
- Redis SET `echo:user:online` 存储在线用户集合
- Redis STRING `echo:user:status:{user_id}` + TTL 心跳续期
- Pub/Sub 推送好友上下线通知

### 管理端扩展
- 在线监控页面（自动 30s 刷新 + 统计卡片）
- 好友关系管理（分页列表 + 强制删除）

---

## 三、Phase 1 完成总结

| Task | 描述 | 状态 |
|------|------|------|
| Task 1-11 | 基础设施 + 认证 + 用户管理 | ✅ 全部完成 |

- Go 后端 15+ API、JWT 有状态认证、RBAC 角色权限（level 等级体系）
- 前台 uni-app 登录/注册/TabBar/个人中心
- 管理端 Vue 3 登录/仪表盘/用户列表/详情
- Docker Compose 一键启动

---

## 四、关键技术决策记录

### 后端（Go）
1. **框架组合**：Gin + GORM + Wire + Zap + Viper
2. **JWT 策略**：有状态 JWT，Token 按 clientType 隔离存储在 Redis
3. **WebSocket**：`gorilla/websocket` + Redis Pub/Sub 跨实例路由
4. **在线状态**：混合方案（Redis SET + STRING TTL + Pub/Sub 推送）
5. **角色等级**：`auth_roles.level`（1=超管, 10=管理员, 100=普通用户）

### 前台用户端（frontend/）
1. **框架**：uni-app 3.0（Vue 3.4.21）
2. **状态管理**：Pinia 2.1.7 + pinia-plugin-persistedstate@3
3. **WebSocket**：`uni.connectSocket`（小程序）/ `WebSocket`（H5）
4. **设计系统**：ui-ux-pro-max 规范

### 后台管理端（admin/）
1. **框架**：Vue 3.5+ + Vite 7.x + Element Plus
2. **HTTP 客户端**：Axios
3. **存储隔离**：localStorage key 前缀 `admin_`

---

## 五、目录结构概览

```
EchoChat/
├── backend/go-service/
│   ├── app/
│   │   ├── admin/               # 管理端（controller/service/provider）
│   │   ├── auth/                # 认证模块
│   │   ├── contact/             # [Phase 2a] 联系人模块
│   │   │   ├── controller/
│   │   │   ├── dao/
│   │   │   ├── model/
│   │   │   ├── service/
│   │   │   ├── router.go
│   │   │   └── provider.go
│   │   ├── ws/                  # [Phase 2a] WebSocket 模块
│   │   │   ├── handler.go
│   │   │   ├── online_service.go
│   │   │   ├── provider.go
│   │   │   └── router.go
│   │   ├── constants/
│   │   ├── dto/
│   │   └── provider/
│   ├── pkg/
│   │   ├── ws/                  # [Phase 2a] WebSocket 核心
│   │   │   ├── hub.go
│   │   │   ├── client.go
│   │   │   ├── pubsub.go
│   │   │   └── message.go
│   │   ├── db/ logs/ middleware/ utils/
│   └── router/router.go
├── frontend/                    # 前台（uni-app）
│   └── src/
│       ├── api/{auth,contact,user}.js
│       ├── services/websocket.js     # [Phase 2a]
│       ├── store/{user,websocket,contact}.js
│       ├── pages/contact/            # [Phase 2a] 6 个页面
│       │   ├── index.vue
│       │   ├── request.vue
│       │   ├── detail.vue
│       │   ├── search.vue
│       │   ├── groups.vue
│       │   └── blacklist.vue
│       └── components/CustomTabBar.vue
├── admin/                       # 管理端（Vue 3 + Element Plus）
│   └── src/
│       ├── api/{auth,user,monitor,contact}.js
│       ├── views/
│       │   ├── monitor/online.vue    # [Phase 2a]
│       │   ├── contact/list.vue      # [Phase 2a]
│       │   ├── layout/ login/ dashboard/ user/
│       └── router/index.js
├── deploy/
├── design-system/
└── docs/
    ├── api/
    │   ├── frontend/{auth,contact,websocket}.md
    │   ├── admin/{auth,user,online,contact}.md
    │   └── websocket.md
    ├── plans/
    ├── progress/CURRENT_STATUS.md
    └── conventions/
```

---

## 六、开发测试指南

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

### Phase 2a 可测试功能

- **前台联系人**：好友列表 → 搜索添加 → 好友申请 → 好友详情 → 备注/分组 → 拉黑/删除
- **前台 WebSocket**：自动连接 → 心跳 → 在线状态实时更新 → 好友申请推送
- **管理端在线监控**：在线用户数 → 在线用户列表 → 自动刷新
- **管理端好友管理**：好友关系列表 → 强制删除关系

---

## 七、已知问题

1. uni-app 的 `tabBar.custom: true` 配合自定义 TabBar 组件使用
2. Go 依赖版本需匹配 Go 1.23.12
3. 管理端 Element Plus 全量导入导致打包体积较大（后续可改为按需导入）

---

## 八、Phase 2a 后期 Bug 修复记录（2026-03-02）

### 修复 1: GetRecommendFriends 返回空数据
- **问题**：对每个候选人调用 `SearchUsers` 并忽略结果，返回的 DTO 只有 ID 字段
- **修复**：FriendshipDAO 新增 `GetUsersByIDs` 批量查询方法，Service 层收集所有候选人 ID 后一次查询，正确填充 Username/Nickname/Avatar
- **改进**：排序算法从手动冒泡改为 `sort.Slice`

### 修复 2: 上下线好友通知未实现
- **问题**：`UserOnline`/`UserOffline` 没有调用已有的 `NotifyFriendsStatusChange` 方法
- **修复**：OnlineService 新增 `FriendIDsGetter` 接口依赖（由 FriendshipDAO 隐式实现），在上线/下线时获取好友列表并推送状态变更通知
- **架构**：使用接口注入避免 ws 包直接依赖 contact 包

### 修复 3: 管理端在线用户 API 缺少用户名
- **问题**：`GetOnlineUsers` 只返回 `[]int64`，管理端无法展示用户名
- **修复**：OnlineManageService 注入 `*gorm.DB`，返回 `[]OnlineUserInfo`（含 user_id + username），查询 auth_users 表补充信息

### 修复 4: WebSocket Token 未校验 Redis 状态（I2）
- **问题**：`ws.handler.Upgrade` 只校验 JWT 签名和过期时间，未检查 Token 是否仍在 Redis 中有效（已登出用户仍可建立 WS 连接）
- **修复**：新增 `TokenValidator` 接口，由 `AuthService` 实现；`Upgrade` 流程增加 Redis Token 校验步骤
- **架构**：使用接口注入避免 ws 包直接依赖 auth 包

### 修复 5: HeartbeatRenew Redis 错误未检查（M6）
- **问题**：`HeartbeatRenew` 中 `rdb.Expire` 调用结果被忽略
- **修复**：增加错误检查与日志记录

### 修复 6: json.Marshal 错误未处理（M2）
- **问题**：`handler.go` 心跳/默认响应的 `MarshalResponse` 和 `online_service.go` 中 `json.Marshal` 错误被忽略
- **修复**：所有 Marshal 调用增加错误检查，失败时记录日志并提前返回

### 修复 7: Controller 统一错误处理（I8）
- **问题**：`contact_controller.go` 中多数 endpoint 使用硬编码的 `utils.ResponseError`，未经过 `handleError` 业务错误映射
- **修复**：所有 13 个 endpoint 统一走 `handleError`，确保已知业务错误（404/400/403）被正确返回

### 修复 8: 管理端 Controller 注释与日志规范化（M1）
- **问题**：`online_controller.go` 和 `contact_manage_controller.go` 缺少包注释、函数注释和 `funcName` 日志模式
- **修复**：补全所有注释，增加 `funcName` + `logs.Error/Info` 结构化日志

---

## 九、下一阶段规划

### Phase 2b - 即时通讯消息系统
- 会话管理（单聊/群聊）
- 消息收发 + 离线消息
- 消息通知
- 已读回执
