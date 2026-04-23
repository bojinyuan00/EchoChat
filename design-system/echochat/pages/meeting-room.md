# Meeting Room (会议室主页面) Page Overrides

> **PROJECT:** EchoChat
> **Generated:** 2026-04-24 (via Task 15 UI 打磨)
> **Page Type:** Full Screen — Meeting Main View (Fixed Fullscreen Layer)
> **Route:** `/pages/meeting/room`

> ⚠️ Rules in this file **override** `design-system/echochat/MASTER.md`.
> ⚠️ 本页面是 EchoChat 会议模块原创视觉的集大成体现，包含 6 条原创特色：
> **流光轮廓 / 柔性网格 / 自视频浮窗吸附 / 静音氛围色 / 入会滑入 / 波浪网络徽标**。

---

## Page Layout

- **结构:** 顶部信息条（Title + Code + State pill + NetworkBadge + Timer） + 主视频区 + 聊天侧栏（可隐藏）+ 底部工具栏
- **Z-index 层级（严格遵守）:**
  - `room root`: 1000（高于 CustomTabBar）
  - `top-bar`: 5
  - `video-grid`: auto
  - `SelfVideoFloat`: 120
  - `MemberPanel / InviteDialog 遮罩`: 200
  - `MeetingToolbar`: 210
  - `Leave Confirm Modal`: 220

## 原创特色 1：说话者流光轮廓

**规范:**

```css
.tile.speaking {
  border: 2rpx solid transparent;
  background:
    linear-gradient(#0B1220, #0B1220) padding-box,
    conic-gradient(from var(--flow-angle, 0deg),
      #10B981, #2563EB, #6366F1, #10B981) border-box;
  animation: flow 3s linear infinite;
  box-shadow: 0 0 24rpx rgba(99, 102, 241, 0.45);
}

@property --flow-angle {
  syntax: '<angle>';
  initial-value: 0deg;
  inherits: false;
}
@keyframes flow { to { --flow-angle: 360deg; } }
```

- **触发:** `audioLevel >= 0.05`，1 in / 2 out 防抖（500ms polling）
- **Safari 17- fallback:** 静态蓝环 `border-color: #3B82F6 + box-shadow`（既有实现）

## 原创特色 2：柔性网格（2/3 人非等分）

| 成员数（不含本地浮窗） | 布局 | 说明 |
|---|---|---|
| 0 | 居中文案 "等待其他人加入…" | 只有自己时本地浮窗承担唯一画面 |
| 1 | 单块铺满 | 远端唯一成员占满视频区 |
| 2 | 左右 65/35 | 首位（加入时间早）为主画面 |
| 3 | 大块左侧 60% + 右侧 2 块上下叠放各 50% | 三角视觉焦点 |
| 4-9 | `repeat(3, 1fr)` 等分 | 保守兜底 |
| 10+ | `repeat(ceil(sqrt(n)), 1fr)` | 同现有 |

- **间隙:** `gap: 12rpx`（保持现有）
- **小屏（`max-width: 750px`）:** 回退到纵向等分，不启用非等分

## 原创特色 3：自视频浮窗四角吸附

**规范:**

- **默认位置:** 右下角，距视频区 `24rpx`
- **尺寸:**
  - 桌面端: `280px × 180px`
  - 移动端: `240rpx × 150rpx`（16:10）
- **视觉:** `border-radius: 16rpx`; 阴影 `0 12rpx 32rpx rgba(0,0,0,0.45)`; `1rpx solid rgba(255,255,255,0.1)` 微边
- **图钉按钮:** 右上角 `36rpx × 36rpx` 半透明圆形按钮，点击切换 float ↔ grid
- **拖拽:**
  - PC: `mousedown/mousemove/mouseup`
  - 移动: `touchstart/touchmove/touchend`
  - 拖拽中禁用 `user-select: none`，`cursor: grabbing`
- **吸附:**
  - 释放时计算距离 4 角最近者
  - `transition: transform 0.22s cubic-bezier(0.2, 0.8, 0.2, 1)`
- **Z-index:** 120

## 原创特色 4：静音氛围色

**规范:**

- 工具栏背景 `computed(allMuted)`：
  - 正常（默认或有人开麦）: `rgba(17, 24, 39, 0.88)`
  - 全员静音且 ≥ 2 人：`linear-gradient(to top, rgba(30, 58, 138, 0.88), rgba(30, 64, 175, 0.78))`（冷蓝）
- **过渡:** `transition: background 0.4s ease`
- **触发条件:** `tiles.length >= 2 && tiles.every(t => !t.audioEnabled)`

## 原创特色 5：入会滑入动效

- 新成员 `tile` 插入网格时使用 `@keyframes slideIn`:

```css
@keyframes slideIn {
  from { transform: translateY(16rpx); opacity: 0; }
  to   { transform: translateY(0);     opacity: 1; }
}
.grid-cell { animation: slideIn 0.28s cubic-bezier(0.2,0.8,0.2,1) both; }
```

- **注意:** 避免所有 tile 在页面首次挂载时全部跑动画；用 `v-for` key + `<transition-group>` 或者标志位 `isFirstLoad = false` 后才启用。

## 原创特色 6：NetworkBadge 3 条波浪

**规范:**

- 3 条 SVG 曲线 `<path d="M0 8 C 2 4, 4 12, 6 8 S 10 4, 12 8 S 16 12, 18 8" />`
- 宽度 `18px`，高 `16px`，`stroke-width: 2`，`fill: none`
- level → 可见条数 & 颜色：
  - `4` 优秀: 3 条 `#10B981`
  - `3` 良好: 2 条 `#22C55E`
  - `2` 一般: 1 条 `#F59E0B`
  - `1` 很差: 1 条短版 `#EF4444`
  - `0` 已断: 静态水平红线 + `!` 小图标
- **动效:** `@keyframes wave { from { transform: translateX(0); } to { transform: translateX(-6px); } }`，`1.2s linear infinite`，3 条相位差 `0s / -0.3s / -0.6s`

## Color Palette (页面专属)

| 用途 | Hex |
|------|-----|
| 房间背景 | `#0F172A` |
| 视频区背景 | `#0B1220` |
| 视频块背景 | `#0B1220` |
| 聊天侧栏 | `#1F2937` |
| Toolbar 默认 | `rgba(17,24,39,0.88)` |
| Toolbar 全员静音 | `linear-gradient(to top, rgba(30,58,138,0.88), rgba(30,64,175,0.78))` |
| 状态 pill 已连接 | `rgba(16,185,129,0.2)` + `#86EFAC` |
| 状态 pill 重连 | `rgba(245,158,11,0.25)` + `#FBBF24` |

## Motion Timeline

| 事件 | 动效 | 时长 |
|------|------|-----|
| 页面挂载 | Top-bar fade-in | 200ms |
| 新成员加入 | `tile slideIn` | 280ms |
| 开始说话 | 边框流光启动 | 持续 |
| 全员静音 | Toolbar 背景渐变 | 400ms |
| 浮窗释放吸附 | `transform` 缓动 | 220ms |

## Interactions

- **工具栏按钮:** 所有交互事件见 `MeetingToolbar.vue`；二次点击成员/聊天/邀请均 toggle 关闭
- **浮窗图钉:** 点击切换 float/grid 模式，`meetingStore.uiPrefs.selfFloat` 记忆
- **主持人:**
  - 成员面板右侧三点菜单展开"请他静音 / 请他开麦 / 转让主持人 / 踢出会议"
  - 四件套全部收敛在 MemberPanel 内，不在主界面暴露

## Accessibility

- 所有动效尊重 `@media (prefers-reduced-motion: reduce)`，禁用流光旋转与入会滑入，改为渐变态
