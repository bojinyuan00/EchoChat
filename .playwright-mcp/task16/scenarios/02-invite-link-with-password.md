# 场景 02：邀请链接加入（带密码）

## 目标

- 验证 `POST /api/v1/meeting/rooms/:code/invites`（签发邀请）+ `POST /api/v1/meeting/invites/:token/consume`（消费邀请）
- 覆盖设计 §6.2（邀请 token 7 天有效期）、§6.4（邀请链接 URL）
- **验证 Task 16 P0-4 修复**：密码不出现在 URL query，仅通过内存 state 或 Sec-Fetch 头传递

## 前置条件

依 [`../README.md`](../README.md) 通用前置。

## MCP 执行脚本

### 步骤 1：Alice 创建带密码的会议

1. 以 `task15_a` 登录
2. 会议首页 → 「创建会议（自定义）」→ 切换到有密码模式
3. 填充：
   - 会议主题：`Task16 邀请回归`
   - 密码：`invite@2026`
   - 其他留默认
4. 点「创建并进入」
5. 记录 `$ROOM_CODE`

### 步骤 2：Alice 生成邀请链接

1. 工具栏 → 「邀请」 → 「复制邀请链接」
2. 从剪贴板读取链接；格式示例：`https://localhost:5173/#/pages/meeting/preview?mode=join&code=xxx&invite_token=yyy`
3. **断言 URL query 不含 password 明文**（P0-4 核心断言）：
   ```js
   // browser_evaluate
   const link = await navigator.clipboard.readText()
   return { link, hasPassword: /password=/i.test(link) }
   ```
   期望：`hasPassword=false`

### 步骤 3：Bob 通过邀请链接加入

1. 开 Bob tab，已登录 `task15_b`
2. `browser_navigate` → 步骤 2 拿到的链接
3. 预览页展示「即将加入 `$ROOM_CODE` / 邀请人 Alice」+ 设备选择面板
4. 点「立即加入」
5. 期望：**无需输入密码**（邀请 token 已授权）
6. 截图 → `02-bob-joined-via-invite.png`

### 步骤 4：资源与 DB 断言

```bash
# meeting_invites.consumed_at 应被填充
mysql -e "SELECT invite_token, invitee_id, consumed_at FROM meeting_invites \
         WHERE room_id=(SELECT id FROM meeting_rooms WHERE room_code='$ROOM_CODE')" echochat

# meeting_participants 应有 Bob 行，joined_method='invite'
mysql -e "SELECT user_id, joined_method, invite_token FROM meeting_participants \
         WHERE room_id=(SELECT id FROM meeting_rooms WHERE room_code='$ROOM_CODE')" echochat
```

期望：
- `consumed_at` 时间戳为最近几秒
- Bob 行 `joined_method='invite'`，`invite_token` 与邀请链接一致

### 步骤 5：直接输入会议号（不含邀请 token）应要求密码

1. 第三个 tab 以 `task15_c` 登录
2. 会议首页 → 「通过会议号加入」 → 输入 `$ROOM_CODE`
3. 预览页 / 加入接口应返回 `ErrPasswordRequired` → UI 弹出密码输入框
4. 输入错误密码 `wrong`，断言返回 401 + UI 提示
5. 输入正确密码 `invite@2026`，成功加入
6. 截图 → `03-carol-password-join.png`

### 步骤 6：资源清理（场景结束）

Alice 结束会议；执行 [`01-create-and-join-by-code.md`](01-create-and-join-by-code.md) 步骤 6-7 的 Redis/DB 断言。

## 预期结果汇总

| 断言 | 预期 |
|---|---|
| 邀请 URL 不含 password | `hasPassword=false` |
| Bob 免密通过邀请加入 | HTTP 200 + WS `room.join` ACK |
| `meeting_invites.consumed_at` | 时间戳 ≈ 加入瞬间 |
| Carol 直接加入需要密码 | 401 → 输入后成功 |

## 失败排查

| 现象 | 定位 |
|---|---|
| URL 仍含 `password=` | P0-4 回归：`frontend/src/store/meeting.js` `draftJoinPayload` 是否将密码存内存 |
| Bob 使用邀请仍被要求密码 | `meeting_service.JoinRoom` 的 invite token 分支 |
| `consumed_at` 仍为 NULL | `MeetingInviteDAO.Consume` 未调 / 事务回滚 |
