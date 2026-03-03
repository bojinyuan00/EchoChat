# Chat Conversation (聊天对话) Page Overrides

> **PROJECT:** EchoChat
> **Generated:** 2026-03-03 (via ui-ux-pro-max)
> **Page Type:** Full Screen — Chat Dialog (custom navigation)

> ⚠️ Rules in this file **override** `design-system/echochat/MASTER.md`.

---

## Page Layout

- **结构:** Custom NavBar + Scrollable Message List + Input Bar
- **背景:** `#F1F5F9` (区分于白色列表页)
- **导航栏:** 自定义，包含返回 + 标题/输入提示 + 更多设置

## Icon Mapping

| 位置 | uni-icons type | size | color |
|------|---------------|------|-------|
| 返回按钮 | `back` | 20 | `#1E293B` |
| 更多按钮 | `more-filled` | 20 | `#475569` |
| 发送按钮 | `paperplane` | 22 | `#FFFFFF` |
| 发送中状态 | `loop` | 16 | `#94A3B8` |
| 发送失败 | `info-filled` | 18 | `#EF4444` |

## Message Bubble Spec

### 自己的消息（右侧）

```css
.bubble-self {
  background-color: #2563EB;
  color: #FFFFFF;
  border-radius: 24rpx;
  border-bottom-right-radius: 8rpx;
  max-width: 560rpx;
}
```

### 对方的消息（左侧）

```css
.bubble-other {
  background-color: #FFFFFF;
  color: #1E293B;
  border-radius: 24rpx;
  border-bottom-left-radius: 8rpx;
  max-width: 560rpx;
}
```

### 已撤回消息

- 背景透明，灰色斜体文字 `#94A3B8`

## Input Bar

- **背景:** `#FFFFFF` + 顶部 `1rpx` 边框 `#E2E8F0`
- **输入框:** `#F1F5F9` 背景，`36rpx` 圆角
- **发送按钮:** 圆形 72rpx，默认 `#CBD5E1`，有内容时 `#2563EB`
- **安全区:** `padding-bottom: calc(16rpx + env(safe-area-inset-bottom))`

## Interactions

- **长按自己消息:** ActionSheet 撤回
- **点击失败图标:** Modal 确认重发
- **下拉到顶:** 加载更多历史消息
- **正在输入:** 导航栏标题下方蓝色小字提示
