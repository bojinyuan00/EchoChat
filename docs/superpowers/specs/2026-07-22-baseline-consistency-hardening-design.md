# EchoChat 基线一致性修复设计

## 1. 背景与目标

EchoChat 的 Phase 2e-2 会议 MVP 已完成，但当前仓库存在三类影响后续持续开发和上线的问题：

1. 当前摘要/API 导航与源码路由不一致，且有两份被导航引用的 API 文档缺失。
2. `start.sh`、`stop.sh`、`status.sh` 对本地混合模式和 `full` 模式的生命周期语义不对称，失败时的状态反馈也不可靠。
3. 公网 `.env` 中的数据库、Redis、MinIO 和运行模式配置没有完整传递到容器内 Go 服务，部署前检查无法证明最终配置有效。

本批次只修复仓库基线，不登录公网服务器，不修改 1Panel，不创建 Cloudflare 记录，也不交付 Caddy/生产前端容器。公网服务器已经运行 1Panel、Docker、数据库和代理软件，后续部署设计必须以共存为前提。

## 2. 事实来源

- REST 路由以 `backend/go-service/app/*/router.go` 为准。
- 会议 WebSocket 事件以 `backend/go-service/app/constants/meeting.go` 和 Handler 注册表为准。
- media-server 路由以 `media-server/src/app.ts`、`src/routes/*.route.ts` 为准。
- 当前状态说明可修正；历史任务报告保留当时事实，不回写成今天的口径。
- Compose 配置以 `docker compose ... config` 的最终渲染结果为准，而不是仅检查 `.env` 文本。

## 3. 修复范围

### 3.1 文档与接口一致性

- 新增 `docs/api/frontend/group.md`，覆盖当前 19 个用户端群聊 REST 接口。
- 新增 `docs/api/admin/group.md`，覆盖当前 3 个管理端群聊 REST 接口。
- 修正 `docs/api/README.md` 中的导航、文件树和接口数量：IM 9、群聊 19、会议 REST 12。
- 将会议 WebSocket 当前口径统一为 16 个唯一事件：9 个 C→S、8 个 S→C，其中 `meeting.member.state.changed` 双向复用。
- 将 media-server 当前口径统一为 `/internal/v1` 下 10 个资源生命周期接口，另有 1 个鉴权的 `/internal/info` 和 2 个公开健康接口。
- 保留 Task 2 历史“9 个接口”的记录，但追加当前状态，明确 Transport Stats 尚未实现。
- 增加 Markdown 本地链接检查，阻止导航再次指向不存在的文件。

### 3.2 启停与状态脚本

统一模式语义：

- `start.sh`：Docker 中间件 + 本地 Go/media/frontend/admin。
- `start.sh full`：Compose 后端栈 + 本地 frontend/admin。
- `stop.sh`：停止本地应用进程，保留容器。
- `stop.sh full`：停止 `full` 模式产生的 Compose 服务和本地 frontend/admin。
- `stop.sh --all`：停止所有 EchoChat 本地进程与全部 Compose 服务，保留数据卷。

启动流程不再以固定 1 秒存活作为成功依据。Go、media、frontend、admin 分别执行条件式健康等待；所有结果汇总后，任一必需服务失败则脚本返回非零，且不打印误导性的“启动完成”。

`status.sh` 区分 PID、监听端口、HTTP 健康和 Docker health，标记 stale PID，不把“进程存在”直接等同于“服务可用”。

### 3.3 公网配置传递

- 将 PostgreSQL 用户、密码、数据库名完整注入 Go 容器。
- 将 Redis 密码同时注入 Redis 服务、Redis healthcheck 和 Go 容器；本地空密码仍可运行。
- 将 MinIO Access Key、Secret Key、Bucket 注入 Go 容器。
- 增加 `GO_SERVER_MODE` 和 `WS_ALLOWED_ORIGINS`，公网模板分别使用 `release` 和明确 Origin。
- `deploy-public.sh` 使用最终 Compose 配置做预检；检测开发默认密钥、缺失密码、错误运行模式或 Origin 缺失时拒绝部署。
- 本批次不改变公网端口拓扑；端口收敛、反向代理和前端生产镜像进入下一份部署设计。

## 4. 测试策略

遵循测试先行：

1. 先建立 Shell 行为测试，复现 `full`/`--all` 停止不对称和失败后仍报告成功。
2. 建立文档链接与当前接口计数检查，先观察现有仓库失败。
3. 建立 Compose 最终配置断言，先证明公网密码/模式传递断链。
4. 实施最小修复，逐项转绿。
5. 最终运行 Shell 测试、`bash -n`、文档检查、Compose 渲染检查、Go test/vet/build、media-server test/build、前端 H5 build 和管理端 build。

测试不得读取 `node_modules` 内容；只允许通过现有包执行构建和测试。

## 5. 错误处理与兼容性

- 脚本收集所有服务错误后统一返回非零，避免首个失败掩盖后续状态。
- 健康等待使用超时和条件轮询，不使用长时间盲等。
- `stop` 操作保持幂等；未运行的服务只给提示，不视为失败。
- 不删除 Docker volume，不停止非 EchoChat 容器，不触碰 1Panel 管理的其他服务。
- 保持原有命令名称兼容，文档同步更新真实语义。

## 6. 非目标

- 不部署或改造 1Panel。
- 不登录公网服务器。
- 不配置 Cloudflare、域名、证书或防火墙。
- 不新增生产 frontend/admin/Caddy 容器。
- 不重构会议业务、IM 业务或大型 Pinia Store。
- 不合并当前功能分支到 `main`。

## 7. 完成标准

- API 导航无断链，当前接口数量与源码一致。
- 生命周期命令在本地混合模式和 `full` 模式下对称、幂等、失败可见。
- 公网示例配置能够渲染出无开发默认凭据的 Go/Redis/PostgreSQL/MinIO 配置。
- 新增检查可在后续改动中重复执行。
- 原有 Go、media-server、frontend、admin 验证全部通过，工作区只包含本批次文件。
