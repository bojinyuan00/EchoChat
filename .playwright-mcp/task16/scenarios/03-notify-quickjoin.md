# 场景 03：通知中心「立即加入」按钮跳转

## 目标

- 验证邀请被签发时，受邀人通知中心实时刷新 + 卡片双按钮（「立即加入」/「稍后」）
- 覆盖设计 §6.4（邀请通知类型 `meeting_invitation`）
- 过期卡片应渲染为 disabled 单按钮「邀请已过期」

## 前置条件

依 [`../README.md`](../README.md) 通用前置；Alice 已在会议中（复用场景 02 的 `$ROOM_CODE`），或新开一场会议。

## MCP 执行脚本

### 步骤 1：Alice 邀请 Bob

1. Alice tab 工具栏 → 「邀请」 → 「发送给联系人」
2. 联系人弹窗选择 Bob → 确认
3. 期望：UI toast「邀请已送达」，关闭弹窗

### 步骤 2：Bob 通知中心实时接收

1. Bob tab（已登录 `task15_b`）→ 底部切到「通知」tab
2. 期望：最顶部新增卡片
   - 类型标识：`会议邀请`
   - 内容：`Alice 邀请你加入会议 $ROOM_CODE`
   - 双按钮：**「立即加入」 + 「稍后」**
3. 截图 → `01-notify-new-invite.png`
4. `browser_evaluate` 断言：
   ```js
   const store = window.__pinia?.state?.value?.notify
   const latest = store.list[0]
   return {
     type: latest.notify_type,            // 期望：'meeting_invitation'
     expired: latest.expired_at < Date.now()/1000,  // 期望：false
     hasButtons: !!latest.ext?.room_code  // 期望：true
   }
   ```

### 步骤 3：点击「立即加入」跳转

1. 点「立即加入」
2. 期望跳转 `pages/meeting/preview?mode=join&code=$ROOM_CODE&invite_token=...`
3. 预览页正常展示
4. 点「立即加入」完成入会
5. 截图 → `02-bob-joined-from-notify.png`

### 步骤 4：过期邀请回归

手工构造过期通知（通过 DB 或 gorm script）：

```sql
-- 插入一条 Bob 收到的过期邀请通知（expired_at=1 天前）
INSERT INTO notifications (user_id, notify_type, title, content, ext, expired_at, created_at)
VALUES (56, 'meeting_invitation', '会议邀请',
        'Alice 邀请你加入会议 000-000-000',
        '{"room_code":"000-000-000","invite_token":"expired"}',
        UNIX_TIMESTAMP() - 86400, NOW());
```

1. Bob tab 刷新通知 tab
2. 期望：该卡片渲染为 **disabled 单按钮「邀请已过期」**
3. 尝试点击应无跳转
4. 截图 → `03-notify-expired-invite.png`

### 步骤 5：清理

1. SQL 删除步骤 4 插入的测试通知
2. 结束会议（若 Alice 仍在会中）

## 预期结果汇总

| 断言 | 预期 |
|---|---|
| 通知中心实时刷新 | 邀请送达后 3s 内出现新卡片 |
| 卡片双按钮 | 「立即加入」 + 「稍后」 |
| 「立即加入」跳转 | `preview?mode=join&code=xxx&invite_token=yyy` |
| 过期卡片 | 单按钮 disabled「邀请已过期」 |

## 失败排查

| 现象 | 定位 |
|---|---|
| Bob 通知 tab 空 | WS `notify.new` 事件未收到 / notify store 未订阅 |
| 「立即加入」点击无反应 | `components/notify/NotifyCard.vue` 按钮 handler |
| 过期卡片仍 clickable | NotifyCard 的 `isExpired` 判定逻辑 |
