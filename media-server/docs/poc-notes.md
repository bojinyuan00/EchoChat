# Phase 2e-2 Task 0 — mediasoup PoC Spike 结论

> **状态：** ✅ 通过 / 技术栈可用性已验证
> **执行日期：** 2026-04-21
> **位置：** [`media-server/poc/`](../poc)
> **关联：**
> - [Phase 2e-2 设计文档](../../docs/plans/2026-04-21-phase2e-2-design.md)（§4.1 组件拓扑 / §4.2 Go-Node 协同时序）
> - [Phase 2e-2 实施计划](../../docs/plans/2026-04-21-phase2e-2-implementation.plan.md) §Task 0

---

## 一、目标与验收

按实施计划 §Task 0 要求，用最小代码验证：

1. mediasoup Worker + 单 Router 可在本机跑起来
2. 浏览器 ↔ Node 的完整 SFU 信令链路：`rtpCapabilities → Transport → Producer → Consumer`
3. 两个浏览器上下文之间能互相收到对方的音视频流
4. peer 断开时资源能被自动回收，无泄漏

**验收结果**：全部通过。下述第 §三 章记录了实际运行数据。

---

## 二、最小可行架构（PoC 版）

```
┌──────────────┐   ws://:3300/ws      ┌──────────────────────┐
│  Browser A   │──────信令──────────→ │   Fastify + ws       │
│  (mediasoup- │←──RtpCapabilities──  │                      │
│   client 3)  │                      │   ┌───────────────┐  │
└──────┬───────┘                      │   │ mediasoup     │  │
       │ DTLS/ICE/RTP                 │   │ Worker        │  │
       └──────────────────────────────┼──→│  └── Router   │  │
                                      │   │      ├─Trans.  │  │
       ┌──────────────────────────────┼──→│      ├─Prod.   │  │
       │ DTLS/ICE/RTP                 │   │      └─Cons.   │  │
┌──────┴───────┐                      │   └───────────────┘  │
│  Browser B   │──────信令──────────→ │   /healthz /stats    │
└──────────────┘                      └──────────────────────┘
```

PoC 刻意简化：**无权限、无房间、无鉴权**，全部 peer 加入同一个全局 Router，每个 peer 订阅所有其他 peer。真实 Go-Node 架构（设计文档 §4.2）中由 Go 承载这些权威状态。

信令协议（PoC 自定义，**与正式业务 WS 事件互不相关**，仅用于验证 mediasoup API）：

| 消息 | 方向 | 用途 |
|---|---|---|
| `getRtpCapabilities` | C→S | 取 Router capabilities |
| `createTransport {direction}` | C→S | 创建 send / recv Transport |
| `connectTransport {transportId, dtlsParameters}` | C→S | DTLS 握手 |
| `produce {transportId, kind, rtpParameters}` | C→S | 推流 |
| `consume {transportId, producerId, rtpCapabilities}` | C→S | 订阅远端流 |
| `resumeConsumer {consumerId}` | C→S | 恢复 paused consumer |
| `newProducer {peerId, producerId, kind}` | S→C 广播 | 通知其他 peer 新流出现 |
| `peerLeft {peerId}` | S→C 广播 | 通知其他 peer 离开 |

---

## 三、实际运行数据（2026-04-21）

### 3.1 启动阶段

```
mediasoup worker + router ready
    workerPid: 77170
    routerId: "f21e2f6a-1361-413c-8f2a-41a8290d0611"
    rtcPortRange: "40000-40099"
    listenIp: "0.0.0.0"
    announcedIp: "(unset, use local LAN ip)"
PoC ready: http://localhost:3300
```

`/healthz` 返回 `{ok:true, workerPid, routerId, peers:0}`，服务就绪。

### 3.2 2 peer 会议（Playwright 驱动 Chrome 双 tab）

两个 tab 分别点击「加入并推流」，控制台日志（Tab A）：

```
WS connected as peer adb1eff1
mediasoup Device loaded
local audio producer: 51c16d40
local video producer: 20d0d275
existing remote producers: 0            ← Tab A 先进，房间空
remote producer: peer=8dcec923 kind=audio   ← Tab B 加入后广播
remote producer: peer=8dcec923 kind=video
consuming audio from 8dcec923
consuming video from 8dcec923
```

Tab B：

```
existing remote producers: 2            ← Tab B 后进，拿到 A 的两条 producer
consuming audio from adb1eff1
consuming video from adb1eff1
```

**双向互通成立**。浏览器端 WebRTC `iceGatheringState` 正常走到 `complete`，`connectionState` 正常走到 `connected`。

### 3.3 资源统计（`/stats` 实测）

| 阶段 | peers | transports | producers | consumers | RSS(MB) |
|---|---:|---:|---:|---:|---:|
| 空闲 | 0 | 0 | 0 | 0 | 60 |
| Tab A 加入 | 1 | 2 | 2 | 0 | 60 |
| Tab B 加入 | 2 | 4 | 4 | 4 | 61 |
| Tab B 关闭 | 1 | 2 | 2 | 0 | 64 |

**资源回收链路**：WS `close` → `cleanupPeer` → 级联 close producers/consumers/transports → mediasoup `transportclose`/`producerclose` 事件 → 对端 consumer 自动清理 → 广播 `peerLeft`。实测对端 consumers 自动归零，无泄漏。

### 3.4 线性外推（单 Worker 单 Router）

| N 人会议 | transports | producers | consumers | 单 peer 入向 track |
|---:|---:|---:|---:|---:|
| 2 | 4 | 4 | 4 | 2 |
| 4 | 8 | 8 | 24 | 6 |
| 6 | 12 | 12 | 60 | 10 |
| 8 | 16 | 16 | 112 | 14 |

consumers 数量平方级增长：`N × (N-1) × 2`。MVP 硬上限 8 人，Node 单进程承载毫无压力；以 baseline `61MB / 2 peers` 推断，8 人会议常驻约 200MB RSS，远低于风险阈值。

---

## 四、关键坑与决策锁定

### 4.1 `announcedIp` 在 localhost 的处理

- **现象**：本机 Demo 场景下，若 `announcedIp` 留空 + `listenIp="0.0.0.0"`，mediasoup 生成的 `iceCandidates` 会包含 `0.0.0.0` 条目，Chromium 会自动替换为 `127.0.0.1` 与可用 LAN IP 完成 ICE 协商
- **结论**：本机 Demo **留空 `MEDIASOUP_ANNOUNCED_IP` 即可工作**，不必显式配 `127.0.0.1`，否则反而会让远端浏览器只拿到回环地址从而无法在同机多 tab 之外跑通
- **公网**：必须填服务器对外公网 IP（或 A 记录解析到的 IP），否则 ICE Candidate 全是内网 IP 客户端无法连上

### 4.2 DTLS 握手

- 默认 `enableUdp:true + enableTcp:true + preferUdp:true` 组合开箱即用
- 本机 Demo 全程走 UDP；公网若遇对称 NAT 走 coturn TURN 转发到 UDP
- **未遇到 DTLS 超时**；若后续线上出现，排查顺序：`udp 40000-40199 防火墙` → `announcedIp 正确性` → `TURN 凭证有效性` → `dtlsParameters 传参顺序（先 create 后 connect）`

### 4.3 mediasoup-client 在浏览器端的加载

- mediasoup-client **未提供预构建 UMD bundle**，不能用 `<script src>` 直接引入
- PoC 零构建步骤，使用 `https://esm.sh/mediasoup-client@3` 通过 ESM CDN 加载（实测解析到 3.19.0）
- **正式 frontend 模块（Task 9）必须改为 `npm i mediasoup-client@^3` + vite 打包**，避免线上依赖公共 CDN

### 4.4 Consumer 必须 paused-then-resume

- `transport.consume({paused:true})` → 客户端 `transport.consume(...)` → 服务端 `resumeConsumer` → 下发首帧
- 理由：若直接 `paused:false` 启动，某些浏览器会在 consumer 还没完成 RTP SDP 协商前就开始渲染，导致前几百毫秒丢帧或画面静止
- **本 PoC 完整复现了这个模式**，Task 2 / Task 9 需延续

### 4.5 Worker 崩溃保护

- `worker.on('died')` 在 PoC 中直接 `process.exit(1)` 触发外部重启
- **Task 1 正式实现** 需要按设计文档 §7.3 的约定升级为"监听 `died` → 重启 Worker + 通过内部通道告知 Go 层所有房间重建"

### 4.6 Simulcast 三档 encodings

PoC 已启用 3 档 encodings 并成功推拉：

```js
encodings: [
  { maxBitrate:   150_000, scaleResolutionDownBy: 4 },
  { maxBitrate:   400_000, scaleResolutionDownBy: 2 },
  { maxBitrate: 1_000_000 },
]
```

与设计文档 §7.4 / §11.5 的三档档位完全一致，MVP 可直接复用这组参数。

### 4.7 Playwright 自动化兼容性

Playwright 启动的 Chromium 在无真实摄像头环境下会使用**内置 fake 媒体源**（`--use-fake-device-for-media-stream` 默认启用），`getUserMedia` 能返回带假画面/静音的 MediaStream，PoC 验证完全可自动化。这项能力为后续 Task 16 的 E2E 自动化扫清障碍。

---

## 五、技术栈可用性判定（Task 0 决策关卡）

| 维度 | 判定 |
|---|---|
| mediasoup v3 在 Node 24 下可用 | ✅（原生模块 C++ 编译 53 秒内完成，无 Python/GCC 警告） |
| Fastify 4 + @fastify/websocket 信令栈 | ✅ |
| mediasoup-client 3 在最新 Chromium 可用 | ✅（通过 esm.sh 加载） |
| PoC 代码量 | 约 250 行 server + 220 行 client，总 ~500 行 |
| 学习曲线 | 比预期低（Transport/Producer/Consumer 三件套概念清晰） |
| 是否需要改选型（livekit-server） | ❌ 不需要，mediasoup 选型成立 |

**决策**：维持原选型 **mediasoup v3 + fastify + mediasoup-client 3**，按实施计划推进 Task 1-2（正式 `media-server/` 骨架 + 9 个 Internal REST API）。

---

## 六、对后续 Task 的输入

### Task 1（media-server 项目骨架）直接复用

- `package.json` 的依赖列表（mediasoup `3.14.11` / fastify `4.28.1` / pino）可直接平移
- `createWorker({rtcMinPort,rtcMaxPort,logLevel:"warn"})` + `createRouter({mediaCodecs})` 的代码范式
- Worker `died` 事件监听 + 自动退出给外部进程管理器

### Task 2（Node 9 个 REST API）直接复用

- `router.createWebRtcTransport(listenIps, enableUdp, enableTcp, preferUdp, initialAvailableOutgoingBitrate)` 参数已验证
- Consumer 必须先 `paused:true`，再通过独立 API `resume` 的契约
- Transport / Producer / Consumer 的 `Map<id, resource>` 管理模式
- 资源回收依赖 mediasoup 内建事件（`transportclose` / `producerclose` / `observer.close`），Node 层无需主动追踪

### Task 9（前端 mediasoup-client + Pinia Store）直接复用

- `SignalClient` 的 Promise 化 WS 请求（`reqId` 配对）可升级为 `utils/ws.js` 的 meeting 模块分发器
- `sendTransport.on('connect')` 与 `on('produce')` 的 callback-to-Promise 桥接模式
- Consumer 创建后调 `resumeConsumer` 的两步流程

### Task 14（双态部署）已验证参数

- `MEDIASOUP_LISTEN_IP` / `MEDIASOUP_ANNOUNCED_IP` / `MEDIASOUP_RTC_MIN_PORT` / `MEDIASOUP_RTC_MAX_PORT` 四个环境变量的语义与默认值
- UDP 端口范围 MVP 使用 `40000-40199`（200 端口，可承载约 25 用户级 Transport）

---

## 七、PoC 清理与归档

PoC 代码保留在 [`media-server/poc/`](../poc/)，不进入正式生产路径：

- `server.mjs`（248 行）
- `public/index.html`（101 行）
- `public/client.mjs`（215 行）
- `package.json`（依赖清单）
- `.gitignore`（`node_modules/`）

**后续处理策略**：
1. Task 1-2 完成后，`media-server/poc/` 作为参考样例保留在仓库，方便新成员上手
2. Task 16 收官时若归档到单独分支，可从主干移除

---

## 八、启动步骤（供后续开发者复现）

```bash
# 1. 安装依赖（mediasoup 需要编译 C++，macOS 需 Xcode Command Line Tools）
cd media-server/poc
npm install

# 2. 启动（默认监听 0.0.0.0:3300）
node server.mjs

# 3. 健康检查
curl http://localhost:3300/healthz
curl http://localhost:3300/stats

# 4. 浏览器验证：打开两个 Chrome 窗口（或 Chrome + Firefox）访问 http://localhost:3300/
#    两边都点「加入并推流」→ 授权摄像头/麦克风 → 应能互相看到对方画面
```

**环境变量（可选）**：

```bash
HTTP_PORT=3300                 # HTTP/WS 监听端口
MEDIASOUP_LISTEN_IP=0.0.0.0    # RTCTransport 监听
MEDIASOUP_ANNOUNCED_IP=        # 本机留空；公网填服务器公网 IP
MEDIASOUP_RTC_MIN_PORT=40000
MEDIASOUP_RTC_MAX_PORT=40099
```

---

## 九、变更记录

| 日期 | 作者 | 内容 |
|---|---|---|
| 2026-04-21 | Agent | Task 0 PoC Spike 首版落盘，技术栈可用性验证通过，维持 mediasoup 选型 |
