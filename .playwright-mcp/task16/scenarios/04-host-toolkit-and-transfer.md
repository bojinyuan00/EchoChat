# 场景 04：主持人四件套 + 宽限期自动转让

## 目标

- **验证 Task 15 主持人四件套**：请他静音、请他开麦、转让主持人、踢出会议
- **验证 Task 8 host 宽限期状态机**（P1-1 原子转让 + P1-7 处理锁）：
  - host 主动断开 WS → 宽限期内重连 → 撤销转让
  - host 宽限期到期 → 自动转让给最早加入的参会人
- 覆盖设计 §6.5（生命周期状态机）、§6.6（权限矩阵）

## 前置条件

依 [`../README.md`](../README.md) 通用前置；**需 3 个账号 Alice/Bob/Carol 同时入会**。

## MCP 执行脚本 A：主持人四件套

### 步骤 1：三人入会

1. Alice 创建即时会议 → `$ROOM_CODE`
2. Bob 用会议号加入
3. Carol 用会议号加入
4. 截图 → `01-three-joined.png`

### 步骤 2：Alice 请他静音 Bob

1. Alice tab 打开成员面板 → Bob 右侧 「⋯」 菜单 → 「请他静音」
2. Bob tab 期望：
   - Toast：「主持人已将你的麦克风关闭」
   - `localAudioEnabled=false`
   - 成员面板自己的 AV 图标变灰
3. Alice tab 期望：Bob 行 AV 图标变灰
4. 截图 → `02-bob-muted-by-host.png`

### 步骤 3：Alice 请他开麦 Bob

1. Alice tab → Bob 「⋯」菜单 → 「请他开麦」
2. Bob tab 期望：
   - 弹窗请求确认「主持人请你开启麦克风，是否同意？」
   - 点「同意」后 `localAudioEnabled=true`
3. 截图 → `03-bob-ask-unmute.png`

### 步骤 4：Alice 踢出 Carol

1. Alice tab → Carol 「⋯」菜单 → 「踢出会议」
2. 弹窗确认「踢出 Carol？」→ 确认
3. Carol tab 期望：
   - 收到 `meeting.member.kicked` 广播
   - Toast：「主持人已将你移出会议」
   - 跳回会议首页
4. Alice / Bob tab 期望：Carol 行从成员列表消失
5. 截图 → `04-carol-kicked.png`

### 步骤 5：Alice 转让主持人给 Bob

1. Alice tab → Bob 「⋯」菜单 → 「转让主持人」
2. 弹窗确认 → 确认
3. **原子性验证**（P1-1）：
   ```bash
   mysql -e "SELECT host_id, (SELECT role FROM meeting_participants WHERE room_id=r.id AND user_id=55) AS alice_role, \
             (SELECT role FROM meeting_participants WHERE room_id=r.id AND user_id=56) AS bob_role \
             FROM meeting_rooms r WHERE room_code='$ROOM_CODE'" echochat
   ```
   期望同时满足：`host_id=56` + `alice_role=participant` + `bob_role=host`（同一事务原子更新）
4. Alice tab / Bob tab 工具栏切换（Alice 看到"结束会议"按钮消失，Bob 看到它出现）
5. 截图 → `05-host-transferred.png`

## MCP 执行脚本 B：宽限期自动转让

### 前置：接续场景 A 的会议；此时 Bob 是主持人，Alice 是参会人

### 步骤 6：Bob 断开 WS（模拟网络断开）

1. Bob tab DevTools → Network 面板 → Throttling 切换到 **Offline**
2. 等待 WS 心跳超时（~15s），Bob 客户端 `_onWsDisconnected` 触发
3. 期望 Redis 立即写入 `echo:meeting:host_grace:$ROOM_CODE`：
   ```bash
   redis-cli GET "echo:meeting:host_grace:$ROOM_CODE"
   # 期望：JSON {host_id:56, started_at:..., grace_until:...}
   redis-cli TTL "echo:meeting:host_grace:$ROOM_CODE"
   # 期望：约 120 秒（配置 `meeting.host_grace_seconds`）
   ```
4. Alice tab 期望：工具栏显示「主持人掉线中（宽限期 2:00）」倒计时（若 UI 已实现）

### 步骤 7a：宽限期内重连（撤销转让）

1. 15 秒后 Bob tab Network 恢复 Online
2. 期望 `_onWsReconnected` 触发，执行 `OnHostReconnect`：
   ```bash
   redis-cli EXISTS "echo:meeting:host_grace:$ROOM_CODE"
   # 期望：0（key 已 DEL）
   ```
3. Alice tab 工具栏倒计时消失
4. 查日志：
   ```bash
   tail -50 backend/go-service/logs/*.log | grep "host 宽限期内重连"
   # 期望：`host 宽限期内重连，撤销转让计划`
   ```

### 步骤 7b：宽限期到期（自动转让）

**若步骤 7a 已执行，请先让 Bob 重新 Offline 并等待整个 120s 周期完成。**

1. Bob tab Offline，保持 120+s
2. 期望本地 timer 到期触发 `HandleHostGraceExpired`：
   - Alice tab 收到 `meeting.host.changed` 广播，`auto_reason='host_grace_expired'`
   - 工具栏/成员面板 Alice 头像变为主持人标识
   - `meeting_rooms.host_id=55` + `meeting_participants.role` 原子变更
3. **处理锁验证（P1-7）**：
   ```bash
   # 宽限期到期瞬间抓一下处理锁
   redis-cli GET "echo:meeting:host_grace_handling:$ROOM_CODE"
   # 期望：`1`（SETNX 成功 60s TTL），60s 后自然过期
   ```
4. 截图 → `06-host-auto-transferred.png`

### 步骤 8：清理

Alice（现任主持人）结束会议，走场景 01 的资源清理断言。

## 预期结果汇总

| 断言 | 预期 |
|---|---|
| 请他静音 | Bob `localAudioEnabled=false` + toast |
| 请他开麦 | Bob 弹窗确认后 `localAudioEnabled=true` |
| 踢出 | Carol 跳首页 + 成员列表移除 |
| 主动转让原子性 | `host_id` + 两行 `role` 同事务更新 |
| host 宽限期 Redis | key 写入 + TTL≈120s |
| 宽限内重连 | key DEL + 本地 timer cancel |
| 宽限到期自动转让 | 最早加入者成为新 host + 广播 `host.changed` |
| 处理锁 SETNX | `host_grace_handling:*` 存在 60s |

## 失败排查

| 现象 | 定位 |
|---|---|
| 转让后 `host_id` 和 `role` 不一致 | P1-1 回归：检查 `TransferHost` 是否包在单事务 |
| 宽限到期但 host 不变 | 看 handling 锁是否被"自然过期 + DEL=0"误拦（P1-7 应已修） |
| 多节点部署下双重转让 | SETNX 处理锁应保证单节点唯一处理 |
