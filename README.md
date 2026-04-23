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
├── media-server/          # mediasoup Node 媒体服务 (Phase 2e-2 Task 0-2 已落地 9 REST API；Task 7 起 Go 通过 HTTPMediaOrchestrator 接入；Task 14 纳入 docker-compose 双态编排)
├── deploy/                # 部署配置 (Docker Compose + 三份 .env 双态模板)
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

### 方式一：一键脚本（推荐）

项目在 `scripts/` 目录遵循「**首次初始化 / 日常启停**」职责分离模式，参考 Rails / Django / Next.js 社区通用做法：

| 脚本 | 用途 | 使用频率 |
|---|---|---|
| `scripts/dev-setup.sh` | **首次环境初始化**：检查 Docker、拉起 Postgres/Redis/MinIO、重试式健康检查 | 仅首次 clone 或重建卷时 |
| `scripts/start.sh` | **日常启动**：秒级拉起全部服务（容器 + 应用层）；`full` 子命令拉起含 media-server 的全量栈 | 每天多次 |
| `scripts/stop.sh` | **日常停止**：优雅终止应用层，默认保留容器；`full` 子命令停止全量栈 | 每天多次 |
| `scripts/status.sh` | **状态查看**：端口 / PID / 应用容器（含 media-server / coturn）一览 | 排障随时 |
| `scripts/deploy-public.sh` | **公网部署**（Phase 2e-2 Task 14）：env 校验 + 端口 checklist + Docker 自检 + `--profile public` 启动全栈 + media-server 健康检查 | 公网部署时 |

#### 首次使用（仅一次）

```bash
# 首次 clone 后执行一次，完成 Docker 中间件初始化 + 健康检查
./scripts/dev-setup.sh
```

#### 日常启停

```bash
# 启动全部服务
./scripts/start.sh

# 查看服务状态（端口/PID/容器）
./scripts/status.sh

# 停止应用层（保留 Docker 中间件）
./scripts/stop.sh

# 停止全部（含 Docker 中间件）
./scripts/stop.sh --all

# 只启停单项（可选 docker|backend|frontend|admin）
./scripts/start.sh backend
./scripts/stop.sh frontend
```

后台进程的 PID 与日志默认写入 `.run/`（已加入 gitignore），排障时可直接查看 `.run/logs/*.log`。

启动后各服务地址：

| 服务 | 端口 | 说明 |
|---|---|---|
| 前台用户端 (H5) | 5173 | `http://localhost:5173` |
| 后台管理端 | 3100 | `http://localhost:3100` |
| Go 后端 API | 8085 | `http://localhost:8085`（`/health` 健康检查） |
| PostgreSQL | 5432 | Docker 容器 `echochat-postgres` |
| Redis | 6379 | Docker 容器 `echochat-redis` |
| MinIO API | 9000 | 对象存储，S3 兼容 |
| MinIO Console | 9001 | `http://localhost:9001`（echochat / echochat123456） |

### 方式二：Docker Compose 全量启动（含 media-server）

```bash
cd deploy
cp .env.local.example .env   # 本机 Demo 模板
cd ..
./scripts/start.sh full      # 拉起 postgres + redis + minio + go-service + media-server 容器
```

公网部署请参考 `docs/deployment/meeting-mvp.md`：`cp deploy/.env.public.example deploy/.env` → 替换占位符 → `./scripts/deploy-public.sh`，脚本会自动校验 `MEDIASOUP_ANNOUNCED_IP` / 端口 / Docker 环境，并按需拉起 coturn（`--profile public`）。

### 方式三：分步手动启动（开发调试）

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

## 开发进度

> 最新进度、里程碑与技术决策以 [docs/progress/CURRENT_STATUS.md](docs/progress/CURRENT_STATUS.md) 为准（本节为概览）。

### Phase 1 — 基础设施与用户认证 ✅（2026-03 完成）

- [x] 设计方案与架构文档 · Docker Compose（PostgreSQL + Redis + MinIO）
- [x] Go 后端骨架（Gin + GORM + Wire + Zap）+ JWT + RBAC
- [x] 前台 uni-app 骨架 + 后台 Vue 3 管理端 + Docker 化

### Phase 2a / 2b / 2c / 2d — 即时通讯全量 ✅（2026-03 完成）

- [x] 联系人 / 好友管理（2a）
- [x] WebSocket 长连接 + 单聊即时通讯（2b）
- [x] 群聊 + 已读回执（2c）
- [x] 消息类型扩展（图片 / 文件 / 语音，含 MinIO S3 存储）（2d）

### Phase 2e-1 — 统一通知中心 ✅（2026-04-20 完成）

- [x] 通知数据模型 + `notify.Pusher` 统一推送抽象
- [x] 前端通知中心页 + 卡片渲染 + WS 实时推送
- [x] `meeting_invite` 预留类型（供 Phase 2e-2 联动）

### Phase 2e-2 — 多人音视频会议 MVP ✅（2026-04-24 完成）

- [x] 技术栈：mediasoup Node 媒体控制层 + C++ SFU Worker + mediasoup-client（WebRTC）
- [x] 控制面：Go 会议 service + 9 REST API + 14 WS 事件（创建 / 加入 / 离开 / 主持人四件套 / 邀请 / 聊天 / 生命周期）
- [x] 媒体面：Node 媒体服务 12 内部 REST（Router / Transport / Producer / Consumer / Stats）+ Token 内部鉴权
- [x] 前端会议闭环：Hub → 预览 → 房间（网格 / 演讲者 / 工具栏 / 聊天 / 成员列表 / 主持人菜单）
- [x] 会议生命周期：host 掉线 2 分钟宽限 + 自动转让 + 空房 5 分钟 TTL + 定时兜底清理
- [x] 双态部署：本机 Docker Compose 即装即用 + 公网（announcedIp + coturn `--profile public`）
- [x] 原创 UI 特色 6 条（桌面端恒浮窗 / 加入时视频块滑入等）+ 主持人权限四件套（静音他人 / 移除 / 转让 / 结束）
- [x] 代码审查闭环：4 P0 + 8 P1 + 7 P2 + 7 Nit 全部修复，5 项（WS token 迁出 URL / Chat 服务拆分等）登记推迟到 Phase 2f/3
- [x] 验收产物：[test-report-phase2e-2-meeting.md](test-report-phase2e-2-meeting.md)（八章完整报告）+ 5 份 Playwright MCP E2E 剧本

### Phase 2e-3 / Phase 3 — 待启动（规划中）

- [ ] 预约会议 + 会议提醒 + 等候室 + 预览高级参数（2e-3）
- [ ] 屏幕共享 + 会议录制与回放（3）
- [ ] 互动直播（主播 / 观众 / 弹幕）（3）
- [ ] WS token 从 URL 迁出 + MeetingChatService 拆分（P2 推迟项）

## 后续规划

- 微信授权登录
- 微服务拆分 + K8s 部署
- AI 辅助功能（语音转文字、会议纪要）

---

## 文档导航

| 文档 | 路径 | 说明 |
|------|------|------|
| 项目进度 | [docs/progress/CURRENT_STATUS.md](docs/progress/CURRENT_STATUS.md) | 当前开发进度与技术决策（SSOT） |
| 整体设计方案 | [docs/plans/2026-02-27-echochat-system-design.md](docs/plans/2026-02-27-echochat-system-design.md) | 系统完整设计方案 |
| Phase 2e 路线图 | [docs/plans/2026-04-20-phase2e-design.md](docs/plans/2026-04-20-phase2e-design.md) | Phase 2e-1 / 2e-2 / 2e-3 三子阶段总览 |
| Phase 2e-2 设计 | [docs/plans/2026-04-21-phase2e-2-design.md](docs/plans/2026-04-21-phase2e-2-design.md) | 会议 MVP 专项设计（✅ 已完成） |
| Phase 2e-2 实施 | [docs/plans/2026-04-21-phase2e-2-implementation.plan.md](docs/plans/2026-04-21-phase2e-2-implementation.plan.md) | Task 0-16 实施计划 |
| Phase 2e-2 代码审查 | [docs/reviews/2026-04-23-phase2e-2-code-review.md](docs/reviews/2026-04-23-phase2e-2-code-review.md) | 26 项审计结果 + Task 16 修复追踪 |
| Phase 2e-2 验收报告 | [test-report-phase2e-2-meeting.md](test-report-phase2e-2-meeting.md) | 八章完整测试报告 |
| 会议 MVP 部署指南 | [docs/deployment/meeting-mvp.md](docs/deployment/meeting-mvp.md) | 本机 Demo + 公网双态部署手册 |
| 系统架构文档 | [docs/architecture/system-architecture.md](docs/architecture/system-architecture.md) | 架构分层与技术选型 |
| API 接口文档 | [docs/api/](docs/api/) | 按端+模块拆分的 REST API + WebSocket 事件定义 |

---

## 开源协议

MIT License
