# EchoChat - 音视频会议直播系统

EchoChat 是一套跨端可用、可扩展、可演进的实时音视频会议直播系统。支持即时聊天、多人音视频会议、互动直播等核心功能，采用控制面与媒体面彻底分离的架构设计。

---

## 技术栈

| 层级 | 技术 | 说明 |
|------|------|------|
| 前台前端 | uniapp (Vue 3) + mediasoup-client | 多端适配（H5/App/小程序） |
| 后台管理端 | Vue 3 + Vite + Element Plus | PC Web 管理后台 |
| 后端服务 | Go (Gin + GORM + Wire + zap) | 业务逻辑、信令控制 |
| 媒体服务 | Node.js + mediasoup | SFU 媒体控制与转发 |
| 数据库 | PostgreSQL 17 | 持久化数据存储 |
| 缓存 | Redis 7 | 实时状态、会话缓存 |
| 部署 | Docker Compose / Nginx | 容器化部署，预留 K8s |

---

## 系统架构

```
客户端 (uniapp / Vue3 管理端)
    │
    │ WebSocket + HTTP
    │
Go 单体服务 (模块化)
    │ ├── auth      用户认证鉴权
    │ ├── im        即时通讯
    │ ├── contact   联系人管理
    │ ├── meeting   会议控制/信令
    │ ├── notify    消息通知
    │ └── admin     后台管理
    │
    ├── PostgreSQL  (持久化数据)
    ├── Redis       (实时状态)
    │
    │ HTTP
    │
mediasoup Node 服务
    │ IPC
mediasoup Worker (C++ SFU)
```

核心设计思想：**控制面与媒体面分离**。Go 处理所有业务逻辑和信令控制，mediasoup 专注音视频媒体转发，音视频流直连 SFU Worker，不经过 Go 服务。

---

## 项目结构：

```
EchoChat/
├── frontend/              # 前台用户端 (uni-app + Vue 3.4)
├── admin/                 # 后台管理端 (Vue 3.5 + Element Plus)
├── backend/
│   └── go-service/        # Go 后端服务 (Gin + GORM + Wire)
│       ├── app/           # 业务模块 (auth / admin)
│       ├── cmd/server/    # 服务入口
│       ├── config/        # 配置文件
│       ├── pkg/           # 公共包 (db / logs / middleware / utils)
│       └── router/        # 路由聚合
├── media-server/          # mediasoup Node 媒体服务 (Phase 3)
├── deploy/                # 部署配置 (Docker Compose)
├── design-system/         # UI 设计系统 (ui-ux-pro-max 生成)
├── docs/                  # 项目文档
│   ├── progress/          # 开发进度
│   ├── plans/             # 实施计划
│   ├── api/               # API 接口文档
│   └── architecture/      # 架构设计文档
└── README.md
```

---

## 快速开始

### 环境要求

- Go 1.23+
- Node.js 18+
- Docker & Docker Compose
- PostgreSQL 17（通过 Docker 自动启动）
- Redis 7（通过 Docker 自动启动）

### 方式一：Docker Compose 一键启动（推荐）

```bash
cd deploy
docker compose -f docker-compose.dev.yml up -d
```

启动后服务地址：
- Go 后端 API：`http://localhost:8085`
- PostgreSQL：`localhost:5432`
- Redis：`localhost:6379`

### 方式二：分步手动启动（开发调试）

**1. 启动基础设施（数据库 + 缓存）**

```bash
cd deploy
docker compose -f docker-compose.dev.yml up -d postgres redis
```

**2. 启动 Go 后端**

```bash
cd backend/go-service
go mod tidy
go run cmd/server/main.go
```

后端启动在 `http://localhost:8085`，健康检查：`GET /health`

**3. 启动前台用户端（uni-app H5）**

```bash
cd frontend
npm install --legacy-peer-deps
npm run dev:h5
```

前台 H5 启动在 `http://localhost:5173`（端口可能递增）

**4. 启动后台管理端（Vue 3）**

```bash
cd admin
npm install
npm run dev
```

管理端启动在 `http://localhost:3100`（自动代理 `/api` 到后端 8085）

---

## MVP 功能规划（第一期）

### Phase 1 — 基础设施与用户认证 ✅

- [x] 设计方案与架构文档
- [x] Docker Compose 开发环境（PostgreSQL + Redis）
- [x] Go 后端服务骨架（Gin + GORM + Wire + Zap）
- [x] 用户注册/登录 API（用户名+邮箱+密码）
- [x] JWT Token 认证（有状态 JWT + Redis）
- [x] RBAC 角色权限（user / admin / super_admin）
- [x] 前台 uni-app 骨架 + 登录/注册/TabBar/个人中心
- [x] 后台 Vue 3 管理端 + 登录/布局/用户管理
- [x] Go 服务 Dockerfile + Docker Compose 全栈部署

### Phase 2 — 即时通讯（待开始）

- [ ] 即时聊天（单聊 + 群聊，文字/图片/文件）
- [ ] 联系人/好友管理
- [ ] 消息通知系统

### Phase 3 — 音视频会议（待开始）

- [ ] 多人音视频会议（即时会议 + 预约会议）
- [ ] 后台管理端会议监控

## 后续规划

- 屏幕共享
- 微信授权登录
- 互动直播（主播/观众/弹幕）
- 会议录制与回放
- 微服务拆分 + K8s 部署
- AI 辅助功能（语音转文字、会议纪要）

---

## 文档导航

| 文档 | 路径 | 说明 |
|------|------|------|
| 项目进度 | [docs/progress/CURRENT_STATUS.md](docs/progress/CURRENT_STATUS.md) | 当前开发进度与技术决策 |
| 整体设计方案 | [docs/plans/2026-02-27-echochat-system-design.md](docs/plans/2026-02-27-echochat-system-design.md) | 系统完整设计方案 |
| 第一阶段实施计划 | [docs/plans/2026-02-27-phase1-foundation-and-auth.md](docs/plans/2026-02-27-phase1-foundation-and-auth.md) | 基础设施+用户体系实施步骤 |
| 系统架构文档 | [docs/architecture/system-architecture.md](docs/architecture/system-architecture.md) | 架构分层与技术选型 |
| API 接口文档 | [docs/api/](docs/api/) | 按端+模块拆分的 REST API + WebSocket 事件定义 |

---

## 开源协议

MIT License
