# EchoChat 启动与部署手册

> **适用阶段**：Phase 2e-2 会议 MVP 已完成
> **覆盖形态**：本机混合开发 / 本机容器化演示 / 生产服务器公网部署
> **最后更新**：2026-04-24

---

## 一、部署形态总览

EchoChat 支持三种启动形态，**同一套代码、同一份 docker-compose 定义**，只靠 `deploy/.env` 与启动脚本参数切换：

| 形态 | 启动方式 | 中间件（PG/Redis/MinIO） | 应用层（Go/Node/前端/Admin） | TURN | 典型场景 |
|---|---|---|---|---|---|
| **1. 本机混合（推荐开发用）** | `./scripts/start.sh` | Docker | 本地进程（nohup + 热更） | 关 | 日常开发、边改边调 |
| **2. 本机全容器** | `./scripts/start.sh full` | Docker | Docker（Go + Node 容器）+ 本地 Vite | 关 | 接近生产形态的自测 |
| **3. 公网生产** | `./scripts/deploy-public.sh` | Docker | Docker | **开** | 上线 / 跨公网用户 |

### 该选哪种？

```
你是谁？
├── 前端/后端/信令开发者 → 形态 1（改完立刻热更）
├── 要验证容器化 / 离上线还差一公里 → 形态 2
└── 要部署到云服务器供真实用户使用 → 形态 3
```

**注意**：形态 2 在使用 Clash / Surge / v2ray 等 **TUN 模式代理** 的开发机上容器内 DNS 会被 fake-ip 段（`198.18.0.0/16`）劫持，导致 apt/apk 拉不到包。如果在本机复现，请先关掉代理 TUN 模式，或直接跳过形态 2，去云服务器验形态 3。

---

## 二、形态 1：本机混合模式（开发常用）

### 2.1 前置依赖

| 工具 | 版本 | 作用 |
|---|---|---|
| Docker Desktop | 4.x+（含 Compose V2） | 起中间件 |
| Node.js | **24.x**（.nvmrc 已锁定） | 前端 / Admin / media-server |
| Go | **1.22+** | go-service |
| `nvm`（可选） | 任意 | `nvm use` 自动对齐 Node 版本 |
| 浏览器 | Chrome / Edge 最新版 | WebRTC 测试 |

首次准备：

```bash
# 1. 按 .nvmrc 使用 Node 24（若装了 nvm）
cd frontend && nvm use && cd ../admin && nvm use && cd ../media-server && nvm use && cd ..

# 2. 安装 node 依赖（任选其一）
npm --prefix frontend install
npm --prefix admin install
npm --prefix media-server install

# 3. Go 依赖自动在 go run 时下载，无需手动 go mod download
```

### 2.2 一键启动

```bash
./scripts/start.sh
```

这条命令等价于：

1. `docker compose -f deploy/docker-compose.dev.yml up -d postgres redis minio` —— 只起 3 个中间件容器
2. `nohup go run backend/go-service/cmd/server/main.go` —— Go 后端本地进程（端口 8085）
3. `nohup npm run dev -- --prefix media-server` —— Node mediasoup 本地进程（端口 3300）
4. `nohup npm run dev:h5 -- --prefix frontend` —— Vite 前端（端口 5173，HTTPS 自签）
5. `nohup npm run dev -- --prefix admin` —— Vite 管理端（端口 3100）

首次运行会自动从 `.env.local.example` 复制 `deploy/.env`，无需手动操作。

启动完成后访问：

| 地址 | 用途 |
|---|---|
| https://localhost:5173 | 前台用户端 H5（**HTTPS 强制**，首次访问需信任自签证书） |
| http://localhost:3100 | 后台管理端 |
| http://localhost:8085/health | Go 后端健康检查 |
| http://localhost:3300/healthz | media-server 健康检查 |
| http://localhost:9001 | MinIO 控制台（`echochat` / `echochat123456`） |

> **为什么前端是 HTTPS？** 浏览器 `getUserMedia`（摄像头 / 麦克风）强制要求 Secure Context。`frontend/vite.config.js` 已启用 `@vitejs/plugin-basic-ssl` 自动签发自签证书。后端接口（`/api`）通过 Vite 代理走 HTTP（同源策略成立），无需额外配置。

### 2.3 单服务启动（调试场景）

| 只启动 | 命令 |
|---|---|
| 中间件 | `./scripts/start.sh docker` |
| Go 后端 | `./scripts/start.sh backend` |
| media-server | `./scripts/start.sh media` |
| 前端 | `./scripts/start.sh frontend` |
| 管理端 | `./scripts/start.sh admin` |
| 应用层全部（假设容器已在运行） | `./scripts/start.sh --no-docker` |

> 脚本会检查端口占用，已被占用则跳过重复启动。任一实际启动步骤失败时，其余步骤仍会尝试执行，但脚本最终以非 0 退出且不会输出虚假的“启动完成”。

### 2.4 前台调试（直接看日志）

脚本默认用 `nohup` 后台跑，日志写到 `.run/logs/`。若要前台起方便直接看输出：

```bash
# 先停掉后台进程
./scripts/stop.sh

# 逐个前台运行
cd backend/go-service && go run cmd/server/main.go
cd media-server && npm run dev
cd frontend && npm run dev:h5
cd admin && npm run dev
```

### 2.5 停止

```bash
./scripts/stop.sh          # 停应用层，保留容器（下次 start 很快）
./scripts/stop.sh full     # 停本地前端/管理端 + docker compose 全栈（匹配 start full）
./scripts/stop.sh --all    # 停全部本地进程 + compose 全栈（数据卷保留）
```

### 2.6 常用查看命令

```bash
./scripts/status.sh                              # 所有服务状态
tail -f .run/logs/backend.log                    # Go 后端日志
tail -f .run/logs/media.log                      # mediasoup 日志
docker logs -f echochat-postgres                 # 查 Postgres
docker logs -f echochat-redis                    # 查 Redis
docker logs -f echochat-minio                    # 查 MinIO

# 直连数据库
docker exec -it echochat-postgres psql -U echochat -d echochat
docker exec -it echochat-redis redis-cli
```

---

## 三、形态 2：本机全容器化

### 3.1 使用场景

- 验证 Dockerfile 没问题（PR 合入前的最后一公里）
- 不想在本机装 Go / Node，一键跑
- 模拟生产的启动顺序、健康检查、依赖关系

### 3.2 前置依赖

- Docker Desktop（Resources → 建议分配 ≥ 4 CPU / ≥ 6GB RAM）
- 至少 8GB 可用磁盘（首次构建会下 2 个基础镜像共 ~300MB）

### 3.3 一键启动

```bash
./scripts/start.sh full
```

这条命令做：

1. `docker compose -f deploy/docker-compose.dev.yml up -d --build`
   构建并启动 **5 个容器**：postgres / redis / minio / **go-service** / **media-server**
2. 等待 media-server healthcheck 变绿（最多 180 秒）
3. 在本地**仍然**起 Vite 前端 + 管理端（热更体验更好）

### 3.4 形态 1 vs 形态 2 的关键差异

| 项 | 形态 1 | 形态 2 |
|---|---|---|
| Go backend | 本机 `go run` | 容器 `echochat-go-service` |
| media-server | 本机 `npm run dev` | 容器 `echochat-media-server` |
| 改一行 Go 代码 | 自动 rebuild | 需要 `docker compose build go-service` |
| 看日志 | `.run/logs/backend.log` | `docker logs echochat-go-service` |
| 首次启动时间 | 30 秒 | 3~8 分钟（首次构建镜像） |

### 3.5 国内网络构建加速（可选）

国内机器首次构建可能因为 `deb.debian.org` / `dl-cdn.alpinelinux.org` 走国际出口而慢。在 `deploy/.env` 里取消注释这 3 行即可走阿里云镜像：

```bash
APT_MIRROR=mirrors.aliyun.com
APK_MIRROR=mirrors.aliyun.com
GO_BUILD_PROXY=https://goproxy.cn,direct
```

这些是 **Linux 发行版镜像**，不是"Docker Hub 加速"，是两个完全独立的服务。

### 3.6 停止

```bash
./scripts/stop.sh full
```

这会同时停止 `start.sh full` 拉起的本地 Vite 前端、管理端和全部 Compose 服务；只执行 `stop`，不会删除 volume。

---

## 四、形态 3：生产服务器公网部署

本节默认服务器上已经有可用的 Docker Compose（例如由现有 1Panel 环境提供）。EchoChat 部署脚本不会安装、升级或替换 1Panel，也不会修改 1Panel 中已有的其他应用。

### 4.1 前置条件

| 资源 | 要求 |
|---|---|
| 服务器 | 有**公网 IP** 或已解析 A 记录的域名（PaaS-only 如 Vercel / Netlify 不适用） |
| 带宽 | 单会议 ~400Kbps × 用户数（25 人会议约 10Mbps） |
| 操作系统 | Ubuntu 22.04+ / Debian 11+ / CentOS 7+ 均可 |
| Docker | 最新稳定版 + Compose V2 |
| 防火墙 | 能自己控制云安全组 / iptables（见 4.4） |
| 域名+HTTPS | 强烈推荐（WebRTC 在 Secure Context 外无法访问摄像头） |

### 4.2 安装 Docker（如服务器尚未安装）

```bash
curl -fsSL https://get.docker.com | bash
sudo systemctl enable --now docker
sudo usermod -aG docker $USER    # 登出再登入生效
docker --version && docker compose version
```

### 4.3 拉取代码 + 准备 .env

```bash
# 约定目录
sudo mkdir -p /opt/echochat && sudo chown $USER /opt/echochat
git clone <your-repo-url> /opt/echochat
cd /opt/echochat

# 从公网模板复制 env
cp deploy/.env.public.example deploy/.env
vim deploy/.env
```

**必改字段**（`deploy/.env`）：

| 字段 | 示例 | 说明 |
|---|---|---|
| `DEPLOY_MODE` | `public` | 已预填，勿改 |
| `POSTGRES_PASSWORD` | `openssl rand -base64 24` | Postgres 强密码 |
| `REDIS_PASSWORD` | `openssl rand -base64 24` | Redis 强密码；Compose 自动同时注入服务端 `requirepass`、健康检查和 Go 客户端 |
| `MINIO_ROOT_PASSWORD` | 同上 | MinIO 控制台密码 |
| `GO_SERVER_MODE` | `release` | 公网必须为 release |
| `WS_ALLOWED_ORIGINS` | `https://chat.example.com` | 允许建立 WebSocket 的 HTTPS 前端 Origin；多个值用逗号分隔 |
| `JWT_SECRET` | `openssl rand -hex 32` | 64 字符随机串 |
| `MEDIA_INTERNAL_TOKEN` | `openssl rand -hex 32` | Go ↔ Node 内部鉴权 |
| `MEDIASOUP_ANNOUNCED_IP` | `203.0.113.42` | **服务器公网 IP**（写内网 IP 或 127.0.0.1 会导致远端建不连） |
| `TURN_USERNAME` / `TURN_PASSWORD` | 自设 | 不改会被互联网匿名中继滥用 |

**可选（国内服务器推荐打开）**：

```bash
APT_MIRROR=mirrors.aliyun.com
APK_MIRROR=mirrors.aliyun.com
GO_BUILD_PROXY=https://goproxy.cn,direct
```

### 4.4 开放云安全组 / 防火墙端口

| 协议 | 端口 | 用途 | 备注 |
|---|---|---|---|
| TCP | 443 | 前端 HTTPS 入口 | 建议反代到 5173 |
| TCP | 8085 | Go API | 生产可只对内网开 |
| UDP | 40000-40199 | mediasoup 媒体流 | **必开**，否则视频黑屏 |
| TCP | 40000-40199 | ICE-TCP fallback | 防火墙穿透场景 |
| UDP | 3478 | coturn STUN/TURN | 必开 |
| TCP | 3478, 5349 | coturn / TLS | 建议开 |
| UDP | 49160-49200 | coturn 中继 | 必开 |

**iptables 示例**（CentOS / Ubuntu 均通用）：

```bash
sudo iptables -I INPUT -p tcp --dport 443 -j ACCEPT
sudo iptables -I INPUT -p tcp --dport 8085 -j ACCEPT
sudo iptables -I INPUT -p udp --dport 40000:40199 -j ACCEPT
sudo iptables -I INPUT -p tcp --dport 40000:40199 -j ACCEPT
sudo iptables -I INPUT -p udp --dport 3478 -j ACCEPT
sudo iptables -I INPUT -p tcp --dport 3478 -j ACCEPT
sudo iptables -I INPUT -p tcp --dport 5349 -j ACCEPT
sudo iptables -I INPUT -p udp --dport 49160:49200 -j ACCEPT
sudo iptables-save | sudo tee /etc/iptables/rules.v4    # 持久化
```

**阿里云 / 腾讯云 / AWS** 在云控制台安全组里照搬上表。

### 4.5 启动

```bash
./scripts/deploy-public.sh
```

脚本会按 5 步依次执行：

1. **校验 .env**：占位符、强密码、`ANNOUNCED_IP`、`GO_SERVER_MODE=release` 和 HTTPS Origin
2. **本机端口探测**：防止已有其它服务占用 8085 / 3300 / 3478
3. **现有 Docker 环境**：daemon 可访问、Compose V2 存在；不安装或替换 1Panel/Docker
4. **只读预检**：渲染最终 Compose 配置，失败时不创建或修改容器
5. **启动**：`docker compose --profile public up -d --build`（包含 coturn 容器）

> 当前公网脚本不构建或启动前端/管理端生产容器，也不配置域名与反向代理；这些将在后续公网部署阶段与现有 1Panel 反代共存设计后完成。

### 4.6 反向代理 + HTTPS（强烈建议）

用 Caddy（最省事、自动申请并续期 Let's Encrypt 证书）：

```bash
# /etc/caddy/Caddyfile
your-domain.com {
    # 前端 H5
    reverse_proxy /* localhost:5173
    # 后端 API
    reverse_proxy /api/* localhost:8085
    # WebSocket 信令
    reverse_proxy /ws/* localhost:8085 {
        header_up Upgrade {http.request.header.upgrade}
        header_up Connection "upgrade"
    }
}

admin.your-domain.com {
    reverse_proxy /* localhost:3100
    reverse_proxy /api/* localhost:8085
}
```

```bash
sudo systemctl enable --now caddy
```

若用 Nginx，对照着改 `proxy_pass` + `proxy_set_header Upgrade / Connection` 即可。

### 4.7 运维常用命令

```bash
# 查看状态
./scripts/status.sh
docker compose -f deploy/docker-compose.dev.yml ps

# 查看各服务日志
docker logs -f echochat-go-service
docker logs -f echochat-media-server
docker logs -f echochat-coturn    # 仅 public profile

# 重启单个服务（代码热修后）
docker compose -f deploy/docker-compose.dev.yml up -d --build go-service

# 停止
./scripts/stop.sh full
# 或：docker compose -f deploy/docker-compose.dev.yml --profile public down

# 数据库备份
docker exec echochat-postgres pg_dump -U echochat echochat \
  | gzip > /backup/echochat-$(date +%F).sql.gz

# 数据库恢复
gunzip -c /backup/echochat-2026-04-24.sql.gz \
  | docker exec -i echochat-postgres psql -U echochat -d echochat
```

### 4.8 生产加固建议

1. **关闭中间件公网端口**：Postgres `5432` / Redis `6379` / MinIO `9000,9001` 只对内网或 127.0.0.1 开放；docker-compose 里的 `ports` 改成 `127.0.0.1:5432:5432` 即可。
2. **日志集中**：`docker compose` 默认 json-file driver 占磁盘，生产接入 Loki / ELK / 阿里 SLS。
3. **数据卷备份**：`pgdata` / `redisdata` / `minio_data` 这 3 个 volume 定期快照。
4. **限制入站 CORS**：`backend/go-service/config/config.docker.yaml` → `server.ws_allowed_origins` 只放行你的正式域名（Task 16 P2 修复新增字段）。

---

## 五、配置文件速查表

下面是所有会影响启动的配置文件，按使用频率排序。

### 5.1 部署层

| 文件 | 用途 | 何时改 |
|---|---|---|
| `deploy/.env` | 运行时环境变量（docker-compose 读取），由 `.env.local.example` 或 `.env.public.example` 拷贝而来 | 每次换部署形态 |
| `deploy/.env.local.example` | 本机模板 | 偶尔：改默认值或加新字段 |
| `deploy/.env.public.example` | 公网模板 | 偶尔：改默认值或加新字段 |
| `deploy/docker-compose.dev.yml` | 5+1 个服务的容器定义 | 加服务 / 改端口映射 |
| `deploy/docker/postgres/init.sql` | Postgres 初始 schema（首次启动执行） | 改初始数据 |
| `deploy/docker/redis/redis.conf` | Redis 通用配置 | 通常无需改；密码由 Compose 根据 `REDIS_PASSWORD` 自动启用 |

### 5.2 Go 后端

| 文件 | 用途 | 何时改 |
|---|---|---|
| `backend/go-service/config/config.dev.yaml` | 本机开发配置（DB 指向 `localhost:5432` 等） | 本机改端口 / 加数据库字段 |
| `backend/go-service/config/config.docker.yaml` | 容器内配置（DB 指向 `postgres:5432` 等） | 形态 2/3 改配置 |
| `backend/go-service/config/config.go` | 配置结构体定义（Go 代码） | 加新配置字段时必改 |
| `backend/go-service/Dockerfile` | 容器构建 | 改基础镜像 / 构建参数 |

**关键字段速查**（以 `config.docker.yaml` 为例）：

```yaml
server:
  port: 8085                    # API 监听端口
  ws_allowed_origins:           # CORS / WebSocket Origin 白名单（Task 16 P2）
    - "http://localhost:5173"
    - "https://localhost:5173"

jwt:
  secret: "${ECHOCHAT_JWT_SECRET}"   # 从 env 注入
  expire_hours: 72

database:
  host: postgres                # 容器网络里的 service name
  user: echochat
  password: "${POSTGRES_PASSWORD}"

redis:
  addr: "redis:6379"
  password: "${REDIS_PASSWORD}"

media_server:
  base_url: "http://media-server:3300"
  internal_token: "${ECHOCHAT_MEDIA_SERVER_INTERNAL_TOKEN}"
  timeout_ms: 10000             # Task 16 P1 增大默认值
  create_router_retry: 1        # Task 16 P1 新增
```

### 5.3 media-server（Node / mediasoup）

| 文件 | 用途 | 何时改 |
|---|---|---|
| `media-server/.env` | 运行时配置（端口、token、announcedIp） | 首次启动自动从 `.env.example` 拷贝；公网部署必改 announcedIp |
| `media-server/.env.example` | 模板 | 加字段时 |
| `media-server/Dockerfile` | 容器构建 | 改基础镜像 / 构建参数（Task 16 升级至 node:24） |
| `media-server/src/config/` | 配置加载与校验（TypeScript） | 加新字段时必改 |

**关键字段速查**（`media-server/.env`）：

```bash
HTTP_HOST=0.0.0.0
HTTP_PORT=3300

MEDIA_INTERNAL_TOKEN=dev-internal-token-change-me   # Go ↔ Node 鉴权，双方必须一致

MEDIASOUP_LISTEN_IP=0.0.0.0        # 绑定 IP
MEDIASOUP_ANNOUNCED_IP=            # 对外通告 IP；本机留空、公网填公网 IP

MEDIASOUP_RTC_MIN_PORT=40000       # RTC 端口范围
MEDIASOUP_RTC_MAX_PORT=40199

MEDIASOUP_MAX_ROUTERS=200          # 单 worker Router 上限（MVP 200 路够用）
```

### 5.4 前端（用户端 H5）

| 文件 | 用途 | 何时改 |
|---|---|---|
| `frontend/vite.config.js` | Vite 配置，含 HTTPS 自签 + `/api` `/ws` 代理 | 改后端地址 / 改端口 |
| `frontend/src/utils/request.js` | HTTP 客户端封装，`BASE_URL=''` 依赖 Vite 代理（开发）；生产由 nginx/caddy 同源 | 少改 |
| `frontend/src/services/websocket.js` | WebSocket 客户端，自动从 BASE_URL 推导 `ws/wss` | 少改 |
| `frontend/.nvmrc` | Node 版本锁定 24 | 升级 Node 时改 |

**VITE_DEV_BACKEND 切换后端**（可选环境变量）：

```bash
VITE_DEV_BACKEND=http://192.168.1.10:8085 npm run dev:h5
# 默认值：http://localhost:8085
```

### 5.5 后台管理端

| 文件 | 用途 |
|---|---|
| `admin/vite.config.js` | Vite 配置，`/api` 代理到 `localhost:8085`（**未启 HTTPS**，管理端不涉及 WebRTC） |
| `admin/.nvmrc` | Node 版本锁定 24 |

### 5.6 Git Hook / CI / 其它

| 文件 | 用途 |
|---|---|
| `scripts/start.sh` | 启动编排 |
| `scripts/stop.sh` | 停止 |
| `scripts/status.sh` | 状态检查 |
| `scripts/deploy-public.sh` | 公网部署 + 参数校验 |
| `scripts/dev-setup.sh` | 首次环境初始化（Docker 检查、拉中间件） |

---

## 六、故障排查 FAQ

### Q1：本机 `start.sh` 启动后前端报 "WebSocket 连接失败"

**排查顺序**：

1. `./scripts/status.sh` 看 Go 后端是否起来
2. `tail -100 .run/logs/backend.log` 看 Go 是否连上 Postgres / Redis
3. 浏览器 Network 看实际 WS URL 是不是 `wss://localhost:5173/ws`（应该经过 Vite 代理到 8085）

### Q2：进会议后只看到自己，看不到对端画面

**排查顺序**：

1. 浏览器控制台看 `[MediaEngine]` 日志，`consumer 创建成功` 应该有两条（audio + video）
2. `docker logs echochat-media-server | grep -i announcedip`，确认探测到的 IP 正确
3. **公网场景**：确认云安全组 UDP 40000-40199 已放通（最常见原因）
4. **公网场景**：在 `chrome://webrtc-internals` 里看 `Remote candidate type`，如果都是 `host`/`srflx` 但不互通，需要开 TURN

### Q3：形态 2 构建卡在 apt-get / npm ci

**最常见原因**：本机代理（Clash / Surge / v2ray）的 TUN 模式劫持了 Docker 容器 DNS 到 `198.18.0.0/16` fake-ip 池。

**验证**：看错误信息里域名解析到的 IP 是不是 `198.18.x.x`，是则就是这个问题。

**解法**：
- 临时：关掉代理软件的 TUN / 增强模式
- 长期：不用形态 2 在本机跑，直接去云服务器跑形态 3

### Q4：公网部署后访问 https://domain 但摄像头授权弹窗不出来

**原因**：未走 HTTPS 或证书不受信任。

**解法**：
- 检查 Caddy/Nginx 日志确认 Let's Encrypt 证书已签发
- 浏览器地址栏有没有锁图标？不受信任的自签证书在生产环境不可用

### Q5：公网部署后 coturn 容器起不来，日志报端口占用

**原因**：coturn 用 `network_mode: host`，如果宿主机已有其它 STUN/TURN 服务（例如你装了 nginx rtmp 模块），端口会冲突。

**解法**：`sudo lsof -i :3478` / `lsof -i :5349` 找出占用者，停掉或改 TURN 监听端口。

### Q6：改了 Go 代码，形态 2 下生效慢

**原因**：`go-service` 在容器里跑，改本机代码不会热更。

**解法**：

```bash
docker compose -f deploy/docker-compose.dev.yml up -d --build go-service
```

或者干脆开发期间用形态 1，上线前再用形态 2 验一次。

### Q7：Redis / Postgres 数据丢了

**原因**：`docker compose down -v` 会一起删 volume。普通 `docker compose down` 或 `stop.sh full` **不会**删 volume，数据保留。

**检查数据卷位置**：

```bash
docker volume inspect deploy_pgdata
docker volume inspect deploy_redisdata
docker volume inspect deploy_minio_data
```

---

## 七、附录

### 7.1 服务端口总览

| 服务 | 端口 | 协议 | 对外暴露 |
|---|---|---|---|
| 前端 Vite | 5173 | HTTPS | ✅（生产通过反代） |
| 管理端 Vite | 3100 | HTTP | ✅（管理员自用） |
| Go 后端 | 8085 | HTTP / WS | ✅（通过 `/api` `/ws`） |
| media-server | 3300 | HTTP | ❌ 内网 only（Go ↔ Node） |
| mediasoup RTC | 40000-40199 | UDP+TCP | ✅**必须放通** |
| Postgres | 5432 | TCP | ❌ 生产建议只 127.0.0.1 |
| Redis | 6379 | TCP | ❌ 同上 |
| MinIO API | 9000 | HTTP | ❌ 同上 |
| MinIO Console | 9001 | HTTP | ❌ 同上 |
| coturn STUN/TURN | 3478 | UDP+TCP | ✅（仅 public） |
| coturn TLS | 5349 | TCP | ✅（仅 public） |
| coturn Relay | 49160-49200 | UDP | ✅（仅 public） |

### 7.2 相关文档

- 架构设计：[docs/architecture/system-architecture.md](../architecture/system-architecture.md)
- 会议 MVP 部署（快速版）：[docs/deployment/meeting-mvp.md](../deployment/meeting-mvp.md)
- Phase 2e-2 设计：[docs/plans/2026-04-21-phase2e-2-design.md](../plans/2026-04-21-phase2e-2-design.md)
- 代码审查与修复记录：[docs/reviews/2026-04-23-phase2e-2-code-review.md](../reviews/2026-04-23-phase2e-2-code-review.md)
- 会议 API：[docs/api/frontend/meeting.md](../api/frontend/meeting.md)
- WebSocket 信令：[docs/api/websocket.md](../api/websocket.md)
- 项目总进度：[docs/progress/CURRENT_STATUS.md](../progress/CURRENT_STATUS.md)

### 7.3 版本与变更

- 2026-04-24 创建本文档，整合形态 1 / 2 / 3 的启动与部署操作（Phase 2e-2 Task 16 收官后）
- 同步更新项目：Dockerfile 升级至 Node 24、引入 `APT_MIRROR` / `APK_MIRROR` / `GO_BUILD_PROXY` 可选镜像源
