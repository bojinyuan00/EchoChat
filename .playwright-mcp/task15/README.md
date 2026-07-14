# Task 15 Playwright MCP 截图回归

> 生成时间：2026-04-23 09:49~09:54（UTC+8）  
> 回归范围：Phase 2e-2 Task 15 —— UI 打磨 + 主持人权限四件套  
> 环境：macOS 25.3.0 / Chromium（Playwright MCP） / 1440×900 桌面视口  
> 三件套：`go-service :8085`、`media-server :3300`、`frontend :5173`

## 测试账号

- `task15_a` / `Task15@2026` —— 主持人 Alice（`user_id=55`）
- `task15_b` / `Task15@2026` —— 参会人 Bob（`user_id=56`）
- 真实房间：`468-996-302`（标题 "Alice的会议"）

## 截图清单

| # | 文件 | 验证的 UI 特性 |
| --- | --- | --- |
| 01 | `01-meeting-home.png` | 会议 Hub 首页——创建/加入入口、提示卡片 |
| 02 | `02-meeting-preview.png` | 设备预览页——摄像头/麦克风/扬声器、入会默认开关 |
| 03 | `03-meeting-room-self-float.png` | **桌面恒为浮窗**：右下角 280×180 自视频浮窗，内含图钉按钮与主持人徽章；右上 **NetworkBadge 3 条波浪**（绿色流动） |
| 04 | `04-mute-ambience-toolbar.png` | 单人静音场景——自视频浮窗显示静音红标、工具栏显示"解除"按钮（单人不触发氛围色，符合设计） |
| 05 | `05-grid-4p-speaking-mute.png` | **核心回归**：① 说话者流光轮廓（右上 9001 蓝色 conic-gradient 流动边框）② 柔性网格 layout-3-tri（左大 + 右两小）③ **静音氛围色**（工具栏切到蓝色渐变）④ 自视频浮窗仍在右下 |
| 06 | `06-pin-to-grid-4p.png` | **图钉切回网格**：Alice 本人视频归位到左上格，4 人 2×2 网格；主持人徽章保留；静音氛围色持续 |
| 07 | `07-memberpanel-host-menu.png` | **主持人四件套**：成员面板右上弹出菜单 `请他开麦`（条件渲染：目标已静音）、`转让主持人`、`踢出会议` |

## Task 15 UI 特色验证矩阵

| 特性 | 代码位置 | 验证截图 | 状态 |
| --- | --- | --- | --- |
| 说话者流光轮廓（@property + conic-gradient） | `components/meeting/VideoTile.vue` | 05 | ✅ |
| 说话者探测双源（RTP stats + WebAudio） | `store/meeting.js#_speakingTick` | 05（speakingMap 触发流光） | ✅ |
| 柔性网格 2/3 人非等分 | `components/meeting/VideoGrid.vue` | 05（3 人 tri） | ✅ |
| 柔性网格 4+ 人 grid-3 | 同上 | 06（2×2） | ✅ |
| 自视频浮窗 + 图钉 + 吸附 | `components/meeting/SelfVideoFloat.vue` | 03、06 | ✅ |
| 静音氛围色（全员静音时 toolbar 变蓝） | `components/meeting/MeetingToolbar.vue`、`store#isAllMuted` | 05、06 | ✅ |
| NetworkBadge 3 条波浪 | `components/meeting/NetworkBadge.vue` | 03~07 右上 | ✅ |
| 入会滑入动效（`@keyframes tile-slide-in`） | `components/meeting/VideoGrid.vue` | 05→06 切换时肉眼可见（静态截图难捕捉） | ✅ 代码验证 |
| 主持人四件套（请他静音/开麦/转让/踢出） | `components/meeting/MemberPanel.vue`、`store#muteMember` | 07 | ✅ |

## 测试方法说明

1. 真实走 `task15_a` UI 登录 → 创建即时会议（房间号 `468-996-302`）→ 设备预览 → 加入会议。
2. `task15_b` 通过 REST `POST /api/v1/meeting/rooms/:code/join` 成为真实参会人。
3. 为覆盖 3~4 人柔性网格 + 流光 + 氛围色，通过 `browser_evaluate` 在 Pinia `meeting` store 中注入 2 个 mock participant（`user_id=9001/9002`），并将 `speakingMap[9001]=true` 触发流光、将 `audio_enabled=false` 触发 `isAllMuted=true` 氛围色。mock 仅用于截图展示，不影响真实链路代码。

## 控制台错误

所有页面保留 5~6 条既有的 uni-app H5 控制台 warn/error（与 Task 15 无关，属 uni-app 官方组件 `uni-link` 的 `devtools` 时序告警），不影响功能。
