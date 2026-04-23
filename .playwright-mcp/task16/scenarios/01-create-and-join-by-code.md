# 场景 01：创建会议 → 会议号加入 → 双向音视频打通 → 主持人结束

## 目标

- 验证 `POST /api/v1/meeting/rooms`（创建）+ `POST /api/v1/meeting/rooms/:code/join`（加入）+ mediasoup 信令全链路打通
- 覆盖设计 §6.1（REST 建房）、§6.3（WS 信令 11 事件）、§6.6（NodeClient 媒体编排）
- 配套断言 Step A 资源清理：结束会议后 Redis / MySQL / media-server 全部归零

## 前置条件

依 [`../README.md`](../README.md) 通用前置；额外确认：

- 两个浏览器上下文（MCP 多 tab）分别以 `task15_a` / `task15_b` 登录
- Alice tab 为主持人，Bob tab 为参会人

## MCP 执行脚本

### 步骤 1：Alice 登录并创建即时会议

1. `browser_navigate` → `https://localhost:5173/#/pages/auth/login`
2. `browser_fill_form` → 账号 `task15_a` / 密码 `Task15@2026` → 点 "登录"
3. 断言首页进入「消息」tab
4. 底部切换到「会议」tab（`pages/meeting/index`）
5. 点「创建即时会议」
6. 设备预览页：默认勾选麦克风 + 摄像头；点「立即进入」
7. **进入 `pages/meeting/room`**，截图 → `01-alice-created.png`
8. 读取 URL query 得到 `roomCode`，记录到上下文变量 `$ROOM_CODE`

### 步骤 2：Bob 通过会议号加入

1. 开第二个 browser context；`browser_navigate` → `https://localhost:5173/#/pages/auth/login`
2. 以 `task15_b` 登录
3. 切到「会议」tab，点「通过会议号加入」
4. 输入 `$ROOM_CODE`
5. 预览页 → 「立即加入」
6. 截图 → `02-bob-joined.png`

### 步骤 3：双向音视频打通断言

- Alice tab 成员面板应展示 2 个成员，Bob 的 AV 图标为蓝色（开启状态）
- Bob tab 成员面板同理
- Alice 的视频宫格应能看到 Bob 的视频流（`VideoTile[user_id=56] video.srcObject` 非空）
- 控制台验证 mediasoup 关键日志：
  - `[MediaEngine] producer 创建成功 kind=audio`
  - `[MediaEngine] producer 创建成功 kind=video`
  - `[MediaEngine] consumer 创建成功`
- `browser_evaluate`:
  ```js
  const store = window.__pinia?.state?.value?.meeting
  return {
    participantCount: store.participants.length,
    localAudio: store.localAudioEnabled,
    localVideo: store.localVideoEnabled,
    remoteUserCount: Object.keys(store.remoteConsumers).length,
  }
  ```
  期望：`participantCount=2`、`localAudio=true`、`localVideo=true`、`remoteUserCount=1`

### 步骤 4：资源创建态断言

在第三个终端执行：

```bash
# media-server 应有 1 Router + 4 Transport（2 users × send/recv）+ 4 Producer + 4 Consumer
curl -s -H "X-Internal-Token: $INTERNAL_TOKEN" http://localhost:3300/internal/info | jq .stats

# Redis 应有两个用户的资源追踪 Set
redis-cli KEYS "echo:meeting:resource:$ROOM_CODE:*"
# 期望输出：echo:meeting:resource:xxx:55 / echo:meeting:resource:xxx:56
# 每个 SMEMBERS 应有 5 条（transport×2 + producer×2 + consumer×2 之类，具体取决于 producer.new 收敛顺序）
redis-cli SMEMBERS "echo:meeting:resource:$ROOM_CODE:55"
redis-cli SMEMBERS "echo:meeting:resource:$ROOM_CODE:56"
```

### 步骤 5：Alice 结束会议

1. Alice tab 工具栏 → 「结束会议」→ 弹窗确认「主持人结束」
2. 截图 → `03-alice-ended.png`（页面转为「会议已结束」提示页）
3. Bob tab 应收到 WS `meeting.room.ended` 广播，展示「会议已结束」toast，1.2s 后跳回会议首页
4. 截图 → `04-bob-received-ended.png`

### 步骤 6：资源释放断言（核心验证 Step A）

```bash
# 1. Redis 清理验证（P1-4 + Step A B1/B3 的核心断言）
redis-cli KEYS "echo:meeting:resource:$ROOM_CODE:*"         # 期望：(empty array)
redis-cli KEYS "echo:meeting:member_state:$ROOM_CODE:*"     # 期望：(empty array)
redis-cli KEYS "echo:meeting:host_grace:$ROOM_CODE"         # 期望：(nil)
redis-cli KEYS "echo:meeting:empty_ttl:$ROOM_CODE"          # 期望：(nil)

# 2. MySQL 状态验证
mysql -e "SELECT status, ended_reason FROM meeting_rooms WHERE room_code='$ROOM_CODE'" echochat
# 期望：status=ended, ended_reason=host_ended

mysql -e "SELECT user_id, left_at, left_reason FROM meeting_participants WHERE room_id=(SELECT id FROM meeting_rooms WHERE room_code='$ROOM_CODE')" echochat
# 期望：Alice + Bob 都有 left_at ≠ NULL, left_reason='host_end'

# 3. media-server Router / Transport / Producer / Consumer 全部归零
curl -s -H "X-Internal-Token: $INTERNAL_TOKEN" http://localhost:3300/internal/info | jq '.stats | .routers, .transports, .producers, .consumers'
# 期望：对应 $ROOM_CODE 的 Router 已销毁；整体 stats 可能非 0（其他并行测试）但本场 Router 必须不存在
```

### 步骤 7：前端资源释放断言

在 Alice tab 执行 `browser_evaluate`:

```js
// 结束页停留时的 state 快照
const store = window.__pinia?.state?.value?.meeting
return {
  localState: store.localState,       // 期望：'ended'
  engineExists: !!window.__engine,    // 期望：false（_cleanupMedia 已置空）
  participantCount: store.participants.length,  // 结束页保留渲染数据，非 0
}
```

然后手动返回首页（模拟 `onUnload` 触发）：

1. `browser_navigate_back` 或点结束页的「返回首页」按钮
2. 再次 `browser_evaluate` 断言 pinia state：
   ```js
   const store = window.__pinia?.state?.value?.meeting
   return {
     localState: store.localState,       // 期望：'idle'
     participantCount: store.participants.length,  // 期望：0
     currentRoom: store.currentRoom,     // 期望：null
   }
   ```

## 预期结果汇总

| 断言维度 | 预期 |
|---|---|
| WS 信令 | `room.join` / `producer.new` / `consume` 全链路无异常 |
| 双向媒体 | 两端视频宫格均能播放对方画面，`localAudioEnabled=true` 且 `remoteConsumers` 含对方 |
| Redis 清理 | `resource:*` / `member_state:*` / `host_grace` / `empty_ttl` 全空 |
| DB 一致 | `meeting_rooms.status=ended`、两行 `meeting_participants.left_at` 非空 |
| media-server | 对应 Router 已删除，level transport/producer/consumer 观察端点无残留 |
| 前端 | `_engine=null` / `_pendingBroadcastTimers.size=0`；`onUnload` 后 pinia state 全部 reset |

## 失败排查

| 现象 | 可能原因 | 排查点 |
|---|---|---|
| Bob 看不到 Alice 的画面 | `pushExistingRoomState` 同步性（Task 16 P1-8 修复） | `logs.Info` 搜 `room.join` 是否先于 `member.joined` |
| 结束后 Redis `resource:*` 仍存 | `EndRoom` 未走 `cleanupRoomRedisResidual`（Step A B1 回归） | `grep "cleanupRoomRedisResidual" backend/go-service/app/meeting/service/meeting_service.go` |
| `localState` 卡在 `connected` | `_onRoomEnded` 未触发 | 检查 Bob tab 是否收到 `meeting.room.ended` WS 事件（网络面板 frame） |
