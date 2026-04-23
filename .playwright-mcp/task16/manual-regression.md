# Task 16 手动回归点清单

> Playwright MCP 难以稳定覆盖的交互场景，由开发者手工在 HTTPS 环境过一遍并记录结论。
> 状态列：✅ 通过 / ❌ 失败 / ⏳ 待跑 / ➖ 不适用

| # | 场景 | 操作步骤 | 期望 | 状态 | 备注 |
|---|---|---|---|---|---|
| M1 | HTTPS 自签证书首次允许 | Chromium 打开 `https://localhost:5173/` → 点「高级」→「继续访问」→ 依次放行摄像头/麦克风 | uni-app 首页正常渲染；控制台无 Mixed Content 告警 | ⏳ | Playwright 多 context 需预注入证书信任 |
| M2 | iOS Safari 真机跨网段入会 | 手机浏览器（Safari / 微信内置）访问 `https://<局域网 IP>:5173` → 扫码加入会议 | 音视频双向打通；`NetworkBadge` 显示 3 条绿波浪 | ⏳ | iOS Safari `srcObject` 行为与 Chromium 存在差异 |
| M3 | 网络抖动：WS 断开 20s 后恢复 | 会议中开发者工具切到 Offline，等 20s 后恢复 Online | 无声无画面约 5s 后自动 resume；`remoteConsumers` 索引无漂移；不重复出现「其他用户」 | ⏳ | 关联 Task 14 `reconnect` 路径 |
| M4 | 网络抖动：WS 断开 > 2 分钟 | Offline 维持 130s 后 Online | 被服务端视为宽限到期，Alice（主持人）转让给 Bob；本地 UI 跳回首页 | ⏳ | 场景 04 路径 B 的手工版 |
| M5 | 物理返回按钮离会 | 会议进行中按浏览器 / 手机物理返回 | `onUnload` 触发 `leave()` → REST `leaveRoom` 成功 → 本地资源释放 | ⏳ | 关注 `pages/meeting/room.vue` onUnload |
| M6 | 结束页停留久后返回 | 主持人结束 → 停在「会议已结束」提示页 5 分钟 → 再点返回 | `exitEndedRoom` 触发，pinia state 重置；再次入会无残留 | ⏳ | **Step A F1 核心回归** |
| M7 | 跨 tab 并发入同一会议 | Alice tab 入会后复制 URL 到新 tab 再次打开 | 新 tab 命中"已在此会议中"保护；点提示按 `cleanupStaleMeetings` 自动清理后重试入会 | ⏳ | 关联 Task 11 `ErrAlreadyInMeeting` |
| M8 | 摄像头被系统抢占 | 入会后用其他应用抢占摄像头（如 FaceTime） | UI 显示占位封面 + 「摄像头被占用」toast；会议不崩溃；释放后可重新开摄像头 | ⏳ | `NotReadableError` 分支 |
| M9 | 浏览器关闭/崩溃恢复 | 入会后直接 kill 浏览器进程，重启后重新登录并入同会 | 残留 participant 通过 `cleanupStaleMeetings` + 自动重试加入恢复 | ⏳ | 与 M7 互补 |
| M10 | Chat 面板 - 长消息发送 | 在会议中聊天面板发送 5000 字消息 | 服务端拒绝 `msg too long`（期望 P2 新增限制）或前端提前拦截 | ⏳ | 对应 P2-7 |
| M11 | 邀请 token 超时但会议仍存在 | 手工 SQL 将 `meeting_invites.expired_at` 改为过去时间 → 访问邀请链接 | 预览页显示「邀请已过期」，不允许加入；点「重新申请」可弹窗 | ⏳ | 对应 §6.4 |
| M12 | 主持人关闭摄像头但麦克风保留 | 会议中点视频开关 → 仅摄像头关闭 | 自己画面占位 + 工具栏视频按钮变灰；音频仍正常；其他成员成员面板图标同步 | ⏳ | 对应 Task 15 状态 |
| M13 | 静音氛围色 | Alice + Bob + Carol 全员关麦 | 工具栏背景切蓝色渐变；有任何一人开麦则复位 | ⏳ | Task 15 原创特色 4 |
| M14 | 说话者流光轮廓 | Bob 说话 ≥0.5s | Bob 视频 tile 出现流光边框；停 1s 后消失 | ⏳ | Task 15 原创特色 1 |
| M15 | 图钉 / 浮窗切换 | 桌面端点自视频浮窗的图钉图标 | 自视频从浮窗切回 2×2 宫格格子；再点一次返回浮窗 | ⏳ | Task 15 原创特色 3 |
| M16 | 无网情况下直接访问会议 | 断网后访问会议首页 | 友好错误提示；Retry 可恢复 | ⏳ | 通用容错 |
| M17 | 高延迟网络（150ms+ + 1% 丢包） | DevTools Throttling 选 `Slow 3G`；入会并说话 | 音视频可连通（降级）；不崩溃；`NetworkBadge` 波浪数降为 1-2 | ⏳ | 生产近似网络 |

## 手工验证数据记录模板

每次跑完某一条，在 `docs/reports/test-report-phase2e-2-meeting.md` 填入：

```md
### M<编号> 验证记录

- 执行时间：2026-04-xx HH:MM
- 环境：Chromium 126 / macOS 25.3.0 / HTTPS 自签
- 结论：✅/❌
- 截图证据：`.playwright-mcp/task16/manual/M<编号>.png`
- 异常描述（若 ❌）：...
- 修复动作：...
```
