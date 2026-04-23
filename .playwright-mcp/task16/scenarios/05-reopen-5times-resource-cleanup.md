# 场景 05：反复开关 5 次会议 —— 资源清理专项回归

## 目标

- **端到端验证 Step A 资源清理修复（提交 `f5ae095`）**：
  - 前端 `_engine` / `AudioContext` / `setInterval` / `_pendingBroadcastTimers` 在每次结束会议后完全释放
  - 后端 Redis `resource:*` / `member_state:*` / `host_grace:*` / `empty_ttl:*` 零累积
  - mediasoup Router / Transport / Producer / Consumer 不残留
- 回归 B1 / B2 / B3 三条资源清理路径
- 资源断言曲线：5 轮循环后 Redis `dbsize` 与 mediasoup `/internal/info` 恢复到 baseline

## 前置条件

依 [`../README.md`](../README.md) 通用前置；额外确认：

- 单浏览器 context 双 tab（Alice + Bob）
- 开启 Chromium DevTools → Memory → **勾选「Record allocation stacks」**，在脚本执行前 + 第 5 轮结束后各拍一次 heap snapshot

## MCP 执行脚本

### 步骤 0：baseline 采集

```bash
# baseline_redis.txt
redis-cli DBSIZE | tee /tmp/baseline_redis.txt
redis-cli KEYS 'echo:meeting:*' | wc -l >> /tmp/baseline_redis.txt

# baseline_media.json
curl -s -H "X-Internal-Token: $INTERNAL_TOKEN" http://localhost:3300/internal/info \
  | jq .stats > /tmp/baseline_media.json
```

在 Alice tab 的 DevTools Performance 面板录制整段脚本；脚本起止时刻各打一个 `performance.mark`。

### 步骤 1-5：循环 5 次创建→打通→结束

对 `i in {1..5}`：

1. **创建**：Alice tab 会议首页 → 创建即时会议 → 设备预览 → 立即进入（记录 `$ROOM_CODE_i`）
2. **加入**：Bob tab 通过会议号加入 `$ROOM_CODE_i`
3. **打通断言**（轻量）：两端 `browser_evaluate` 检查 `localAudioEnabled && localVideoEnabled`
4. **结束**：Alice 点「结束会议」；Bob tab 等待 `room.ended` toast
5. **结束后清理断言**（每轮都要做）：
   ```bash
   echo "===== Round $i ====="
   redis-cli KEYS "echo:meeting:resource:$ROOM_CODE_i:*"  # 必须为空
   redis-cli KEYS "echo:meeting:member_state:$ROOM_CODE_i:*"  # 必须为空
   redis-cli KEYS "echo:meeting:host_grace:$ROOM_CODE_i"      # 必须 nil
   redis-cli KEYS "echo:meeting:empty_ttl:$ROOM_CODE_i"       # 必须 nil
   ```
6. **前端资源断言**（每轮）：
   ```js
   // 在 Alice tab 的结束页执行
   const store = window.__pinia?.state?.value?.meeting
   return {
     round: $i,
     localState: store.localState,                // 'ended'
     pendingBroadcastTimers: window.__broadcastTimersSize ?? 'N/A',
   }
   ```
7. **回首页**：Alice tab 物理返回（`browser_press_key` Back / 点击结束页「返回」按钮）触发 `onUnload` → `exitEndedRoom`
8. `browser_evaluate` 再次确认 pinia state 全 reset：`localState='idle' && participantCount=0 && !engineExists`
9. Bob tab 同样回首页
10. **延迟 2 秒**模拟用户操作间隙，进入下一轮

### 步骤 6：5 轮结束后的累积断言（核心）

```bash
# 对比 baseline，必须零增量
echo "===== Redis 累积对比 ====="
redis-cli DBSIZE
redis-cli KEYS 'echo:meeting:*' | wc -l
diff /tmp/baseline_redis.txt <(redis-cli DBSIZE && redis-cli KEYS 'echo:meeting:*' | wc -l)

echo "===== media-server 累积对比 ====="
curl -s -H "X-Internal-Token: $INTERNAL_TOKEN" http://localhost:3300/internal/info | jq .stats > /tmp/after5_media.json
diff /tmp/baseline_media.json /tmp/after5_media.json

echo "===== MySQL 快照 ====="
mysql -e "SELECT COUNT(*) AS active FROM meeting_rooms WHERE status='active'" echochat
# 期望：active=0
mysql -e "SELECT COUNT(*) AS alive FROM meeting_participants WHERE left_at IS NULL" echochat
# 期望：alive=0
```

### 步骤 7：前端内存快照对比

1. 在 Alice tab 打开 DevTools → Memory → 拍 heap snapshot #2（5 轮结束后）
2. 对比 #1（baseline）与 #2：
   - 关键类：`MediaStream` / `RTCPeerConnection` / `AudioContext` / `Consumer` / `Producer` 的对象数量
   - 允许少量增长（mediasoup-client `Device` 缓存），但必须远小于 5×
3. 记录 Performance tab 的「JS Heap」曲线：应呈锯齿下降（每次 GC 回收），**不得持续单调上升**

## 预期结果汇总

| 断言维度 | 期望 |
|---|---|
| 每轮 Redis 清理 | `resource/member_state/host_grace/empty_ttl` 全空 |
| 5 轮累积 Redis `DBSIZE` | 与 baseline 一致（允许 ±2 容错，针对 session/heartbeat key） |
| 5 轮累积 media-server | Router / Transport / Producer / Consumer 回到 baseline |
| MySQL | `active` 会议数 = 0，`left_at IS NULL` 参与者 = 0 |
| 前端 heap snapshot | 关键对象类数量增长 ≤ 2（近似常数，非线性） |
| 前端 Performance | JS Heap 曲线呈锯齿，无持续上升趋势 |
| 控制台 | 无 `unhandled promise rejection` / `NotReadableError` / `leaked listener` |

## 资源清理验证矩阵（对应 Step A 修复）

| 代码路径 | 验证手段 | 通过条件 |
|---|---|---|
| F1 `_onRoomEnded` 结束页数据保留 | `localState='ended'` 时 `currentRoom` 非 null | 结束页正确渲染 |
| F1 `exitEndedRoom` 动作 | onUnload 后 `currentRoom=null` | pinia state reset |
| F2 `endMeeting` 本地兜底 | 断开 WS 再结束（额外场景） | 本地仍能触发 `_onRoomEnded` |
| F3 `_pendingBroadcastTimers` | 结束后 `clearTimeout` | `window.__broadcastTimersSize=0` |
| B1 `cleanupRoomRedisResidual` | 每轮 Redis 查 | `resource:*` / `member_state:*` = 0 |
| B3 `OnRoomEnded` timer 取消 | pprof / sync.Map 大小 | `graceTimers` / `emptyTTLTimers` 稳定无增长 |

## 失败排查

| 现象 | 定位 |
|---|---|
| 某轮 `resource:*` 非空 | `EndRoom` 未走 cleanup → 检查 commit `f5ae095` 是否正确落地 |
| Redis `DBSIZE` 持续增长 | 检查 session/heartbeat key 是否泄漏；排除 meeting 模块 |
| 前端 heap 持续上升 | 抓 heap diff，找保留路径（常见：`Producer._events`、`Vue reactive` 持有 Consumer） |
| `localState` 卡 `ended` 不回 `idle` | `onUnload` 未调 `exitEndedRoom` 或 `room.vue` 组件未卸载（检查 H5 `uni.reLaunch` 行为） |

## 延伸测试（可选）

- **10 轮**：循环次数改 10，观察是否线性无增长
- **跨 tab 并发**：第 3 轮时 Bob 主动 leave（而非等 Alice end），验证双路径清理一致
- **主动断网**：第 4 轮打通后 Alice 关网卡 2 分钟 → 触发 host grace → 进入场景 04 路径
