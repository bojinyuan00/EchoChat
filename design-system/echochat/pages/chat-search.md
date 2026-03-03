# Chat Search (消息搜索) Page Overrides

> **PROJECT:** EchoChat
> **Generated:** 2026-03-03 (via ui-ux-pro-max)
> **Page Type:** Full Screen — Global Message Search

> ⚠️ Rules in this file **override** `design-system/echochat/MASTER.md`.

---

## Page Layout

- **结构:** Search Bar + Result List
- **搜索栏:** 胶囊输入框 + 取消按钮

## Icon Mapping

| 位置 | uni-icons type | size | color |
|------|---------------|------|-------|
| 搜索框内图标 | `search` | 18 | `#94A3B8` |

## Search Bar

- **输入框:** `#F1F5F9` 背景，`36rpx` 圆角，`68rpx` 高
- **搜索图标:** 左侧内嵌 `uni-icons search`
- **取消按钮:** `#2563EB` 文字，无边框

## Result Item

```
[Avatar 80rpx] | [Sender Name]
               | [Content (单行截断)]
               | [Time]
```

- **头像:** 80rpx 圆角 20rpx
- **发送者名称:** `28rpx` `font-weight: 500` `#1E293B`
- **内容:** `26rpx` `#64748B` 单行截断
- **时间:** `22rpx` `#94A3B8`

## Empty State

- **位置:** 上方 200rpx 留白后居中
- **文字:** "未找到相关消息" `28rpx` `#94A3B8`
