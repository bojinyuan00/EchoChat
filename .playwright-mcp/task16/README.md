# Task 16 Playwright MCP E2E 剧本

> 生成时间：2026-04-23（UTC+8）
> 回归范围：Phase 2e-2 Task 16 —— E2E 总回归 + 资源清理专项
> 环境：macOS 25.3.0 / Chromium（Playwright MCP）/ 1440×900 桌面视口
> 三件套：`go-service :8085`、`media-server :3300`、`frontend :5173`（HTTPS 自签）

## 场景矩阵

| # | 文件 | 覆盖设计文档章节 | 验证目标 | 资源断言 |
|---|---|---|---|---|
| 01 | [`scenarios/01-create-and-join-by-code.md`](scenarios/01-create-and-join-by-code.md) | §6.1 / §6.3 | 创建即时会议 → 会议号加入 → 双向音视频打通 → 主持人结束 | Router/Transport/Producer/Consumer 全量创建 + CloseRouter 级联释放 |
| 02 | [`scenarios/02-invite-link-with-password.md`](scenarios/02-invite-link-with-password.md) | §6.2 / §6.4 | 邀请链接带密码 → 受邀人免密加入 | `meeting_invites` 表 `consumed_at` 更新；密码不出现在 URL query |
| 03 | [`scenarios/03-notify-quickjoin.md`](scenarios/03-notify-quickjoin.md) | §6.4 | 通知中心「立即加入」按钮跳转 | `meeting_invitation` 通知类型 + `expired_at` 边界 |
| 04 | [`scenarios/04-host-toolkit-and-transfer.md`](scenarios/04-host-toolkit-and-transfer.md) | §6.5 / §6.6 | 主持人四件套（请静音/开麦/转让/踢出）+ host 宽限期自动转让 | `meeting_participants.role` 原子转让；`host_grace` Redis key + 处理锁 SETNX |
| 05 | [`scenarios/05-reopen-5times-resource-cleanup.md`](scenarios/05-reopen-5times-resource-cleanup.md) | §5.4 / §资源清理专项 | **反复开关 5 次会议，验证资源无泄漏** | `echo:meeting:resource:*` / `echo:meeting:member_state:*` / `echo:meeting:host_grace:*` / `echo:meeting:empty_ttl:*` 全部不残留；前端 `_engine` / `AudioContext` / `setInterval` 全部释放 |

## 测试账号

沿用 Task 15 账号体系（固定 `user_id` 便于断言）：

| 账号 | 密码 | user_id | 角色 |
|---|---|---:|---|
| `task15_a` | `Task15@2026` | 55 | 主持人 Alice |
| `task15_b` | `Task15@2026` | 56 | 参会人 Bob |
| `task15_c` | `Task15@2026` | 57 | 参会人 Carol（场景 04 需要第三人验证 host 宽限期转让目标） |

> 账号若不存在可用 `POST /api/v1/auth/register` 脚本化创建；密码策略要求大写 + 特殊字符。

## 通用前置条件

1. **三件套已就绪**：
   ```bash
   # backend/go-service
   go run cmd/allinone/main.go
   # media-server
   cd media-server && npm run dev
   # frontend
   cd frontend && npm run dev:h5:https
   ```
2. **DB/Redis 干净**：
   ```bash
   # MySQL：确认 meeting_rooms / meeting_participants / meeting_invites 无遗留 active 会话
   mysql -e "UPDATE meeting_rooms SET status='ended' WHERE status='active'" echochat
   # Redis：清掉可能的残留
   redis-cli --scan --pattern 'echo:meeting:*' | xargs -r redis-cli DEL
   ```
3. **浏览器**：Chromium 允许 `localhost` 自签证书（`chrome://flags/#allow-insecure-localhost`），并授予 `camera` / `microphone` 权限。
4. **控制台期望**：无 `RTCPeerConnection error` / `NotReadableError` / `ResourceNotOwned`。

## 运行方式

每个剧本文件的 **"MCP 执行脚本"** 段落直接复制给 Playwright MCP Agent（Cursor Chat 中 `Use playwright-mcp` 触发）：

```text
请按照 @.playwright-mcp/task16/scenarios/01-create-and-join-by-code.md 的 "MCP 执行脚本" 章节逐步执行并逐条断言。
```

Agent 会用 `browser_navigate` / `browser_click` / `browser_fill_form` / `browser_evaluate` 完成剧本，每步结果以截图 + 控制台日志 + Pinia state 快照形式落盘到 `.playwright-mcp/task16/<scene>/`。

## 资源断言脚本

每个场景结束后需要运行的资源检查（用于证明 Step A 的资源清理修复生效）：

```bash
# Redis：会议期间 key 清单（应仅含活跃会议，结束后应清零）
redis-cli KEYS 'echo:meeting:resource:*'
redis-cli KEYS 'echo:meeting:member_state:*'
redis-cli KEYS 'echo:meeting:host_grace:*'
redis-cli KEYS 'echo:meeting:empty_ttl:*'

# MySQL：所有 active 会议
mysql -e "SELECT id, room_code, status, host_id FROM meeting_rooms WHERE status='active'" echochat

# media-server 观测端点：Router / Transport / Producer / Consumer 汇总
curl -s -H "X-Internal-Token: $INTERNAL_TOKEN" http://localhost:3300/internal/info | jq .stats
```

## 产出物

每个场景执行后需要落盘：

- `scenarios/XX-*/screenshots/`：关键 UI 截图（入会前、打通后、结束后）
- `scenarios/XX-*/console.log`：浏览器控制台归档
- `scenarios/XX-*/result.md`：逐条断言结果（✅/❌ + 证据链接）

所有 5 场景跑完后在 `docs/reports/test-report-phase2e-2-meeting.md` 汇总。

## 手动回归点

Playwright MCP 难以覆盖的交互场景归档在 [`manual-regression.md`](manual-regression.md)，由开发者手工过一遍并记录结论。
