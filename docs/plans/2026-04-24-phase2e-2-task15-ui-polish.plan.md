# Phase 2e-2 Task 15：UI 打磨 + 主持人权限四件套补齐

> **状态：** 📋 设计阶段（等待用户 Review）
> **上级实施计划：** [docs/plans/2026-04-21-phase2e-2-implementation.plan.md §Task 15](./2026-04-21-phase2e-2-implementation.plan.md)
> **设计依据：** [Phase 2e-2 设计文档 §9.1 / §9.2 / §3.3 D08](./2026-04-21-phase2e-2-design.md)
> **分支：** `feature/phase2e-2-meeting-mvp`
> **预估工时：** 1.5-2 人日
> **范围锁定：** 满配（6 条原创特色 + 静音他人 + 4 屏设计文档 + 桌面端恒浮窗 + RTP stats 双源说话者探测）

---

## 一、目标

把设计文档 §9.2 定义的 EchoChat 会议原创特色 **6 条** 全部落地到代码，补齐主持人权限四件套的最后一件（静音他人），同时把会议模块 4 个核心屏幕的「设计 SSOT」写进 `design-system/echochat/pages/meeting-*.md`，让设计产物与代码实现可持续对齐。

不在本期做：
- 不做分布式 SFU / simulcast 档位切换
- 不做 `meeting.room.update`（会议主题 / 入会静音开关在线修改）
- 不做加入时视频块从网格外滑入（候选 §9.2 第 5 条，已被确认归入本期但作为低优先级，若工时紧可削减）

---

## 二、改动清单

### 2.1 新建文件（5）

| 路径 | 作用 |
|---|---|
| `design-system/echochat/pages/meeting-home.md` | 会议 Hub 首页（即时会议入口 + 加入会议）视觉规范 |
| `design-system/echochat/pages/meeting-preview.md` | 设备预览页（大预览 + 右侧设备列表 + 音量波浪）视觉规范 |
| `design-system/echochat/pages/meeting-room.md` | 会议室主页面（浮窗 / 流光 / 静音氛围色 / 柔性网格 / 波浪网络徽标）视觉规范 |
| `design-system/echochat/pages/meeting-invite.md` | 邀请弹窗（复制链接 / 会议号 / 联系人选择器）视觉规范 |
| `frontend/src/components/meeting/SelfVideoFloat.vue` | 自视频浮窗（桌面端恒浮窗 + 四角吸附拖拽 + 图钉切换回网格）|

### 2.2 改动文件（7）

| 路径 | 改动点 |
|---|---|
| `frontend/src/components/meeting/VideoTile.vue` | 流光轮廓 CSS `conic-gradient` + `@keyframes` 动效；`speaking` class 由上层喂入 |
| `frontend/src/components/meeting/VideoGrid.vue` | 2 人左右 65/35 非等分；3 人大/小/小三角布局；4+ 人维持等分；剔除本地 tile（由浮窗承接）|
| `frontend/src/components/meeting/MeetingToolbar.vue` | 新增 `:all-muted` prop，背景色绑定冷蓝渐变（`all-muted=true` 时）|
| `frontend/src/components/meeting/MemberPanel.vue` | 操作菜单新增「请他静音 / 请他开麦」（仅 host + 非自己 + 当前音频开/关）|
| `frontend/src/components/meeting/NetworkBadge.vue` | 4 条竖柱 → 3 条 SVG 波浪线 + 相位差动画；`level` prop 语义保留 |
| `frontend/src/store/meeting.js` | 新增 `muteMember(uid, mute)` action + `speakingMap` state；新增 `_startSpeakingDetection` / `_stopSpeakingDetection` 内部方法；`isAllMuted` getter |
| `frontend/src/pages/meeting/room.vue` | 组装 `SelfVideoFloat` + `all-muted` 绑定 + speaking 下发到 tile；离会弹窗未改 |

**动效性能目标：** 所有 `@keyframes` 限制在 `transform` / `opacity` / `border-image` / `background` 属性上，不触发 layout；Chrome Performance 面板滚动会议室 10 秒，保持 60fps、无 long task > 50ms。

---

## 三、关键技术点

### 3.1 说话者流光轮廓

- CSS 实现：`border: 2rpx solid transparent` + `background-image: conic-gradient(from ...)` 切换为 `linear-gradient(#1e293b,#1e293b) padding-box, conic-gradient(...) border-box`；`@keyframes flow` 旋转 `--flow-angle`（`@property --flow-angle` 声明，不触发 layout）。
- 触发：`VideoTile.vue` 接收 `isSpeaking` prop，上层 `room.vue` 通过 `store.speakingMap[userId]` 投影。
- 兼容：Safari 不支持 `@property` 时 fallback 为静态蓝色环形边（现有 `.speaking` 样式）。

### 3.2 说话者探测（RTP stats + WebAudio 双源）

- **远端**：`store._startSpeakingDetection()` 每 500ms 遍历 `remoteConsumers[uid].audio.rtpReceiver`，调用 `getSynchronizationSources()`（标准 W3C API，Chrome/Firefox/Edge 支持），读 `audioLevel`（0-1 线性）；阈值 0.05 以上且持续 ≥ 1 个 tick 判定为 `speaking=true`，< 阈值连续 2 个 tick 为 `speaking=false`（防抖）。
- **本地**：`localProducers.audio` 拿不到 `rtpReceiver`（因为它是 sender）。本地用 `AudioContext` + `MediaStreamAudioSourceNode` + `AnalyserNode.getFloatTimeDomainData` 算 RMS；音量门限归一化到同一 0-1 区间。
- **资源释放**：`leave()` / `stopLocalAudio()` / `_clearRemoteConsumer()` 必须调 `_stopSpeakingDetection()`；轮询用 `setInterval` + Pinia 内持有 handle，避免泄漏。
- **H5 Only**：整段逻辑包在 `#ifdef H5`，非 H5 平台 `speakingMap` 空对象，UI 兜底为 never speaking。

### 3.3 柔性网格

- `VideoGrid.vue` 按 `tiles.length` 选择 layout class：
  - `layout-1`：全屏单块
  - `layout-2`：左右 65/35，横向 flex（说话者自动成为大块，Task 16 做；本期先固定"首个 tile 为大块"）
  - `layout-3`：左上大块 65%，右侧两个小块各 50% 高度叠放
  - `layout-4+`：继续 `repeat(n, 1fr)` 等分
- 小屏（`<= 750px`）统一回退到纵向等分（现有逻辑保留）
- 本地 tile 由 `SelfVideoFloat` 渲染，`VideoGrid` 只展示 `tiles.filter(t => !t.isLocal)`（N-1 个远端）。所以实际 layout 判断基于 `tiles.length - 1`。

### 3.4 自视频浮窗（SelfVideoFloat.vue）

- 尺寸：默认 `240rpx × 150rpx`（16:10），桌面端 `280×180px`；圆角 `16rpx` + 阴影 `0 12rpx 32rpx rgba(0,0,0,0.45)`
- 位置：默认右下角，距视频区边缘 `24rpx`
- 拖拽：`@mousedown` + `@touchstart` 注册 `mousemove/touchmove` 全局监听；释放时计算 4 角距离，`transform: translate(x,y)` 带 `transition: 0.22s cubic-bezier(0.2,0.8,0.2,1)` 吸附
- 图钉按钮：右上角 pin 图标，点击切换 `floatMode` ↔ `gridMode`；gridMode 时浮窗隐藏、本地 tile 回到 Grid 里
- 持久化：`floatMode` 存入 `meetingStore.uiPrefs`（Pinia 内存，不落盘；下次进会议默认浮窗）
- Z-index：浮窗 120，低于 toolbar(210) 低于 MemberPanel(200)，高于 VideoGrid

### 3.5 静音氛围色

- `room.vue` 新增 `allMuted = computed(() => tiles.length >= 2 && tiles.every(t => !t.audioEnabled))`
- `MeetingToolbar` 接收 `:all-muted`，`style: background` 绑定：
  - 默认：`rgba(17, 24, 39, 0.88)`（保持现有）
  - 全员静音：`linear-gradient(to top, rgba(30, 58, 138, 0.88), rgba(30, 64, 175, 0.78))`（冷蓝）
  - 过渡：`transition: background 0.4s ease`
- 单人情况（只有自己）不触发，避免独自加入就冷色

### 3.6 NetworkBadge 3 条波浪

- 改为 3 个 SVG `<path d="M 0 8 C 2 4, 4 12, 6 8 S 10 4, 12 8 S 16 12, 18 8" />` 波浪曲线
- 动画：`@keyframes wave { 0% { transform: translateX(0) } 100% { transform: translateX(-6px) } }`；3 条相位差 0 / -0.3s / -0.6s
- level → 可见条数：4=3条 / 3=2条 / 2=1条 / 1=1条短 / 0=静态红色中划线
- 颜色映射保持（`#10B981` / `#22C55E` / `#F59E0B` / `#EF4444`）

### 3.7 静音他人（主持人权限第 4 件）

- 后端能力已就绪：`meeting.member.state.changed` WS 事件带 `target_user_id` 参数，host 身份校验在 `MeetingSignalService.OnMemberStateChanged` 已实现
- 前端 store：
  ```js
  async muteMember(uid, mute = true) {
    if (!this.isHost) throw new Error('仅主持人可操作')
    const payload = { target_user_id: uid, audio_enabled: !mute }
    await wsService.sendWithAck('meeting.member.state.changed', payload, 3000)
  }
  ```
- `MemberPanel` 操作菜单：
  - host 视角下每行尾部三点菜单新增 2 项：
    - 若 `p.audio_enabled === true`：「请他静音」（灰色，非 danger）
    - 若 `p.audio_enabled === false`：「请他开麦」（灰色）
  - 现有「转让主持人 / 踢出会议」保留
- 被静音侧 UX：后端 `meeting.member.state.changed` 广播回来时，store `_onMemberStateChanged` 已识别 `target_user_id === 我自己`，直接调 `stopLocalAudio()` 停本地 producer；本期补一个 toast 提示「主持人请你静音」

---

## 四、任务拆分

| # | 任务 | 工时 | 产出 |
|---|---|---|---|
| T15.1 | 4 屏 design-system 文档 | 0.3 人日 | `design-system/echochat/pages/meeting-{home,preview,room,invite}.md` |
| T15.2 | VideoTile 流光轮廓 + CSS @property fallback | 0.15 人日 | VideoTile.vue 改 |
| T15.3 | 说话者探测双源（store + 资源释放 + 防抖） | 0.3 人日 | `store/meeting.js` + VideoTile 接线 |
| T15.4 | VideoGrid 柔性网格 2/3 人布局 | 0.15 人日 | VideoGrid.vue 改 |
| T15.5 | SelfVideoFloat.vue 浮窗 + 拖拽 + 吸附 + 图钉 | 0.3 人日 | 新文件 + room.vue 接入 |
| T15.6 | MeetingToolbar 静音氛围色 + allMuted 绑定 | 0.1 人日 | Toolbar + room.vue |
| T15.7 | NetworkBadge 3 条波浪 SVG + level 映射 | 0.15 人日 | NetworkBadge.vue 重写 |
| T15.8 | 静音他人：store action + MemberPanel 菜单 + 被静音 toast | 0.2 人日 | store / MemberPanel / room.vue |
| T15.9 | 本机 Playwright 截图回归（4 屏 + 流光 + 浮窗吸附 + 静音氛围） | 0.15 人日 | 截图归档 |
| T15.10 | 进度文档同步 + commit + push | 0.1 人日 | CURRENT_STATUS / implementation.plan / project-context |
| **合计** | | **1.9 人日** | |

---

## 五、检查点（对齐设计文档 §9 + 实施计划 §Task 15 验收）

- [ ] 4 屏 `design-system/echochat/pages/meeting-*.md` 全部落盘，包含 Layout / Color / Typography / Motion 四栏
- [ ] 说话者流光轮廓在双人会议中稳定触发，阈值 `audioLevel >= 0.05`，防抖 1 in / 2 out
- [ ] 2 人 65/35 / 3 人三角布局视觉与设计一致；4+ 人保持等分
- [ ] 自视频浮窗拖拽松手后 0.22s 内吸附到最近四角；桌面端 `Cmd+Drag` 不触发系统 drag
- [ ] 图钉按钮可切换 float ↔ grid，grid 模式本地 tile 回到网格里
- [ ] 全员静音时 Toolbar 冷蓝渐变，恢复说话立即回暖；单人独处不触发
- [ ] NetworkBadge 3 条波浪流动，level 4/3/2/1/0 对应可见条数 3/2/1/1(短)/0
- [ ] 主持人菜单显示「请他静音 / 请他开麦」，非主持人不可见；被请对方收到 toast 并本地静音
- [ ] Chrome Performance 面板会议室滚动 10 秒 60fps，无 long task > 50ms
- [ ] `npm run build:h5` 无新增警告；`ReadLints` 零错误
- [ ] Playwright 截图 4 张归档到 `.playwright-mcp/task15/`（home / preview / room / invite）

---

## 六、风险

| 风险 | 严重度 | 缓解 |
|---|---|---|
| `@property --flow-angle` Safari 17- 不支持 | 低 | CSS `@supports` + fallback 静态蓝环 |
| `getSynchronizationSources()` 在 Firefox 返回空数组 | 中 | `if (sources.length === 0)` 回退到 WebAudio |
| 自视频浮窗 `mousemove` 全局监听内存泄漏 | 中 | `useEventListener` 模式 + `onBeforeUnmount` 显式 removeListener |
| 静音氛围色在"全员非 host 静音但 host 未静音"时也触发，可能误伤 | 低 | 条件用 `tiles.length >= 2 && tiles.every(t => !t.audioEnabled)`，host 未静音时不成立 |
| 被主持人静音的用户以为"自己麦克风故障" | 中 | 收到静音指令后显示 `uni.showToast({ title: '主持人请你静音' })` 持续 2s |

---

## 七、变更记录

| 日期 | 变更 |
|---|---|
| 2026-04-24 | 初稿 |
