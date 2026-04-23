# Meeting Preview (设备预览页) Page Overrides

> **PROJECT:** EchoChat
> **Generated:** 2026-04-24 (via Task 15 UI 打磨)
> **Page Type:** Full Screen — Pre-join Device Check
> **Route:** `/pages/meeting/preview`

> ⚠️ Rules in this file **override** `design-system/echochat/MASTER.md`.

---

## Page Layout

- **结构:** 左侧大预览区（≥ 60% 宽） + 右侧设备列表 + 底部入会操作条
- **背景:** `#0F172A`（主区）+ `#111827`（右侧设备面板）
- **顶部导航:** 透明，仅左侧返回图标 `#F3F4F6` 24px

### 左侧视频预览

- **比例:** 16:9，最大宽度 `960px`，圆角 `16rpx`，阴影 `0 12rpx 48rpx rgba(0,0,0,0.5)`
- **视频关闭态:** 深灰占位 `#1F2937` + 用户头像（`140rpx` 圆形） + 昵称下方
- **音量条覆盖层（创新点）:**
  - **不是分段格子**，而是**渐变波浪条**横向贯穿预览底部
  - CSS: `height: 6rpx`，`linear-gradient(90deg, #10B981 0%, #F59E0B 100%)`
  - 音量 0-1 映射为宽度 `0-100%`，`transition: width 0.05s linear`
  - 实时刷新（约 16fps 节流）

### 右侧设备面板（宽 `420rpx`）

- **分组:** 摄像头 / 麦克风 / 扬声器 三段
- **标题:** `24rpx` 600 `#94A3B8` uppercase
- **下拉框:** 高 `72rpx`，`#1F2937` 背景，`12rpx` 圆角，右侧箭头 `chevron-down`
- **入会设置（折叠）:** "麦克风静音入会" + "摄像头关闭入会" 两个 toggle

### 底部操作条

- **高度:** `120rpx` + safe area
- **左侧:** 麦克风 toggle + 摄像头 toggle（圆形 `80rpx`，开=蓝 `#2563EB` 关=灰 `#374151`）
- **右侧:** 主 CTA "进入会议" `160rpx × 72rpx`，`#2563EB → #6366F1` 渐变

## Color Palette (页面专属)

| 用途 | Hex |
|------|-----|
| 预览区背景 | `#0F172A` |
| 设备面板背景 | `#111827` |
| 下拉框面 | `#1F2937` |
| 音量条渐变 | `#10B981 → #F59E0B` |
| 主 CTA | `#2563EB → #6366F1` |
| 禁用态 | `#374151` |

## Motion

- **入场:** 视频预览先于右侧面板 `80ms` 出现
- **设备切换:** 下拉选中后预览区视频有 `200ms` 淡出 + `200ms` 淡入的 crossfade
- **音量波浪:** 超过 0.7（大声说话）时条形底部再叠一层 `0 0 16rpx #F59E0B` glow，脱离后 `300ms` 退场

## Interactions

- **下拉切换设备:** 立即释放旧 track，重新 `getUserMedia`（带 Loading 50ms 超时后再渲染）
- **麦/摄 toggle:** 仅切换本地 preview track 启用态，不持久化设备选择
- **入会设置 toggle:** 保存到 `meetingStore.devicePreview.enterMuted` / `enterCamOff`
- **点击进入会议:** 禁用状态 = 设备权限未授予时灰显
