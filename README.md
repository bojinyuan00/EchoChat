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

## 项目结构

```
EchoChat/
├── frontend/          # 前台用户端 (uniapp)
├── admin/             # 后台管理端 (Vue 3 + Element Plus)
├── backend/
│   └── go-service/    # Go 后端服务
├── media-server/      # mediasoup Node 媒体服务
├── deploy/            # 部署配置 (Docker / Nginx / K8s)
├── docs/              # 项目文档
│   ├── plans/         # 设计方案与实施计划
│   ├── api/           # API 接口文档
│   └── architecture/  # 架构设计文档
├── scripts/           # 脚本工具
└── README.md
```

---

## 快速开始

### 环境要求

- Go 1.22+
- Node.js 18+
- Docker & Docker Compose
- PostgreSQL 17 (通过 Docker)
- Redis 7 (通过 Docker)

### 1. 启动基础设施

```bash
cd deploy
docker compose -f docker-compose.dev.yml up -d
```

### 2. 启动 Go 后端

```bash
cd backend/go-service
go mod tidy
go run cmd/server/main.go
```

### 3. 启动前台前端

```bash
cd frontend
npm install
npm run dev:h5
```

### 4. 启动管理端

```bash
cd admin
npm install
npm run dev
```

---

## MVP 功能规划（第一期）

- [x] 设计方案与架构文档
- [ ] 用户注册/登录（邮箱+密码、用户名+密码）
- [ ] 即时聊天（单聊 + 群聊，文字/图片/文件）
- [ ] 联系人/好友管理
- [ ] 多人音视频会议（即时会议 + 预约会议）
- [ ] 消息通知系统
- [ ] 后台管理端（用户管理、会议监控、系统配置）

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
| 整体设计方案 | [docs/plans/2026-02-27-echochat-system-design.md](docs/plans/2026-02-27-echochat-system-design.md) | 系统完整设计方案 |
| 第一阶段实施计划 | [docs/plans/2026-02-27-phase1-foundation-and-auth.md](docs/plans/2026-02-27-phase1-foundation-and-auth.md) | 基础设施+用户体系实施步骤 |
| 系统架构文档 | [docs/architecture/system-architecture.md](docs/architecture/system-architecture.md) | 架构分层与技术选型 |
| API 接口文档 | [docs/api/](docs/api/) | 按端+模块拆分的 REST API + WebSocket 事件定义 |

---

## 开源协议

MIT License
