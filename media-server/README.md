# EchoChat media-server

EchoChat 多人音视频会议（Phase 2e-2）的 SFU 节点。内部使用 [mediasoup](https://mediasoup.org/) 进行 WebRTC 媒体转发，外部用 [Fastify](https://fastify.dev/) 暴露内部 REST API，供 Go backend 调度；与 Go backend 之间使用 `X-Internal-Token` 鉴权，**不直接对公网暴露**。

> 配套文档：
> - [Phase 2e-2 设计文档](../docs/plans/2026-04-21-phase2e-2-design.md)
> - [Phase 2e-2 实施计划](../docs/plans/2026-04-21-phase2e-2-implementation.plan.md)
> - [mediasoup PoC Spike 记录](./docs/poc-notes.md)

## 目录结构（Task 1 骨架）

```
media-server/
├── src/
│   ├── app.ts                       # Fastify 入口 + 生命周期
│   ├── config.ts                    # zod 校验的配置加载
│   ├── mediasoup/
│   │   └── worker.ts                # mediasoup Worker 单例 + died 自动重启
│   ├── middlewares/
│   │   └── internal-auth.ts         # X-Internal-Token 鉴权插件
│   └── utils/
│       └── logger.ts                # pino 日志
├── poc/                             # Task 0 PoC 源码（归档，不参与构建）
├── docs/poc-notes.md                # PoC 结论文档
├── Dockerfile                       # 多阶段构建（builder + runtime）
├── .env.example                     # 环境变量模板
├── package.json                     # 锁定 mediasoup@3.19 + fastify@5
├── tsconfig.json / tsconfig.build.json
├── .eslintrc.json / .prettierrc
└── README.md
```

## 快速开始

### 环境要求

- Node.js ≥ 20（Fastify 5 要求；Dockerfile 使用 `node:20-bookworm-slim`）
- 构建 mediasoup 原生 worker 需要：`python3` + `make` + C++ 编译工具链（本机 macOS 已自带 Xcode CLT；Linux 见 Dockerfile）

### 本地开发

```bash
cd media-server
cp .env.example .env
# 按需调整 MEDIA_INTERNAL_TOKEN / MEDIASOUP_ANNOUNCED_IP
npm install
npm run dev
```

启动成功后：

```bash
# 健康检查（公开，无需鉴权）
curl -s http://127.0.0.1:3300/healthz | jq
# {
#   "ok": true,
#   "service": "media-server",
#   "mediasoupVersion": "3.19.x",
#   "workerPid": 12345,
#   ...
# }

# 内部接口（需带 token）
curl -s -H "X-Internal-Token: $MEDIA_INTERNAL_TOKEN" \
  http://127.0.0.1:3300/internal/info | jq
```

### 鉴权校验

未带 token 访问 `/internal/*` 应返回 401：

```bash
curl -i http://127.0.0.1:3300/internal/info
# HTTP/1.1 401 Unauthorized
# {"code":"UNAUTHORIZED","message":"missing or invalid X-Internal-Token"}
```

## 常用脚本

| 命令 | 说明 |
| --- | --- |
| `npm run dev` | tsx watch 热重载开发 |
| `npm run build` | 产出 `dist/` 供生产使用 |
| `npm run typecheck` | TypeScript 纯类型校验 |
| `npm run lint` | ESLint 检查 |
| `npm run format` | Prettier 格式化 |
| `npm run test` | vitest 单测 |
| `npm start` | 运行编译后的 `dist/app.js`（生产） |

## Docker 构建

```bash
docker build -t echochat/media-server:dev -f Dockerfile .
docker run --rm \
  -p 3300:3300 \
  -p 40000-40199:40000-40199/udp \
  -p 40000-40199:40000-40199/tcp \
  -e MEDIA_INTERNAL_TOKEN=dev-internal-token-change-me \
  -e MEDIASOUP_ANNOUNCED_IP=127.0.0.1 \
  echochat/media-server:dev
```

## 关键配置说明

完整列表见 [.env.example](./.env.example)。

| 变量 | 缺省 | 说明 |
| --- | --- | --- |
| `HTTP_PORT` | `3300` | 内部 HTTP 端口，仅供 Go backend 调用 |
| `MEDIA_INTERNAL_TOKEN` | —（必填） | Go 与 Node 之间的共享密钥 |
| `MEDIASOUP_LISTEN_IP` | `0.0.0.0` | Worker 绑定 IP |
| `MEDIASOUP_ANNOUNCED_IP` | 空 | 对外通告 IP（本机 demo 可留空，走默认 LAN；公网必填服务器公网 IP） |
| `MEDIASOUP_RTC_MIN_PORT` / `MAX_PORT` | `40000` / `40199` | RTP UDP/TCP 端口范围 |
| `MEDIASOUP_WORKER_LOG_LEVEL` | `warn` | mediasoup worker 自身日志等级 |
| `MEDIASOUP_MAX_ROUTERS` | `200` | 单 Worker Router 水位上限 |

## Task 1 验收清单

- [x] 目录/配置/lint/格式化齐全
- [x] `/healthz` 返回 `ok=true` + `workerPid`
- [x] `/internal/*` 校验 `X-Internal-Token`，非法请求返回 401
- [x] Dockerfile 多阶段构建，含非 root 用户 + 健康检查
- [x] Worker `died` 事件自动重启（指数退避，最多 30s）

## 后续任务

- **Task 2 ✅**：`/internal/v1/routers`、`/transports`、`/produce`、`/consume` 等 9 个内部 REST API 全部落地（58 单元/集成测试，覆盖率 80.89%）。
- **Task 7 ✅**（2026-04-21）：Go 侧 `HTTPMediaOrchestrator` 通过 `X-Internal-Token` 已完整接入本服务的 `/internal/v1/*`；Go↔Node 媒体链路在端到端脚本 `docs/verify/meeting_t7_verify.mjs` 中 **16/16 PASS**，媒体 Router/Transport/Producer/Consumer 生命周期均由 Go 驱动。
- **Task 9 ✅**（2026-04-21）：前端接入 mediasoup-client 完成，`meeting.consume.resume` WS 事件全链路打通；本服务 `/internal/v1/consumers/:id/resume` 已被 Go 侧使用。
- **Task 14 ✅**（2026-04-24）：纳入 `deploy/docker-compose.dev.yml` 双态部署编排，Dockerfile 镜像由 compose 直接 build；配套 `deploy/.env.local.example`（本机 Demo，`MEDIASOUP_ANNOUNCED_IP=""` 自动走内网）与 `deploy/.env.public.example`（公网部署，强制填写公网 IP + 可选 coturn）。本服务现在通过 `scripts/start.sh full` 以容器身份启动，公网部署走 `scripts/deploy-public.sh`。详见 `docs/deployment/meeting-mvp.md`。
- 后续：Task 10 将新增 `GET /internal/v1/transports/:id/stats` 供 Go 定时拉取。
