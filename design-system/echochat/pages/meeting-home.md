# Meeting Home (会议 Hub 首页) Page Overrides

> **PROJECT:** EchoChat
> **Generated:** 2026-04-24 (via Task 15 UI 打磨)
> **Page Type:** Tab Page — Meeting Entrance + Quick Actions
> **Route:** `/pages/meeting/home` / `/pages/meeting/join`

> ⚠️ Rules in this file **override** `design-system/echochat/MASTER.md`.
> ⚠️ 会议相关页面整体采用**深色主题**（`#0F172A` 背景），与聊天模块浅色主题形成品牌视觉分层。

---

## Page Layout

- **结构:** 顶部品牌区 + 主操作卡片（创建 / 加入二选一）+ 底部"最近会议"列表
- **背景:** `#0F172A` 纯色 + 顶部柔和蓝紫径向光晕（参考系统主色 `#2563EB`）
- **状态栏:** 透明穿透，内容顶部留 `env(safe-area-inset-top)` 间距

### 主操作卡片（2 卡片横向）

| 卡片 | 主图标 | 背景 | 文案 |
|------|-------|------|------|
| 创建会议 | `video` svg 28px white | `linear-gradient(135deg, #2563EB, #6366F1)` | 主标题 "发起会议" + 副标题 "创建一个即时会议" |
| 加入会议 | `log-in` svg 28px white | `rgba(255,255,255,0.06)` + `1rpx solid rgba(255,255,255,0.12)` | 主标题 "加入会议" + 副标题 "输入会议号或粘贴链接" |

- **尺寸:** 卡片高 `240rpx`，圆角 `24rpx`，阴影 `0 8rpx 24rpx rgba(37, 99, 235, 0.25)`（仅创建卡）
- **悬停:** `transform: translateY(-4rpx)` + 阴影加深
- **点击反馈:** `transform: scale(0.98)` `.12s`

### 最近会议列表

- **背景:** `rgba(255, 255, 255, 0.04)` + `1rpx solid rgba(255,255,255,0.08)` 边框
- **列表项:** 时间（左对齐 `#94A3B8 22rpx`）+ 标题（`#F3F4F6 28rpx`）+ 时长徽标（右侧 `#10B981` 小标签）
- **空状态:** 居中 icon `clock` + 灰字 "暂无最近会议"

## Color Palette (页面专属)

| 用途 | Hex | 说明 |
|------|-----|------|
| 主背景 | `#0F172A` | Slate 900 |
| 卡片面 | `rgba(255,255,255,0.06)` | Glass 效果 |
| 主 CTA 渐变 | `#2563EB → #6366F1` | 创建会议主按钮 |
| 辅 CTA | `rgba(255,255,255,0.12)` | 加入会议按钮边框 |
| 正文 | `#F3F4F6` | Gray 100 |
| 辅助文字 | `#94A3B8` | Slate 400 |
| 成功点缀 | `#10B981` | 在线 / 时长 |

## Typography

| 层级 | Size | Weight | Color |
|------|------|--------|-------|
| 品牌标题 | `44rpx` | 700 | `#F8FAFC` |
| 卡片主标题 | `32rpx` | 600 | `#FFFFFF` |
| 卡片副标题 | `24rpx` | 400 | `rgba(255,255,255,0.75)` |
| 列表时间戳 | `22rpx` | 400 | `#94A3B8` |

## Motion

- **入场:** 两张主卡片依次淡入 + 向上 `16rpx` 位移，`stagger: 80ms`
- **CTA 呼吸:** 创建卡在空闲 3s 后阴影脉动（`opacity 0.25 → 0.4 → 0.25`，`2.4s ease-in-out infinite`）
- **点击触达:** 任何 CTA 点击后转场用 `slide-up` `.28s cubic-bezier(0.2,0.8,0.2,1)`

## Interactions

- **创建会议:** `navigateTo('/pages/meeting/preview?mode=create')`
- **加入会议:** `navigateTo('/pages/meeting/join')`
- **最近会议项点击:** 直接 `navigateTo('/pages/meeting/preview?code=xxx&mode=join')`
- **长按列表项（暂不做）:** Task 16 拓展
