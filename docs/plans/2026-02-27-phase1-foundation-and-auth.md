# EchoChat 第一阶段：基础设施与用户体系 - 实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**目标：** 搭建项目完整骨架（Go 后端、uniapp 前端、Vue3 管理端），实现用户注册登录体系，建立 Docker 开发环境，为后续 IM/会议等功能奠定基础。

**架构：** Go 单体模块化服务 + uniapp 前台 + Vue3 管理端 + PostgreSQL + Redis，Docker Compose 编排。

**技术栈：** Go 1.22+ (Gin + GORM + Wire + zap) / Vue 3.4+ / uniapp 3.0 / Element Plus / PostgreSQL 17 / Redis 7 / Docker Compose

**设计文档：** `docs/plans/2026-02-27-echochat-system-design.md`

**日志系统设计：** `docs/architecture/system-architecture.md` 第八节

---

## 全局规范（贯穿所有 Task）

### 代码注释规范

以下规范适用于所有 Task 中编写的代码，必须严格遵守：

**Go 后端注释规范：**
1. **包注释**：每个 package 必须有注释，以包名开头，说明包的用途
2. **公开函数/方法**：所有大写字母开头的函数/方法必须有注释，格式：`// FuncName 描述功能`
3. **结构体注释**：每个 struct 需要注释说明用途，关键字段需要行内注释
4. **枚举/常量**：每个常量值必须有注释说明含义
5. **复杂逻辑**：非显而易见的逻辑段落需要注释说明意图和设计考虑
6. **DAO 层**：每个数据库操作函数需说明 SQL 查询逻辑和性能考虑
7. **中间件**：说明中间件的作用、执行顺序依赖和副作用

**前端注释规范（Vue/uniapp）：**
1. **组件注释**：每个 `.vue` 文件顶部说明组件用途和依赖
2. **API 函数**：每个 API 调用函数说明对应后端接口
3. **Store**：state 字段、action 方法需要注释说明
4. **工具函数**：参数和返回值类型、用途说明
5. **复杂模板逻辑**：`v-if`/`v-for` 等复杂条件需要注释说明

### 日志规范

所有 Task 中的 Go 后端代码必须遵守日志规范（详见架构文档第八节）：
1. 每个 Service/DAO 函数使用入口/出口日志模式
2. 所有日志从 context 提取 trace_id
3. 错误必须记录 ERROR 级别日志，包含上下文信息
4. 敏感信息必须脱敏

### Wire 依赖注入

从初期就使用 Wire 管理依赖注入：
1. 每个模块在 `provider.go` 中声明 Provider Set
2. 根目录 `app/provider/` 下维护全局 Wire 配置
3. 新增模块时同步更新 Wire Provider

---

## Task 1: Docker Compose 开发环境搭建

**Files:**
- Create: `deploy/docker-compose.dev.yml`
- Create: `deploy/docker/postgres/init.sql`
- Create: `deploy/docker/redis/redis.conf`
- Create: `scripts/dev-setup.sh`

**Step 1: 创建 Docker Compose 开发环境配置**

创建 `deploy/docker-compose.dev.yml`，包含 PostgreSQL 17 和 Redis 7 两个服务：

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:17-alpine
    container_name: echochat-postgres
    environment:
      POSTGRES_DB: echochat
      POSTGRES_USER: echochat
      POSTGRES_PASSWORD: echochat_dev_2026
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
      - ./docker/postgres/init.sql:/docker-entrypoint-initdb.d/init.sql
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U echochat"]
      interval: 5s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: echochat-redis
    ports:
      - "6379:6379"
    volumes:
      - redisdata:/data
      - ./docker/redis/redis.conf:/usr/local/etc/redis/redis.conf
    command: redis-server /usr/local/etc/redis/redis.conf
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  pgdata:
  redisdata:
```

**Step 2: 创建 PostgreSQL 初始化 SQL**

创建 `deploy/docker/postgres/init.sql`，包含 auth 模块相关的所有表：

```sql
-- auth_users, auth_roles, auth_user_roles 表
-- 参考设计文档第四节数据库设计
-- 同时插入默认角色: user, admin, super_admin
-- 插入默认超级管理员账号
```

**Step 3: 创建 Redis 配置文件**

创建 `deploy/docker/redis/redis.conf`，基础开发环境配置。

**Step 4: 创建开发环境搭建脚本**

创建 `scripts/dev-setup.sh`，一键启动开发环境。

**Step 5: 启动并验证开发环境**

Run: `cd deploy && docker compose -f docker-compose.dev.yml up -d`
Expected: PostgreSQL 和 Redis 正常启动，可以连接

**Step 6: Commit**

```bash
git add deploy/ scripts/
git commit -m "infra: Docker Compose 开发环境（PostgreSQL + Redis）"
```

---

## Task 2: Go 服务骨架搭建

**Files:**
- Create: `backend/go-service/cmd/server/main.go`
- Create: `backend/go-service/go.mod`
- Create: `backend/go-service/config/config.go`
- Create: `backend/go-service/config/config.dev.yaml`
- Create: `backend/go-service/pkg/db/postgres.go`
- Create: `backend/go-service/pkg/db/redis.go`
- Create: `backend/go-service/pkg/logs/logger.go`
- Create: `backend/go-service/pkg/logs/trace.go`
- Create: `backend/go-service/pkg/utils/response.go`
- Create: `backend/go-service/pkg/middleware/cors.go`
- Create: `backend/go-service/pkg/middleware/recovery.go`
- Create: `backend/go-service/pkg/middleware/logger.go`
- Create: `backend/go-service/pkg/middleware/trace.go`
- Create: `backend/go-service/app/provider/provider.go`
- Create: `backend/go-service/app/provider/wire.go`

> **日志系统详细设计见** `docs/architecture/system-architecture.md` 第八节

**Step 1: 初始化 Go Module**

Run: `cd backend/go-service && go mod init github.com/echochat/backend`

**Step 2: 创建配置管理**

创建 `config/config.go` 使用 viper 读取 YAML 配置 + 环境变量覆盖。
创建 `config/config.dev.yaml` 包含开发环境默认配置（数据库连接、Redis 连接、JWT 密钥、服务端口、日志级别等）。

```go
// config/config.go
type Config struct {
    Server   ServerConfig   `mapstructure:"server"`
    Database DatabaseConfig `mapstructure:"database"`
    Redis    RedisConfig    `mapstructure:"redis"`
    JWT      JWTConfig      `mapstructure:"jwt"`
    Log      LogConfig      `mapstructure:"log"`
}

type LogConfig struct {
    Level      string `mapstructure:"level"`       // debug/info/warn/error
    Format     string `mapstructure:"format"`      // text(开发)/json(生产)
    OutputPath string `mapstructure:"output_path"` // stdout 或文件路径
}
```

**Step 3: 创建日志系统（核心）**

创建 `pkg/logs/logger.go`：
- 基于 zap 的结构化日志封装
- 支持从 context 中提取 trace_id 自动附加到每条日志
- 提供分级日志方法：`Debug(ctx, func, msg, fields...)` / `Info(...)` / `Warn(...)` / `Error(...)`
- 敏感信息脱敏工具函数（邮箱、手机号、Token）
- 开发环境输出彩色可读文本，生产环境输出 JSON 结构化格式

创建 `pkg/logs/trace.go`：
- `GenerateTraceID()` — 生成唯一 trace ID（UUID v4 或雪花算法）
- `WithTraceID(ctx, traceID)` — 将 trace_id 注入 context
- `GetTraceID(ctx)` — 从 context 提取 trace_id

**日志输出格式示例（开发环境）：**
```
2026-02-27 10:30:00.123 INFO  [abc-123] auth | service.Login | 用户登录成功 | account=zhangsan | latency=25ms
```

**日志输出格式示例（生产环境 JSON）：**
```json
{"level":"info","ts":"2026-02-27T10:30:00.123Z","trace_id":"abc-123","module":"auth","func":"service.Login","msg":"用户登录成功","account":"zhangsan","latency_ms":25}
```

**Step 4: 创建数据库连接**

创建 `pkg/db/postgres.go` — GORM + PostgreSQL 连接池。GORM 日志适配 zap，SQL 查询日志携带 trace_id。
创建 `pkg/db/redis.go` — go-redis 客户端。Redis 操作日志携带 trace_id。

**Step 5: 创建统一响应工具**

创建 `pkg/utils/response.go` 包含 ResponseOK、ResponseBadRequest、ResponseUnauthorized、ResponseForbidden、ResponseError 等统一响应函数。响应中包含 trace_id 便于前端反馈问题时定位。

```go
type Response struct {
    Code      int         `json:"code"`
    Message   string      `json:"message"`
    Data      interface{} `json:"data,omitempty"`
    TraceID   string      `json:"trace_id,omitempty"`
    Timestamp int64       `json:"timestamp"`
}
```

**Step 6: 创建中间件**

创建 `pkg/middleware/trace.go` — **链路追踪中间件**：
  - 从请求头提取 `X-Request-ID`，不存在则自动生成
  - 注入 context，后续所有日志自动携带 trace_id
  - 在响应头中返回 `X-Request-ID`

创建 `pkg/middleware/logger.go` — **请求日志中间件**：
  - 记录每个请求的完整信息：方法、路径、状态码、耗时、IP、User-Agent
  - 自动携带 trace_id
  - 慢请求告警（>500ms 记录 WARN）

创建 `pkg/middleware/cors.go` — CORS 跨域中间件。
创建 `pkg/middleware/recovery.go` — Panic 恢复中间件（捕获 panic 后记录 ERROR 日志含堆栈信息）。

**Step 7: 初始化 Wire 依赖注入**

创建 `app/provider/provider.go` — 定义基础设施层的 Provider Set：

```go
// provider.go — 基础设施 Provider Set
package provider

import "github.com/google/wire"

// InfraSet 提供所有基础设施组件
var InfraSet = wire.NewSet(
    db.NewPostgres,
    db.NewRedis,
    logs.NewLogger,
)
```

创建 `app/provider/wire.go` — Wire 注入入口：

```go
//go:build wireinject

package provider

// InitializeApp 初始化整个应用（Wire 自动生成实现）
func InitializeApp(cfg *config.Config) (*App, error) {
    wire.Build(
        InfraSet,
        // 后续每增加一个模块，在这里添加对应的 ProviderSet
    )
    return nil, nil
}
```

安装 Wire 工具：`go install github.com/google/wire/cmd/wire@latest`
生成注入代码：`cd app/provider && wire`

**Step 8: 创建 main.go 入口**

创建 `cmd/server/main.go`：
1. 加载配置
2. 初始化日志系统（根据配置设置级别和输出格式）
3. 通过 Wire 生成的 InitializeApp 初始化所有组件
4. 创建 Gin Engine
5. 注册中间件（顺序：Trace → Logger → CORS → Recovery）
6. 注册路由（暂时只有健康检查 GET /health）
7. 启动 HTTP 服务

**Step 9: 安装依赖**

Run: `cd backend/go-service && go mod tidy`

**Step 9: 运行验证**

Run: `cd backend/go-service && go run cmd/server/main.go`
Expected: 服务启动在 :8080，访问 /health 返回 200

**Step 10: Commit**

```bash
git add backend/go-service/
git commit -m "feat(backend): Go 服务骨架（配置、日志、数据库、中间件、路由）"
```

---

## Task 3: Auth 模块 — 用户模型与数据层

**Files:**
- Create: `backend/go-service/app/auth/model/user.go`
- Create: `backend/go-service/app/auth/model/role.go`
- Create: `backend/go-service/app/auth/dao/user_dao.go`
- Create: `backend/go-service/app/auth/dao/role_dao.go`
- Create: `backend/go-service/app/auth/provider.go`
- Create: `backend/go-service/app/constants/user_status.go`
- Create: `backend/go-service/app/constants/role_code.go`

**Step 1: 创建用户状态和角色常量**

```go
// app/constants/user_status.go
const (
    USER_STATUS_ACTIVE   = 1  // 正常
    USER_STATUS_DISABLED = 2  // 禁用
    USER_STATUS_DELETED  = 3  // 注销
)

// app/constants/role_code.go
const (
    ROLE_USER        = "user"
    ROLE_ADMIN       = "admin"
    ROLE_SUPER_ADMIN = "super_admin"
)
```

**Step 2: 创建用户模型**

创建 `app/auth/model/user.go`，定义 User 结构体，JSON tag 使用下划线命名，GORM tag 指定表名 `auth_users`。

**Step 3: 创建角色模型**

创建 `app/auth/model/role.go`，定义 Role 和 UserRole 结构体。

**Step 4: 创建用户 DAO**

创建 `app/auth/dao/user_dao.go`，包含：
- `Create(ctx, user)` — 创建用户
- `FindByEmail(ctx, email)` — 按邮箱查询
- `FindByUsername(ctx, username)` — 按用户名查询
- `FindByID(ctx, id)` — 按 ID 查询
- `Update(ctx, user)` — 更新用户
- `UpdateLastLogin(ctx, userID, ip)` — 更新最后登录信息

包含标准的日志入口/出口记录。

**Step 5: 创建角色 DAO**

创建 `app/auth/dao/role_dao.go`，包含：
- `FindByCode(ctx, code)` — 按角色代码查询
- `AssignRole(ctx, userID, roleID)` — 分配角色
- `GetUserRoles(ctx, userID)` — 获取用户角色列表
- `HasRole(ctx, userID, roleCode)` — 检查用户是否拥有指定角色

**Step 6: 创建 Auth 模块 Wire Provider**

创建 `app/auth/provider.go`：

```go
// provider.go — Auth 模块依赖注入 Provider Set
package auth

import "github.com/google/wire"

// AuthSet 提供 Auth 模块的所有组件
var AuthSet = wire.NewSet(
    dao.NewUserDAO,
    dao.NewRoleDAO,
)
```

同步更新 `app/provider/wire.go`，在 `wire.Build` 中添加 `auth.AuthSet`。

**Step 7: Commit**

```bash
git add backend/go-service/app/
git commit -m "feat(auth): 用户/角色模型与数据访问层"
```

---

## Task 4: Auth 模块 — 认证服务层

**Files:**
- Create: `backend/go-service/app/auth/service/auth_service.go`
- Create: `backend/go-service/app/dto/auth_dto.go`
- Create: `backend/go-service/pkg/utils/jwt.go`
- Create: `backend/go-service/pkg/utils/password.go`
- Create: `backend/go-service/pkg/middleware/auth.go`

**Step 1: 创建密码工具**

创建 `pkg/utils/password.go`：
- `HashPassword(password)` — bcrypt 加密
- `CheckPassword(password, hash)` — 密码校验

**Step 2: 创建 JWT 工具**

创建 `pkg/utils/jwt.go`：
- `GenerateToken(userID, username)` — 生成 Access Token
- `GenerateRefreshToken(userID)` — 生成 Refresh Token
- `ParseToken(tokenStr)` — 解析验证 Token
- Claims 结构体包含 UserID, Username, Roles

**Step 3: 创建认证 DTO**

创建 `app/dto/auth_dto.go`：

```go
type RegisterRequest struct {
    Username string `json:"username" binding:"required,min=3,max=50"`
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required,min=6,max=50"`
    Nickname string `json:"nickname" binding:"max=50"`
}

type LoginRequest struct {
    Account  string `json:"account" binding:"required"`   // 邮箱或用户名
    Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
    Token        string `json:"token"`
    RefreshToken string `json:"refresh_token"`
    ExpiresIn    int64  `json:"expires_in"`
    User         UserInfo `json:"user"`
}

type UserInfo struct {
    ID       int64  `json:"id"`
    Username string `json:"username"`
    Email    string `json:"email"`
    Nickname string `json:"nickname"`
    Avatar   string `json:"avatar"`
    Roles    []string `json:"roles"`
}
```

**Step 4: 创建认证服务**

创建 `app/auth/service/auth_service.go`：
- `Register(ctx, req)` — 注册逻辑（检查重复、加密密码、创建用户、分配默认角色、生成 Token）
- `Login(ctx, req)` — 登录逻辑（查找用户、校验密码、检查状态、更新登录信息、生成 Token）
- `RefreshToken(ctx, refreshToken)` — 刷新 Token
- `GetProfile(ctx, userID)` — 获取用户信息
- `UpdateProfile(ctx, userID, req)` — 更新用户信息
- `ChangePassword(ctx, userID, req)` — 修改密码

**Step 5: 创建认证中间件**

创建 `pkg/middleware/auth.go`：
- `JWTAuth()` — JWT 验证中间件，从 Authorization Header 提取 Token
- `RequireRole(roles ...string)` — 角色检查中间件
- 将 UserID 和 Roles 注入 Gin Context

**Step 6: 更新 Wire Provider**

更新 `app/auth/provider.go`，添加 Service 层 Provider：

```go
var AuthSet = wire.NewSet(
    dao.NewUserDAO,
    dao.NewRoleDAO,
    service.NewAuthService,
)
```

**Step 7: Commit**

```bash
git add backend/go-service/
git commit -m "feat(auth): 认证服务层（注册、登录、JWT、密码加密、中间件）"
```

---

## Task 5: Auth 模块 — Controller 与路由注册

**Files:**
- Create: `backend/go-service/app/auth/controller/auth_controller.go`
- Create: `backend/go-service/app/auth/controller/admin_auth_controller.go`
- Create: `backend/go-service/app/auth/router.go`
- Modify: `backend/go-service/cmd/server/main.go` — 注册 auth 路由
- Modify: `backend/go-service/app/auth/provider.go` — 添加 Controller Provider

**Step 1: 创建前台认证 Controller**

创建 `app/auth/controller/auth_controller.go`：
- `Register(c *gin.Context)` — POST /api/v1/auth/register
- `Login(c *gin.Context)` — POST /api/v1/auth/login
- `Logout(c *gin.Context)` — POST /api/v1/auth/logout
- `RefreshToken(c *gin.Context)` — POST /api/v1/auth/refresh-token
- `GetProfile(c *gin.Context)` — GET /api/v1/auth/profile
- `UpdateProfile(c *gin.Context)` — PUT /api/v1/auth/profile
- `ChangePassword(c *gin.Context)` — PUT /api/v1/auth/password

每个方法：参数绑定 → 调用 Service → 统一响应返回。

**Step 2: 创建后台认证 Controller**

创建 `app/auth/controller/admin_auth_controller.go`：
- `AdminLogin(c *gin.Context)` — POST /api/v1/admin/auth/login
  - 与普通登录的区别：登录后检查用户是否拥有 admin 或 super_admin 角色

**Step 3: 创建路由注册函数**

创建 `app/auth/router.go`：

```go
func RegisterRoutes(r *gin.Engine, ctrl *AuthController, adminCtrl *AdminAuthController, authMiddleware gin.HandlerFunc) {
    // 前台公开路由
    public := r.Group("/api/v1/auth")
    {
        public.POST("/register", ctrl.Register)
        public.POST("/login", ctrl.Login)
        public.POST("/refresh-token", ctrl.RefreshToken)
    }

    // 前台需认证路由
    authed := r.Group("/api/v1/auth").Use(authMiddleware)
    {
        authed.POST("/logout", ctrl.Logout)
        authed.GET("/profile", ctrl.GetProfile)
        authed.PUT("/profile", ctrl.UpdateProfile)
        authed.PUT("/password", ctrl.ChangePassword)
    }

    // 后台认证路由
    admin := r.Group("/api/v1/admin/auth")
    {
        admin.POST("/login", adminCtrl.AdminLogin)
    }
}
```

**Step 4: 在 main.go 中注册路由**

修改 `cmd/server/main.go`，初始化 auth 模块的 DAO → Service → Controller 链，调用 RegisterRoutes。

**Step 5: 运行验证**

Run: `cd backend/go-service && go run cmd/server/main.go`
Expected: 服务启动，可以通过 curl 测试注册和登录接口

```bash
# 测试注册
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","email":"test@example.com","password":"123456","nickname":"测试用户"}'

# 测试登录
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"account":"testuser","password":"123456"}'
```

Expected: 注册返回 Token，登录返回 Token + 用户信息

**Step 6: Commit**

```bash
git add backend/go-service/
git commit -m "feat(auth): 用户注册/登录 API（Controller + 路由注册）"
```

---

## Task 6: uniapp 前端骨架完善

**Files:**
- Create: `frontend/src/utils/request.js`
- Create: `frontend/src/utils/storage.js`
- Create: `frontend/src/api/auth.js`
- Create: `frontend/src/store/user.js`
- Create: `frontend/src/services/websocket.js`（占位）
- Modify: `frontend/src/pages.json`
- Modify: `frontend/src/main.js`
- Modify: `frontend/package.json`

**Step 1: 安装前端依赖**

Run: `cd frontend && npm install pinia @pinia/persist`

uniapp 内置了请求 API（uni.request），不需要额外安装 axios。

**Step 2: 创建 HTTP 请求封装**

创建 `src/utils/request.js`：
- 封装 uni.request 为 Promise
- 请求拦截器：自动添加 Authorization Header
- 响应拦截器：统一处理错误码（401 跳转登录等）
- 基础 URL 配置

**Step 3: 创建存储工具**

创建 `src/utils/storage.js`：
- 封装 uni.setStorageSync / uni.getStorageSync
- Token 存取方法

**Step 4: 创建 Pinia Store**

修改 `src/main.js` 集成 Pinia。
创建 `src/store/user.js`：
- state: userInfo, token, isLoggedIn
- actions: login, register, logout, getProfile, setToken

**Step 5: 创建 Auth API**

创建 `src/api/auth.js`：
- register(data)
- login(data)
- logout()
- getProfile()
- updateProfile(data)
- changePassword(data)

**Step 6: 更新 pages.json**

添加 auth 页面路由、TabBar 配置。

**Step 7: Commit**

```bash
git add frontend/
git commit -m "feat(frontend): uniapp 前端骨架（请求封装、状态管理、Auth API）"
```

---

## Task 7: uniapp 登录/注册页面

**Files:**
- Create: `frontend/src/pages/auth/login.vue`
- Create: `frontend/src/pages/auth/register.vue`
- Modify: `frontend/src/pages.json`

> **Note:** 页面构建使用 ui-ux-pro-max skill 生成设计系统后实现。uniapp 使用 `<view>` `<text>` `<input>` 等组件而非 HTML 标签。

**Step 1: 使用 ui-ux-pro-max 生成设计系统**

```bash
python3 skills/ui-ux-pro-max/scripts/search.py "video conference meeting chat modern professional" --design-system -p "EchoChat" --persist
```

然后获取 Vue stack 的实现指南：
```bash
python3 skills/ui-ux-pro-max/scripts/search.py "login register form" --stack vue
```

**Step 2: 创建登录页面**

创建 `src/pages/auth/login.vue`：
- 输入框：账号（邮箱或用户名）+ 密码
- 登录按钮
- "没有账号？去注册" 链接
- 表单验证
- 调用 userStore.login()
- 登录成功跳转首页

**Step 3: 创建注册页面**

创建 `src/pages/auth/register.vue`：
- 输入框：用户名 + 邮箱 + 密码 + 确认密码 + 昵称（选填）
- 注册按钮
- "已有账号？去登录" 链接
- 表单验证
- 调用 userStore.register()
- 注册成功自动登录

**Step 4: 更新 pages.json 路由配置**

**Step 5: 运行验证**

Run: `cd frontend && npm run dev:h5`
Expected: H5 浏览器中可以看到登录/注册页面，输入信息后可以成功注册和登录

**Step 6: Commit**

```bash
git add frontend/
git commit -m "feat(frontend): 登录/注册页面"
```

---

## Task 8: uniapp 首页框架与 TabBar

**Files:**
- Create: `frontend/src/pages/chat/index.vue`（占位）
- Create: `frontend/src/pages/contact/index.vue`（占位）
- Create: `frontend/src/pages/meeting/index.vue`（占位）
- Create: `frontend/src/pages/profile/index.vue`
- Modify: `frontend/src/pages/index/index.vue`
- Modify: `frontend/src/pages.json`

**Step 1: 创建 TabBar 页面占位**

创建消息、联系人、会议三个 Tab 页面的占位版本（显示模块名称和"开发中"提示）。

**Step 2: 创建个人中心页面**

创建 `src/pages/profile/index.vue`：
- 显示用户头像、昵称、用户名
- 编辑资料入口
- 修改密码入口
- 退出登录按钮
- 使用 ui-ux-pro-max 设计系统

**Step 3: 配置 TabBar**

修改 `pages.json`，配置底部 TabBar：消息 | 联系人 | 会议 | 我的

**Step 4: 配置首页重定向**

修改 `src/pages/index/index.vue`，判断登录状态：
- 已登录 → 跳转消息列表页
- 未登录 → 跳转登录页

**Step 5: 运行验证**

Run: `cd frontend && npm run dev:h5`
Expected: 未登录时显示登录页，登录后显示 TabBar 页面，个人中心可以查看信息并退出

**Step 6: Commit**

```bash
git add frontend/
git commit -m "feat(frontend): TabBar 框架与个人中心页面"
```

---

## Task 9: Vue 3 管理端项目搭建

**Files:**
- Create: `admin/` 整个目录（通过 Vite 脚手架创建）
- 项目结构参考设计文档第三节

**Step 1: 创建 Vue 3 项目**

Run: `cd /path/to/EchoChat && npm create vite@latest admin -- --template vue`

**Step 2: 安装核心依赖**

Run: `cd admin && npm install element-plus vue-router@4 pinia axios @element-plus/icons-vue`

**Step 3: 创建布局框架**

创建 `src/views/layout/index.vue`：
- 左侧边栏导航
- 顶部栏（管理员信息、退出登录）
- 主内容区
- 使用 Element Plus 的 Container/Aside/Header/Main 布局

**Step 4: 创建路由配置**

创建 `src/router/index.js`：
- 登录页路由
- 后台布局路由（嵌套子路由）
- 路由守卫（未登录跳转登录页）

**Step 5: 创建请求封装**

创建 `src/api/request.js`：
- 基于 axios 的请求封装
- 请求/响应拦截器
- Token 自动附加

**Step 6: 创建管理员登录页**

创建 `src/views/login.vue`：
- 管理员账号 + 密码登录表单
- 使用 Element Plus 表单组件
- 使用 ui-ux-pro-max 设计系统

**Step 7: 创建仪表盘占位页**

创建 `src/views/dashboard/index.vue`：
- 显示统计卡片占位（总用户数、在线用户、今日会议等）

**Step 8: 运行验证**

Run: `cd admin && npm run dev`
Expected: 浏览器访问管理端，未登录跳转登录页，登录后进入仪表盘

**Step 9: Commit**

```bash
git add admin/
git commit -m "feat(admin): Vue 3 管理端项目搭建（布局、路由、登录页）"
```

---

## Task 10: 管理端用户管理模块

**Files:**
- Create: `backend/go-service/app/admin/controller/user_manage_controller.go`
- Create: `backend/go-service/app/admin/service/user_manage_service.go`
- Create: `backend/go-service/app/admin/dao/user_manage_dao.go`
- Create: `backend/go-service/app/admin/router.go`
- Create: `backend/go-service/app/dto/admin_dto.go`
- Create: `admin/src/api/user.js`
- Create: `admin/src/views/user/list.vue`
- Create: `admin/src/views/user/detail.vue`
- Modify: `backend/go-service/cmd/server/main.go`

**Step 1: 创建管理端 DTO**

创建 `app/dto/admin_dto.go`：
- UserListRequest（分页、搜索、状态筛选）
- UserListResponse（用户列表 + 总数）
- UserDetailResponse

**Step 2: 创建管理端用户 DAO**

创建 `app/admin/dao/user_manage_dao.go`：
- `ListUsers(ctx, req)` — 分页查询用户列表（支持搜索、状态筛选）
- `CountUsers(ctx)` — 统计用户总数
- `UpdateUserStatus(ctx, userID, status)` — 更新用户状态

**Step 3: 创建管理端用户 Service**

创建 `app/admin/service/user_manage_service.go`：
- `GetUserList(ctx, req)` — 获取用户列表
- `GetUserDetail(ctx, userID)` — 获取用户详情（含角色信息）
- `UpdateUserStatus(ctx, userID, status)` — 启用/禁用用户
- `AssignUserRole(ctx, userID, roleCode)` — 分配角色
- `CreateUser(ctx, req)` — 管理员手动创建用户

**Step 4: 创建管理端 Controller 和路由**

创建 Controller 和路由注册函数，所有管理端 API 需要 JWT + admin 角色双重中间件。

**Step 5: 创建管理端用户列表页面**

创建 `admin/src/views/user/list.vue`：
- 搜索框（用户名/邮箱搜索）
- 状态筛选下拉
- 用户数据表格（Element Plus Table）
- 分页组件
- 操作按钮（查看详情、启用/禁用）

**Step 6: 创建管理端用户详情页面**

创建 `admin/src/views/user/detail.vue`：
- 基本信息 Tab
- 角色管理
- 会议记录 Tab（占位，后续完善）

**Step 7: 运行验证**

Expected: 管理端可以查看用户列表、搜索筛选、查看详情、启用/禁用用户

**Step 8: Commit**

```bash
git add backend/go-service/app/admin/ admin/src/
git commit -m "feat(admin): 用户管理模块（列表、详情、状态管理、角色分配）"
```

---

## Task 11: 端到端集成测试与文档

**Files:**
- Create: `backend/go-service/Dockerfile`
- Modify: `deploy/docker-compose.dev.yml` — 添加 go-service
- Create: `README.md` — 项目说明文档

**Step 1: 创建 Go 服务 Dockerfile**

创建 `backend/go-service/Dockerfile`，多阶段构建：
- 第一阶段：Go 编译
- 第二阶段：alpine 运行镜像

**Step 2: 更新 Docker Compose**

在 `deploy/docker-compose.dev.yml` 中添加 go-service 服务。

**Step 3: 编写项目 README**

更新根目录 `README.md`：
- 项目简介
- 技术栈
- 项目结构
- 开发环境搭建步骤
- API 文档链接

**Step 4: 完整流程验证**

1. Docker Compose 启动所有服务
2. 访问前台 H5 → 注册 → 登录 → 查看个人中心 → 退出
3. 访问管理端 → 管理员登录 → 查看用户列表 → 查看用户详情

**Step 5: Commit**

```bash
git add .
git commit -m "feat: 第一阶段完成（基础设施 + 用户体系 + 管理端用户管理）"
```

---

## 阶段完成标准

- [x] Docker Compose 开发环境可一键启动（PostgreSQL + Redis）
- [x] Go 后端服务骨架完整（配置、日志、数据库、中间件）
- [x] 用户注册/登录 API 正常工作（邮箱+密码、用户名+密码）
- [x] JWT Token 认证机制正常（Access Token + Refresh Token）
- [x] RBAC 角色权限体系正常（user / admin / super_admin）
- [x] uniapp 前端骨架完整（请求封装、状态管理、路由）
- [x] 前台登录/注册页面可用
- [x] 前台 TabBar 框架和个人中心可用
- [x] Vue 3 管理端项目搭建完成（布局、路由、登录）
- [x] 管理端用户管理模块可用（列表、详情、状态管理）
- [x] 所有代码已提交 Git
