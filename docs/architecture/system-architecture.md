# EchoChat 系统架构设计

> 本文档从整体设计方案中提取并深化架构设计部分，便于独立查阅。
> 完整设计方案见 `docs/plans/2026-02-27-echochat-system-design.md`

---

## 一、架构概述

EchoChat 采用 **「精简单体 + 媒体微服务」** 架构，核心思想是 **控制面与媒体面彻底分离、业务系统与实时系统解耦**。

- **Go 单体服务**：处理所有业务逻辑（认证、IM、会议控制、好友、通知、后台管理），内部按模块化组织，保留后期拆分为微服务的能力
- **mediasoup Node 服务**：独立的媒体控制微服务，管理 SFU Worker，不涉及任何业务逻辑
- **mediasoup Worker**：C++ SFU 引擎，负责 RTP 转发、拥塞控制、带宽自适应

---

## 二、架构分层图

```
┌─────────────────────────────────────────────────────────────┐
│                     接入层 (Nginx)                            │
│  SSL 终止 · 反向代理 · WebSocket 升级 · 静态资源 · 负载均衡    │
└─────┬───────────────────────┬───────────────────────────────┘
      │                       │
      │  HTTPS / WSS          │  HTTPS
      │                       │
┌─────┴──────────┐     ┌──────┴──────────┐
│  前台用户端     │     │  后台管理端       │
│  uniapp        │     │  Vue3+Element    │
│  (H5/App/小程序)│     │  Plus (PC Web)   │
└─────┬──────────┘     └──────┬──────────┘
      │                       │
      │  WebSocket + HTTP     │  HTTP (RESTful)
      │                       │
┌─────┴───────────────────────┴──────────────────────────────┐
│                    Go 单体服务（模块化）                       │
│                                                              │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐   │
│  │  auth    │ │   im     │ │ meeting  │ │   admin      │   │
│  │ 认证鉴权  │ │ 即时通讯  │ │ 会议控制  │ │  后台管理    │   │
│  └──────────┘ └──────────┘ └──────────┘ └──────────────┘   │
│  ┌──────────┐ ┌──────────┐                                  │
│  │ contact  │ │  notify  │     每个模块: Controller →        │
│  │ 联系人   │ │  通知    │     Service → DAO → Model         │
│  └──────────┘ └──────────┘                                  │
│       │              │              │                        │
│  ┌────┴──────────────┴──────────────┴──────┐                │
│  │          公共基础设施层 (pkg/)            │                │
│  │  db · redis · ws · middleware · utils    │                │
│  └─────────────────────────────────────────┘                │
└────────┬──────────────────────┬─────────────────────────────┘
         │                      │
    PostgreSQL              Redis              HTTP
    (持久化数据)          (实时状态)              │
                                          ┌─────┴─────────────┐
                                          │ mediasoup Node 服务 │
                                          │  Router 管理        │
                                          │  Transport 管理     │
                                          │  Producer/Consumer  │
                                          └─────┬─────────────┘
                                                │ IPC
                                          ┌─────┴─────────────┐
                                          │ mediasoup Worker   │
                                          │  (C++ SFU 引擎)    │
                                          │  RTP 转发          │
                                          │  拥塞控制          │
                                          │  带宽自适应        │
                                          └───────────────────┘
```

---

## 三、各层职责说明

### 3.1 接入层 (Nginx)

| 职责 | 说明 |
|------|------|
| SSL 终止 | 处理 HTTPS/WSS 加密，内部服务间通信使用 HTTP |
| 反向代理 | 将请求分发到 Go 服务或前端静态资源 |
| WebSocket 升级 | 处理 WebSocket 协议升级 |
| 负载均衡 | 后期多实例部署时进行请求分发 |

### 3.2 Go 单体服务

系统的 **"大脑"**，处理所有业务逻辑。

| 模块 | 职责 |
|------|------|
| auth | 用户注册/登录、有状态 JWT Token 管理（Redis 存储）、RBAC 角色权限（当前粗粒度 3 角色，预留细粒度权限点扩展） |
| im | 即时消息收发、会话管理、消息存储 |
| contact | 好友关系管理、好友分组 |
| meeting | 会议创建/管理、信令转发、mediasoup 资源编排 |
| notify | 通知推送、会议邀请、好友申请通知 |
| admin | 后台管理（用户管理、会议监控、系统配置） |

**不负责的事情：** 不处理 RTP 媒体数据、不参与音视频转发、不做 WebRTC 协议协商。

#### 3.2.1 路由架构

路由采用 **"模块自包含 + 主路由汇总"** 模式，每个模块在自己的目录内维护路由定义，主路由文件仅做注册汇总。这样设计的核心目的是**利于微服务拆分**——拆分时整个模块目录原封不动搬走即可。

**目录结构：**

```
router/
└── router.go               ← 主路由入口，只做汇总注册（不含具体路由定义）

app/auth/router.go          ← auth 模块的具体路由定义
app/admin/router.go         ← admin 模块的具体路由定义
app/im/router.go            ← im 模块的具体路由定义（后续阶段）
app/contact/router.go       ← contact 模块的具体路由定义（后续阶段）
app/meeting/router.go       ← meeting 模块的具体路由定义（后续阶段）
```

**调用关系：**

```
main.go
  └── router.Setup(engine, app)
        ├── engine.GET("/health", ...)                  // 健康检查
        ├── auth.RegisterRoutes(engine, ...)             // /api/v1/auth/*
        ├── admin.RegisterRoutes(engine, ...)            // /api/v1/admin/*
        ├── im.RegisterRoutes(engine, ...)               // /api/v1/im/*       （后续）
        ├── contact.RegisterRoutes(engine, ...)          // /api/v1/contact/*  （后续）
        └── meeting.RegisterRoutes(engine, ...)          // /api/v1/meeting/*  （后续）
```

**路由命名规范：**

| 端 | 路径前缀 | 中间件 | 说明 |
|---|---------|--------|------|
| 前台公开 | `/api/v1/auth/*` | 无 | 注册、登录等不需要认证 |
| 前台认证 | `/api/v1/{module}/*` | JWT 认证 | 需要登录后访问 |
| 管理端 | `/api/v1/admin/{module}/*` | JWT + admin 角色 | 需要管理员权限 |

**微服务拆分时的变化：** 每个独立服务的 main.go 直接调用自己模块的 `RegisterRoutes`，不再需要主路由汇总文件。

### 3.3 mediasoup Node 服务

mediasoup C++ SFU 的 **"遥控器"**，不懂业务、不懂用户、只懂媒体对象。

| 职责 | 说明 |
|------|------|
| Worker 管理 | 创建和管理 mediasoup C++ Worker 进程 |
| Router 管理 | 每个会议房间对应一个 Router |
| Transport 管理 | 为每个参与者创建 WebRTC Transport |
| Producer/Consumer | 管理音视频流的推送和消费 |

### 3.4 mediasoup Worker

真正的 **"发动机"**，纯 C++ 实现的 SFU 引擎。

| 职责 | 说明 |
|------|------|
| RTP 转发 | 接收发送者的 RTP 包，转发给所有接收者 |
| 拥塞控制 | 根据网络状况动态调整 |
| 带宽自适应 | Simulcast/SVC 支持 |

---

## 四、数据流说明

### 4.1 即时消息流

```
客户端A                Go 服务                     客户端B
   │                     │                           │
   │── WS: 发送消息 ──→  │                           │
   │                     │── 写入 PostgreSQL          │
   │                     │── 更新 Redis 未读数        │
   │  ←── WS: 发送确认 ──│                           │
   │                     │── WS: 推送新消息 ────────→ │
   │                     │                           │
```

### 4.2 音视频会议流

```
客户端               Go 服务           mediasoup Node     Worker
  │                    │                    │               │
  │── HTTP: 加入会议 →  │                    │               │
  │                    │── HTTP: 创建Router→│               │
  │                    │  ←── RTP能力 ──────│               │
  │ ←── WS: 房间信息 ──│                    │               │
  │                    │                    │               │
  │── WS: 创建Transport→│                   │               │
  │                    │── HTTP: 创建 ────→ │── IPC ──────→ │
  │ ←── WS: Transport参数│                  │               │
  │                    │                    │               │
  │── WS: 开始推流 ──→  │                    │               │
  │                    │── HTTP: Producer → │── IPC ──────→ │
  │                    │                    │               │
  │════════════════ RTP/DTLS 媒体流直连 ═══════════════════→│
  │  (音视频数据不经过 Go 服务，直连 Worker)                  │
```

---

## 五、微服务演进路径

当前架构从第一天起就为微服务拆分做了准备：

### 5.1 代码层面的预留

| 规则 | 说明 |
|------|------|
| 模块间零直接引用 | `auth` 不会 import `im` 内部代码，通过 interface 通信 |
| 模块自包含路由 | 每个模块在自己目录内维护 `router.go`，拆分时整目录搬走 |
| 主路由仅做汇总 | `router/router.go` 只注册各模块路由，不含具体路由定义 |
| 数据库表按模块前缀 | `auth_users`、`im_messages`、`meeting_rooms`，后期可分库 |
| Redis key 按命名空间 | `echo:auth:*`、`echo:im:*`、`echo:meeting:*` |

### 5.2 演进路径

```
第一阶段（当前）           第二阶段                 第三阶段
┌─────────────┐      ┌──────────────────┐     ┌─────────────────┐
│ Go 单体服务  │  →   │ Go 实时服务       │ →   │ auth-service    │
│ (模块化)     │      │ (信令+会议+IM)    │     │ im-service      │
│             │      │                  │     │ meeting-service  │
│             │      │ Go 业务服务       │     │ contact-service  │
│             │      │ (用户+好友+管理)  │     │ admin-service    │
└─────────────┘      └──────────────────┘     └─────────────────┘
                                                + API Gateway
                                                + 服务发现
                                                + 链路追踪
```

---

## 六、部署架构

### 6.1 开发环境 (Docker Compose)

```yaml
services:
  go-service:      # Go 后端         → :8085
  media-server:    # mediasoup Node  → :3000 + :40000-40100/udp
  postgres:        # PostgreSQL 17   → :5432
  redis:           # Redis 7         → :6379
  nginx:           # 反向代理         → :80/:443
```

### 6.2 生产环境 (预留 K8s)

```
┌─────────────────────────────────────────┐
│              Kubernetes 集群             │
│                                         │
│  ┌─────────┐  ┌─────────┐              │
│  │ Go Pod  │  │ Go Pod  │  (水平扩展)   │
│  │ (副本1) │  │ (副本N) │              │
│  └────┬────┘  └────┬────┘              │
│       └──────┬─────┘                    │
│              │                          │
│  ┌───────────┴───────────┐              │
│  │     Service (LB)      │              │
│  └───────────────────────┘              │
│                                         │
│  ┌──────────────┐  ┌──────────────┐     │
│  │ mediasoup    │  │ mediasoup    │     │
│  │ Pod (副本1)  │  │ Pod (副本N)  │     │
│  └──────────────┘  └──────────────┘     │
│                                         │
│  PostgreSQL (StatefulSet / 外部 RDS)     │
│  Redis (StatefulSet / 外部 ElastiCache)  │
└─────────────────────────────────────────┘
```

---

## 七、技术选型依据

| 技术 | 选型理由 |
|------|---------|
| **Go (Gin)** | 高并发、静态编译、内存占用小，适合实时系统 |
| **GORM** | Go 生态最成熟的 ORM，社区活跃 |
| **Wire** | 编译时依赖注入，零运行时开销 |
| **zap** | 高性能结构化日志，Uber 出品 |
| **Viper** | 配置管理标准库，支持 YAML + 环境变量覆盖 |
| **mediasoup** | 最高性能的开源 SFU，C++ 实现 |
| **PostgreSQL 17** | 强一致性、JSONB 支持、性能优异 |
| **Redis 7** | 实时状态存储、发布订阅、高速缓存 |
| **uniapp (Vue 3.4)** | 一套代码多端运行（H5/App/小程序），Vue 版本由 uni-app 框架锁定 |
| **Element Plus** | Vue 3 生态最成熟的 PC 端组件库，管理端使用 |
| **Docker Compose** | 轻量级容器编排，适合初期和开发环境 |

### 7.1 前端技术栈版本策略

系统包含两个独立前端项目，因框架约束，版本策略有所区别：

| 前端项目 | 目录 | 框架 | Vue 版本 | 状态管理 | 原因 |
|----------|------|------|---------|---------|------|
| 前台用户端 | `frontend/` | uni-app 3.0 | 3.4.21（框架锁定） | Pinia 2.x | uni-app 尚未适配 Vue 3.5，Pinia 3.x 需要 Vue >= 3.5.11 |
| 后台管理端 | `admin/` | Vue 3 + Element Plus | 3.5+（不受限） | Pinia 3.x | 独立 Vue 3 项目，可使用最新版本 |

**npm 兼容性说明：** uni-app 的 peer dependency 链与 npm 7+ 的严格依赖解析存在冲突，`frontend/` 项目需在 `.npmrc` 中设置 `legacy-peer-deps=true`，这是 uni-app 社区的标准做法。

**版本升级策略：** 当 uni-app 正式适配 Vue 3.5+ 后，可统一将前台的 Pinia 升级至 3.x，API 层面几乎无需改动（Pinia 2.x 与 3.x 的 `defineStore` API 完全兼容）。

### 7.2 两端开发规范差异

前台用户端和后台管理端是**完全独立**的前端项目，开发规范、依赖版本、构建方式不需要强制统一：

| 维度 | 前台用户端 (`frontend/`) | 后台管理端 (`admin/`) |
|------|------------------------|---------------------|
| 框架 | uni-app 3.0（Vue 3.4.21） | Vue 3.5+ + Vite |
| 状态管理 | Pinia 2.x（`pinia-plugin-persistedstate@3`） | Pinia 3.x（最新版） |
| HTTP 客户端 | `uni.request` 封装 | Axios |
| 路由 | `pages.json` + `uni.navigateTo/switchTab` | Vue Router 4 |
| UI 组件 | 原生 uni-app 组件（`<view>`/`<text>`/`<input>`） | Element Plus |
| 模块系统 | ES Module（Vite 要求） | ES Module（Vite 标准） |
| npm 配置 | 需要 `.npmrc` 设置 `legacy-peer-deps=true` | 标准 npm 配置 |
| 适配能力 | H5 / 小程序 / App / 桌面端 | 仅 PC 浏览器 |
| 设计系统 | ui-ux-pro-max 生成的设计系统 | ui-ux-pro-max + Element Plus 主题 |

---

## 八、日志系统与链路追踪设计

### 8.1 设计目标

- 每一个请求从进入系统到完成响应，全链路可追踪
- Go 服务内部的函数调用链清晰可见
- Go 服务与 mediasoup Node 服务之间的调用可关联
- WebSocket 消息的处理过程可追踪
- 日志分级明确，开发/生产环境日志策略不同
- 日志格式统一、结构化，便于后期接入 ELK/Loki 等日志平台

### 8.2 请求链路追踪（Trace ID / Request ID）

每个请求在进入系统时生成唯一的 `trace_id`，贯穿整个处理链路：

```
客户端请求
    │
    ▼
Nginx（生成或转发 X-Request-ID）
    │
    ▼
Go 中间件（提取或生成 trace_id，注入 context）
    │
    ├── Controller 日志：[trace_id] 收到请求 POST /api/v1/auth/login
    ├── Service 日志：  [trace_id] 开始处理登录，account=zhangsan
    ├── DAO 日志：      [trace_id] SQL查询 auth_users WHERE username=zhangsan
    ├── Redis 日志：    [trace_id] SET echo:auth:token:1
    │
    ├── 如果涉及 mediasoup 调用：
    │   Go 将 trace_id 放入 HTTP Header 传给 Node 服务
    │   Node 服务日志也携带同一个 trace_id
    │
    ▼
Go 中间件：[trace_id] 请求完成 200 OK 耗时 25ms
```

**实现方式：**

```go
// 中间件：为每个请求生成 trace_id 并注入 context
func TraceMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        traceID := c.GetHeader("X-Request-ID")
        if traceID == "" {
            traceID = generateTraceID()  // UUID 或 雪花算法
        }
        ctx := context.WithValue(c.Request.Context(), "trace_id", traceID)
        c.Request = c.Request.WithContext(ctx)
        c.Header("X-Request-ID", traceID)
        c.Next()
    }
}
```

### 8.3 日志级别规范

| 级别 | 使用场景 | 示例 |
|------|---------|------|
| **DEBUG** | 开发调试信息，生产环境不输出 | 函数参数详情、SQL 语句、Redis 操作 |
| **INFO** | 正常业务流程的关键节点 | 用户登录成功、会议创建、消息发送 |
| **WARN** | 异常但不影响主流程 | 参数格式警告、缓存未命中、重试操作 |
| **ERROR** | 业务错误，需要关注 | 数据库查询失败、外部服务调用失败 |
| **FATAL** | 系统级致命错误 | 数据库连接失败、配置加载失败（启动时） |

**环境日志策略：**

| 环境 | 最低级别 | 输出方式 | 格式 |
|------|---------|---------|------|
| 开发 | DEBUG | 控制台（彩色文本）+ 文件（JSON） | 控制台 text / 文件 JSON |
| 生产 | INFO | 控制台（JSON）+ 文件（JSON） | JSON 结构化 |

**日志文件输出（基于 lumberjack 轮转）：**

| 文件 | 级别 | 用途 |
|------|------|------|
| `logs/app.log` | 全量（与配置级别一致） | 持久化存储、接入 ELK/Loki |
| `logs/error.log` | 仅 WARN + ERROR | 快速定位问题 |

**日志文件轮转策略：**

| 配置项 | 开发默认值 | 生产建议值 | 说明 |
|--------|-----------|-----------|------|
| `max_size` | 50 MB | 200 MB | 单文件最大大小，超过自动切割 |
| `max_backups` | 10 | 30 | 保留的旧日志文件数量 |
| `max_age` | 30 天 | 90 天 | 旧日志保留天数 |
| `compress` | false | true | 是否压缩归档文件 |

### 8.4 日志格式规范

**结构化日志字段（每条日志必含）：**

```json
{
    "level": "info",
    "ts": "2026-02-27 10:30:00",
    "trace_id": "abc-123-def-456",
    "module": "auth",
    "func": "service.auth_service.Login",
    "msg": "用户登录成功",
    "user_id": 1,
    "ip": "192.168.1.100",
    "latency_ms": 25
}
```

### 8.5 函数级日志规范

每个 Service 和 DAO 层的关键函数遵循统一的日志模式：

```go
func (s *AuthService) Login(ctx context.Context, req *dto.LoginRequest) (*dto.LoginResponse, error) {
    funcName := "service.auth_service.Login"

    // 入口日志：记录关键入参（脱敏）
    logs.Info(ctx, funcName, "开始处理登录",
        zap.String("account", req.Account),
    )

    var err error
    defer func() {
        if err != nil {
            // 出口日志（失败）：记录错误信息
            logs.Error(ctx, funcName, "登录处理失败",
                zap.String("account", req.Account),
                zap.Error(err),
            )
        } else {
            // 出口日志（成功）
            logs.Info(ctx, funcName, "登录处理完成",
                zap.String("account", req.Account),
            )
        }
    }()

    // 业务逻辑...
    return resp, err
}
```

### 8.6 请求日志中间件（Access Log 层）

HTTP 请求日志中间件采用社区标准的「分层不重复」策略，只记录 HTTP 请求级元信息，不记录请求 Body。

#### 分层日志架构

| 层级 | 职责 | caller 指向 | 记录内容 |
|------|------|------------|---------|
| **中间件层（Access Log）** | HTTP 请求元信息 | 中间件源码 | method / path / handler / status / latency / ip / query |
| **Controller 层** | 结构化请求参数 | Controller 代码行号 | ShouldBindJSON 后的业务参数（可精准脱敏） |
| **Service 层** | 业务逻辑关键节点 | Service 代码行号 | 业务状态变更、外部调用等 |
| **DAO 层** | 数据操作 | DAO 代码行号 | SQL 操作、缓存操作等 |

> 同一请求的各层日志通过 `trace_id` 串联，可完整追踪请求的处理链路。

#### 中间件记录字段

| 字段 | 说明 | 记录时机 |
|------|------|---------|
| `method` | HTTP 方法 | 始终记录 |
| `path` | 请求路径 | 始终记录 |
| `handler` | 处理函数名（如 `controller.auth_controller.Login`） | 始终记录 |
| `status` | HTTP 状态码 | 始终记录 |
| `latency` | 请求耗时 | 始终记录 |
| `ip` | 客户端 IP | 始终记录 |
| `user_agent` | 客户端 User-Agent | 始终记录 |
| `query` | URL Query 参数 | 有 Query 时记录 |
| `resp_body` | 响应 Body（最大 2KB） | 仅 4xx/5xx 错误响应 |
| `error` | Gin 错误信息 | 有 c.Errors 时记录 |

#### 状态码分级输出

| 情况 | 日志级别 |
|------|---------|
| 5xx 服务器错误 | ERROR |
| 4xx 客户端错误 | WARN |
| 请求耗时 > 500ms | WARN（慢请求告警） |
| 正常请求 | INFO |

#### 日志输出示例

**正常 GET 请求（带 Query 参数）：**

```json
{
    "level": "INFO",
    "ts": "2026-02-28 16:08:40",
    "caller": "middleware/logger.go:108",
    "trace_id": "b94a08f8-3c1e-4567-96ab-7bfe1057239a",
    "func": "middleware.Logger",
    "msg": "请求完成",
    "method": "GET",
    "path": "/health",
    "handler": "main.main.func1",
    "status": 200,
    "latency": 0.000530,
    "ip": "192.168.1.100",
    "user_agent": "curl/8.7.1",
    "query": "foo=bar&debug=true"
}
```

**POST 请求（中间件只记录元信息，参数由 Controller 层记录）：**

```json
{
    "level": "INFO",
    "ts": "2026-02-28 16:08:41",
    "caller": "middleware/logger.go:108",
    "trace_id": "abc-123-def-456",
    "func": "middleware.Logger",
    "msg": "请求完成",
    "method": "POST",
    "path": "/api/v1/auth/login",
    "handler": "controller.auth_controller.Login",
    "status": 200,
    "latency": 0.025
}
```

**错误响应（4xx/5xx 额外记录响应 Body）：**

```json
{
    "level": "WARN",
    "ts": "2026-02-28 16:08:42",
    "caller": "middleware/logger.go:99",
    "trace_id": "79f3f6fb-6997-4645-9c33-23f832dc6af2",
    "func": "middleware.Logger",
    "msg": "客户端错误",
    "method": "POST",
    "path": "/api/v1/auth/register",
    "handler": "controller.auth_controller.Register",
    "status": 400,
    "resp_body": "{\"code\":400,\"message\":\"邮箱格式不正确\"}"
}
```

**Controller 层日志（caller 精确指向业务代码行号）：**

```json
{
    "level": "INFO",
    "ts": "2026-02-28 16:08:41",
    "caller": "controller/auth_controller.go:35",
    "trace_id": "abc-123-def-456",
    "func": "controller.auth_controller.Login",
    "msg": "开始处理登录",
    "account": "testuser"
}
```

> **设计理念**：中间件层关注"哪个接口被调用、结果如何"，业务层关注"具体做了什么、哪里出错"。两者通过 `trace_id` 串联，既避免日志重复，又能完整追踪请求链路。

### 8.7 WebSocket 日志

WebSocket 连接和消息也纳入日志追踪体系：

```
[INFO] ws_conn=conn-789 | user_id=1 | WebSocket 连接建立
[INFO] ws_conn=conn-789 | user_id=1 | trace_id=ws-001 | 收到事件: im.message.send | conversation_id=5
[INFO] ws_conn=conn-789 | user_id=1 | trace_id=ws-001 | 消息处理完成 | msg_id=10086 | latency=8ms
[WARN] ws_conn=conn-789 | user_id=1 | 心跳超时，准备断开连接
[INFO] ws_conn=conn-789 | user_id=1 | WebSocket 连接断开 | 在线时长=3600s
```

### 8.8 跨服务链路追踪（Go ↔ mediasoup Node）

Go 服务调用 mediasoup Node 服务时，通过 HTTP Header 传递 trace_id：

```
Go 服务:
  [INFO] trace_id=abc-123 | 调用 mediasoup: POST /media/transport/create | room=123-456

mediasoup Node 服务:
  [INFO] trace_id=abc-123 | 收到请求: POST /media/transport/create | room=123-456
  [INFO] trace_id=abc-123 | Transport 创建成功 | transport_id=xxx | latency=5ms

Go 服务:
  [INFO] trace_id=abc-123 | mediasoup 调用完成 | transport_id=xxx | latency=12ms
```

### 8.9 敏感信息脱敏

日志中的敏感信息必须脱敏处理：

| 字段 | 脱敏规则 | 示例 |
|------|---------|------|
| 密码 | 不记录 | `password=***` |
| Token | 只记录前后各 4 位 | `token=eyJh...5NiJ` |
| 邮箱 | 部分隐藏 | `email=zh***@example.com` |
| 手机号 | 中间 4 位隐藏 | `phone=138****8000` |
| IP | 完整记录（用于安全审计） | `ip=192.168.1.100` |

### 8.10 前端日志与错误上报

前端通过 API 上报关键错误和操作日志：

```
POST /api/v1/client/log    # 前端错误上报接口

{
    "level": "error",
    "module": "mediasoup-client",
    "message": "Transport connection failed",
    "stack": "Error: ICE connection failed...",
    "page": "/meeting/room",
    "user_agent": "...",
    "time": "2026-02-27 18:06:40"
}
```

前端需要记录的关键场景：
- WebSocket 连接失败/断线
- mediasoup-client Transport 连接失败
- API 请求超时/5xx 错误
- 页面 JS 异常（通过 `window.onerror` / `Vue.config.errorHandler`）
