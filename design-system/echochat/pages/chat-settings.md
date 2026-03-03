# Chat Settings (聊天设置) Page Overrides

> **PROJECT:** EchoChat
> **Generated:** 2026-03-03 (via ui-ux-pro-max)
> **Page Type:** Detail — Conversation Settings

> ⚠️ Rules in this file **override** `design-system/echochat/MASTER.md`.

---

## Page Layout

- **结构:** User Card + Settings Groups
- **背景:** `#F1F5F9`

## User Card

- **头像:** 120rpx 居中，圆角 30rpx
- **昵称:** `34rpx` `font-weight: 600` `#1E293B`
- **卡片背景:** `#FFFFFF`，底部 `16rpx` 间距

## Settings Groups

- **组背景:** `#FFFFFF`，组间 `16rpx` 间距
- **行高:** `padding: 32rpx`，底部 `1rpx` 分隔 `#F1F5F9`
- **普通项:** 文字 `#1E293B` 30rpx + 右侧值 `#94A3B8` 26rpx
- **危险项:** 文字 `#EF4444` 30rpx
- **按压态:** `:active { background-color: #F8FAFC }`

## Operations

| 操作 | 类型 | 确认方式 |
|------|------|---------|
| 置顶/取消置顶 | 普通项 | 直接执行 + Toast |
| 清空聊天记录 | 危险项 | Modal 二次确认 |
| 删除会话 | 危险项 | Modal 二次确认 → navigateBack(delta: 2) |
