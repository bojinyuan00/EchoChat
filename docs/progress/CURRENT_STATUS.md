# EchoChat 项目开发进度

> **最后更新**：2026-03-02（前后端错误处理规范统一 + 安全加固）
> **当前阶段**：Phase 1 - 基础设施与用户认证
> **当前分支**：`feature/phase1-foundation-and-auth`
> **实施计划**：`docs/plans/2026-02-27-phase1-foundation-and-auth.md`

---

## 一、Task 完成状态

| Task | 描述 | 状态 | 备注 |
|------|------|------|------|
| Task 1 | Docker Compose 开发环境搭建 | ✅ 完成 | PostgreSQL 17 + Redis 7 |
| Task 2 | 数据库初始化脚本 | ✅ 完成 | users + user_roles + roles 表 |
| Task 3 | Go 后端服务骨架 | ✅ 完成 | Gin + GORM + Wire + Zap |
| Task 4 | Auth Service 层 | ✅ 完成 | 注册/登录/Token/Profile API |
| Task 5 | Auth Controller & Router | ✅ 完成 | JWT + Redis 有状态校验 |
| Task 6 | uniapp 前端骨架 | ✅ 完成 | request/storage/api/store/pages.json |
| Task 7 | uniapp 登录/注册页面 | ✅ 完成 | 基于 ui-ux-pro-max 设计系统 |
| Task 8 | 首页框架与 TabBar | ✅ 完成 | 自定义 TabBar + 路由分发 |
| Task 9 | Vue 3 管理端项目搭建 | ✅ 完成 | Element Plus + Pinia 3.x |
| Task 10 | 管理端用户管理模块 | ✅ 完成 | 后端 admin 模块 + 前端列表/详情页 |
| Task 11 | 端到端集成测试与文档 | ✅ 完成 | Dockerfile + Docker Compose + README + 全流程 API 验证 + Playwright 页面验证 + 代码审查 |

---

## 二、关键技术决策记录

### 后端（Go）
1. **框架组合**：Gin + GORM + Wire + Zap + Viper
2. **JWT 策略**：有状态 JWT，Token 存储在 Redis（`echo:auth:token:{user_id}`）
3. **密码加密**：bcrypt
4. **数据库时间精度**：`TIMESTAMP(0)` 精确到秒
5. **API 响应格式**：统一 `{ "code": 0, "message": "success", "data": ... }`
6. **常量命名**：Go camelCase（`UserStatusActive`），非大写下划线
7. **模块路由**：模块内自注册 + 中央 router 聚合

### 前台用户端（frontend/）
1. **框架**：uni-app 3.0（Vue 3.4.21 框架锁定）
2. **状态管理**：Pinia 2.1.7 + pinia-plugin-persistedstate@3
3. **npm 配置**：`.npmrc` 设置 `legacy-peer-deps=true`（uni-app 兼容性）
4. **模块系统**：ESM（`export` / `import`），禁止 CommonJS
5. **响应式单位**：`rpx`（750rpx = 屏幕宽度）
6. **设计系统**：ui-ux-pro-max 生成，持久化在 `design-system/echochat/`
7. **色板**：Primary `#2563EB` / BG `#F8FAFC` / Text `#1E293B`
8. **开发端口**：`npm run dev:h5` → localhost:5173+

### 后台管理端（admin/）
1. **框架**：Vue 3.5+ + Vite 7.x（独立项目，不受 uni-app 限制）
2. **UI 组件**：Element Plus（中文语言包）
3. **状态管理**：Pinia 3.x（最新版）
4. **HTTP 客户端**：Axios
5. **存储隔离**：localStorage key 前缀 `admin_`
6. **主题色**：CSS 变量覆盖 Element Plus → `--el-color-primary: #2563EB`
7. **开发端口**：`npm run dev` → localhost:3100（Vite proxy 代理后端）

---

## 三、目录结构概览

```
EchoChat/
├── backend/go-service/          # Go 后端服务
│   ├── app/
│   │   ├── admin/               # 管理端模块（controller/service/dao/router/provider）
│   │   ├── auth/                # 认证模块（controller/service/dao/model/router）
│   │   ├── constants/           # 常量（role_code/user_status）
│   │   ├── dto/                 # 数据传输对象（auth_dto + admin_dto）
│   │   └── provider/            # Wire 依赖注入
│   ├── cmd/server/main.go       # 入口
│   ├── config/                  # 配置
│   ├── pkg/                     # 公共包（db/logs/middleware/utils）
│   └── router/router.go         # 中央路由聚合
├── frontend/                    # 前台用户端（uni-app）
│   └── src/
│       ├── api/auth.js
│       ├── components/CustomTabBar.vue
│       ├── pages/{auth,chat,contact,meeting,profile,index}/
│       ├── store/user.js
│       └── utils/{request,storage}.js
├── admin/                       # 后台管理端（Vue 3 + Element Plus）
│   └── src/
│       ├── api/auth.js
│       ├── router/index.js
│       ├── store/user.js
│       ├── utils/{request,storage}.js
│       └── views/{layout,login,dashboard,user}/
├── deploy/
│   ├── docker-compose.dev.yml
│   └── docker/postgres/init.sql
├── design-system/echochat/      # ui-ux-pro-max 生成的设计系统
│   ├── MASTER.md
│   └── pages/{login,admin-login}.md
└── docs/
    ├── api/                     # API 接口文档
    ├── architecture/            # 系统架构文档
    ├── conventions/             # 开发规范文档
    │   └── frontend-backend-integration.md  # 前后端集成规范
    ├── plans/                   # 实施计划文档
    └── progress/                # 进度文档（本文件）
```

---

## 四、开发流程规范

1. **工作流**：使用 superpowers 流程控制开发节奏
2. **前端设计**：**必须**使用 ui-ux-pro-max 技能包，禁止手动设计
3. **代码注释**：所有公开函数、组件、Store 必须有详细注释
4. **文档同步**：代码变更后必须同步更新相关文档
5. **Git 分支**：`feature/phase1-foundation-and-auth`
6. **验证方式**：Playwright MCP 进行页面自动化验证
7. **代码审查**：每个 Task 完成后，使用 `code-reviewer` 子代理进行结构化代码审查，对照实施计划和编码标准检查
8. **完成验证**：使用 `verification-before-completion` 技能，在声称完成前必须运行验证命令并确认输出结果
9. **前后端联动规范**：详见 `docs/conventions/frontend-backend-integration.md`，核心要求：前端错误提示必须优先使用后端 message，禁止硬编码覆盖；后端错误处理禁止忽略 error；登录安全统一返回 401

---

## 五、开发测试指南

### 启动命令（开发模式）

前提：postgres 和 redis 已通过 Docker Compose 运行。

```bash
# 启动 Go 后端（http://localhost:8085）
cd backend/go-service && go run cmd/server/main.go

# 启动管理端（http://localhost:3100）
cd admin && npm run dev

# 启动前台 H5（http://localhost:5173+）
cd frontend && npm run dev:h5
```

如需全容器启动（包括 Go 服务）：`cd deploy && docker compose -f docker-compose.dev.yml up -d`

### 测试账号

| 账号 | 密码 | 角色 | 用途 |
|------|------|------|------|
| `admin_test` | `admin123456` | user + admin | **管理端登录推荐** |
| `testuser1` | `test123456` | user + admin | 前台登录测试 |
| `testuser` | `test123456` | user | 前台登录测试 |
| `testuser3` | `test123456` | user | 前台登录测试 |
| `created_by_admin` | `pass123456` | user | 管理端创建的用户 |
| `super_admin` | `admin123456` | super_admin | **系统预置唯一超管账号** |

> 也可以通过注册接口或管理端"创建用户"功能创建新的测试账号。

### 可测试功能

- **管理端**：登录 → 仪表盘 → 用户列表 → 搜索/筛选 → 用户详情 → 禁用/启用 → 分配角色 → 创建用户
- **前台 H5**：注册 → 登录 → 个人中心 → 修改资料 → 退出登录
- **API**：`GET http://localhost:8085/health`（健康检查）

---

## 六、已知问题与注意事项

1. ~~前端工具模块使用了 CommonJS，已修复为 ESM~~（已解决）
2. ~~前端登录失败时错误提示不正确（硬编码"登录已过期"覆盖后端消息）~~（已修复，详见 `docs/conventions/frontend-backend-integration.md`）
3. ~~后端登录时用户不存在返回 404 暴露用户是否注册~~（已修复为统一返回 401）
4. uni-app 的 `tabBar.custom: true` 配合自定义 TabBar 组件使用
5. 管理端和前台的 localStorage key 通过前缀隔离（`admin_` vs `echo_`）
6. Go 依赖版本需匹配 Go 1.23.12，不要随意升级 Go 工具链
7. `.gitignore` 已配置忽略：`.cursor/`、`.vite/`、`node_modules/`、`dist/`、`*.png`（根目录截图）、`logs/` 等

---

## 六、Phase 1 完成总结

### Phase 1 阶段成果
- **11 个 Task 全部完成**，端到端验证通过
- **Go 后端**：15 个 API 端点，JWT 有状态认证，RBAC 角色权限
- **前台 uni-app**：登录/注册/TabBar/个人中心
- **管理端 Vue 3**：登录/仪表盘/用户列表/用户详情
- **基础设施**：Docker Compose 一键启动（PostgreSQL + Redis + Go 服务）
- **代码审查**：code-reviewer 审查通过，已修复 panic 风险、错误处理等问题

### 下一阶段：Phase 2 - 即时通讯
- 待制定实施计划
- WebSocket 长连接 + 消息系统
- 联系人/好友管理
- 消息通知
