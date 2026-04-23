# EchoChat 会议 MVP 部署指南

> **适用阶段**：Phase 2e-2（会议 MVP ✅ 已完成）
> **覆盖形态**：本机 Demo（零配置）+ 公网部署（含 TURN）
> **最后更新**：2026-04-24（Phase 2e-2 Task 16 收官：E2E 回归 + 代码审查修复 + 资源生命周期审计；本指南 Task 14 双态部署脚本仍为最新有效版本，Task 16 追加 `REDIS_PASSWORD` 与 `redis.conf requirepass` 联动校验，详见 [Task 16 收官说明](../progress/CURRENT_STATUS.md)）

## 一、双态部署总览

EchoChat 会议 MVP 按 Phase 2e-2 设计决策 D01 采用**"一套代码 + 环境变量切换"** 的双态策略，不做代码分支。

| 对比项 | 本机 Demo（local） | 公网部署（public） |
|---|---|---|
| 适用场景 | 开发调试、同局域网演示 | 上线、跨公网用户 |
| `MEDIASOUP_ANNOUNCED_IP` | 留空（自动探测） | **必填**服务器公网 IP 或域名解析 A 记录 IP |
| `TURN_ENABLED` | `false`（不起 coturn） | `true`（coturn 走 `profiles: [public]`） |
| 防火墙/安全组 | 无特殊要求 | 需放通 UDP:40000-40199 + TURN 端口 |
| 启动方式 | `./scripts/start.sh full` | `./scripts/deploy-public.sh` |
| 关键文件 | `deploy/.env.local.example` | `deploy/.env.public.example` |

服务拓扑对比见设计文档 [docs/plans/2026-04-21-phase2e-2-design.md §4.3 双态部署拓扑](../plans/2026-04-21-phase2e-2-design.md)。

---

## 二、本机 Demo 部署

**前置要求**：Docker Desktop（含 Compose V2）、Node 20+、Go 1.22+

### Step 1：复制环境变量文件

```bash
cp deploy/.env.local.example deploy/.env
```

本机模式下无需修改任何字段即可跑通（`MEDIASOUP_ANNOUNCED_IP` 留空，`TURN_ENABLED=false`）。

### Step 2：一键启动全栈

```bash
./scripts/start.sh full
```

这会做以下事情：
1. `docker compose up -d --build` 拉起 5 个容器：`postgres / redis / minio / go-service / media-server`
2. 等待 `media-server` healthcheck 通过（最多 180 秒）
3. 用本地进程启动 `frontend`（5173）+ `admin`（3100）—— 热更体验更好

启动完成后会打印访问地址：

```
前台用户端 (H5):   http://localhost:5173
后台管理端:        http://localhost:3100
Go 后端 API:       http://localhost:8085
媒体服务器 (SFU):  http://localhost:3300 (healthz: /healthz)
MinIO 控制台:      http://localhost:9001 (echochat / echochat123456)
```

### Step 3：验证

```bash
./scripts/status.sh
# 预期全部 ● RUNNING

curl http://localhost:8085/healthz         # Go backend
curl http://localhost:3300/healthz         # media-server
curl http://localhost:3300/readyz          # mediasoup worker 已就绪
```

### 停止

```bash
./scripts/stop.sh full              # 停止 docker compose 全栈（保留数据卷）
./scripts/stop.sh                   # 仅停应用层（保留所有容器）
./scripts/stop.sh --all             # 含数据库中间件
```

---

## 三、公网部署

**前置要求**：
- 服务器有**公网 IP** 或已解析 A 记录的域名
- Docker 已安装
- 云安全组/iptables 可由你控制（不能仅有 80/443 的 PaaS）

### Step 1：复制模板并替换所有占位符

```bash
cp deploy/.env.public.example deploy/.env
vim deploy/.env
```

**必须替换**：所有 `_REPLACE_WITH_*_` 占位符（否则 `deploy-public.sh` 会拒绝启动）。

生成强密码的示例：

```bash
openssl rand -hex 32    # JWT_SECRET / MEDIA_INTERNAL_TOKEN
openssl rand -base64 24 # 数据库密码 / TURN 密码
```

**核心字段**：

| 字段 | 示例 | 说明 |
|---|---|---|
| `MEDIASOUP_ANNOUNCED_IP` | `203.0.113.42` | **必填**服务器公网 IP；不能写 `127.0.0.1` |
| `TURN_ENABLED` | `true` | 对称 NAT fallback 必备 |
| `TURN_USERNAME / TURN_PASSWORD` | 自设 | 避免 TURN 账号被互联网滥用 |
| `JWT_SECRET` | 64 hex | 生产必改 |
| `MEDIA_INTERNAL_TOKEN` | 32 hex | Go ↔ Node 内部鉴权，必须与 `media-server/.env` 一致 |

### Step 2：放通云安全组 / iptables

本脚本**不会**自动改防火墙（不同云厂商命令不同），请你提前在云控制台/iptables 放通：

| 协议 | 端口 | 用途 |
|---|---|---|
| TCP | 8085 | Go backend API |
| TCP | 5173 | 前台用户端 H5（如单独暴露） |
| UDP | 40000-40199 | mediasoup RTC 流量 |
| TCP | 40000-40199 | mediasoup ICE-TCP fallback |
| UDP | 3478 | coturn STUN/TURN |
| TCP | 3478 | coturn |
| TCP | 5349 | coturn TLS |
| UDP | 49160-49200 | coturn 中继流量 |

`iptables` 示例（仅参考，请结合实际 zone 调整）：

```bash
# UDP 范围
sudo iptables -I INPUT -p udp --dport 40000:40199 -j ACCEPT
sudo iptables -I INPUT -p udp --dport 49160:49200 -j ACCEPT
# STUN/TURN
sudo iptables -I INPUT -p udp --dport 3478 -j ACCEPT
sudo iptables -I INPUT -p tcp --dport 3478 -j ACCEPT
sudo iptables -I INPUT -p tcp --dport 5349 -j ACCEPT
# API
sudo iptables -I INPUT -p tcp --dport 8085 -j ACCEPT
```

### Step 3：运行部署脚本

```bash
./scripts/deploy-public.sh
```

脚本会按 4 步依次执行：

1. **校验 `.env`**：占位符、长度、`ANNOUNCED_IP` 必填、`DEPLOY_MODE=public` 等
2. **本机端口检查**：TCP:8085 / TCP:3300 / 潜在冲突
3. **Docker 环境**：docker daemon + compose v2 可访问
4. **启动**：`TURN_ENABLED=true` 时用 `--profile public`（含 coturn），否则跳过

启动完成后会打印访问地址。此时用浏览器访问 `http://<ANNOUNCED_IP>:5173`（或你放通的端口）即可。

### 验证

```bash
./scripts/status.sh

# 容器级检查
docker compose -f deploy/docker-compose.dev.yml ps
docker logs echochat-media-server | tail -40
docker logs echochat-coturn | tail -20
```

公网验证要点：
- 用**两台不同网络**的设备（比如手机 4G + 家里 WiFi）分别入会，双向能看到对端画面
- Chrome 调 `chrome://webrtc-internals` 看 `Remote candidate type` —— 对称 NAT 场景能看到 `relay` 类型，说明 TURN 生效
- `curl http://<ANNOUNCED_IP>:8085/healthz` 返回 200

### 停止

```bash
./scripts/stop.sh full
# 等价于：docker compose -f deploy/docker-compose.dev.yml --profile public stop
```

---

## 四、常见问题 FAQ

### Q1：`docker compose up` 卡在 media-server 构建，报 `python3 not found` 或 `g++ not found`？

**原因**：mediasoup 的 C++ worker 需要在构建阶段编译原生代码。`media-server/Dockerfile` 的 builder 阶段已装 python3 + build-essential；若你绕过 Dockerfile 本地构建，需要手动装这些。

```bash
# Debian/Ubuntu
sudo apt install python3 build-essential pkg-config
# macOS
brew install python3
xcode-select --install
```

### Q2：本机 Demo 下两个浏览器 Tab 都在 127.0.0.1:5173，但视频流对端一片黑？

**检查 1**：`MEDIASOUP_ANNOUNCED_IP` 一定不要写 `127.0.0.1`，留空让 mediasoup 自动探测到局域网 IP 即可。

**检查 2**：浏览器摄像头权限是否授予（Chrome 左上角锁图标）。

**检查 3**：`docker logs echochat-media-server | grep announcedIp`，确认探测到的 IP 在你本机 `ifconfig` 能看到。

### Q3：公网部署后用手机（4G 网络）加入会议，能看到自己但看不到别人？

**典型症状**：单向 ICE 候选协商不成功。通常是以下之一：

1. **TURN 没打开** —— 确认 `TURN_ENABLED=true` 且 `docker ps` 里有 `echochat-coturn`。对称 NAT 用户必走 TURN。
2. **UDP:40000-40199 没放通** —— mediasoup 要在这个范围分配动态 RTC 端口。很多云默认只放 80/443/22。
3. **`MEDIASOUP_ANNOUNCED_IP` 写错** —— 如果是 NAT 机器且写了内网 IP，公网用户拿到的候选地址不可达。

### Q4：为什么 coturn 用 `network_mode: host`？

TURN 服务器为每个会话动态分配中继端口（`--min-port=49160 --max-port=49200`）；Docker 默认 bridge 网络的端口映射会把回包的源端口改写，导致客户端收不到流量。host 网络能绕过这个问题，是 coturn 官方推荐做法。

代价：coturn 会占用宿主机端口命名空间，请确保 `3478 / 5349 / 49160-49200` 宿主机无占用。

### Q5：我有 HTTPS 证书，想把前端端口改成 443？

MVP 阶段脚本没封装 HTTPS 反代，建议在 compose 外层加 Nginx（或 Caddy）作为 TLS 终结。mediasoup 的 RTC 流量本身就走 DTLS 加密，不需要你额外处理。

Caddy 最小示例（独立容器或宿主进程）：

```
yourdomain.com {
  reverse_proxy /api/* localhost:8085
  reverse_proxy /ws localhost:8085
  reverse_proxy * localhost:5173
}
```

### Q6：public profile 下 `stop.sh full` 能停掉 coturn 吗？

能。`scripts/stop.sh full` 里写了 `docker compose --profile public stop`，profile 参数确保 coturn 这种默认不暴露的服务也被覆盖。

### Q7：数据库要备份怎么操作？

```bash
# 快速备份当前数据库
docker exec echochat-postgres pg_dump -U echochat echochat | gzip > backup-$(date +%F).sql.gz

# 从备份恢复（容器停机时）
gunzip -c backup-2026-04-23.sql.gz | docker exec -i echochat-postgres psql -U echochat -d echochat
```

`pgdata / redisdata / minio_data` 三个 volume 的物理位置由 Docker 管理：

```bash
docker volume inspect deploy_pgdata
```

---

## 五、相关文档

- 设计文档：[docs/plans/2026-04-21-phase2e-2-design.md](../plans/2026-04-21-phase2e-2-design.md)
- 实施计划：[docs/plans/2026-04-21-phase2e-2-implementation.plan.md](../plans/2026-04-21-phase2e-2-implementation.plan.md)
- 会议 API：[docs/api/frontend/meeting.md](../api/frontend/meeting.md)
- WebSocket 信令：[docs/api/frontend/meeting.md#websocket-信令协议](../api/frontend/meeting.md)
- media-server 子项目：[media-server/README.md](../../media-server/README.md)（如有）
