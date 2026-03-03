# Chat Index (会话列表) Page Overrides

> **PROJECT:** EchoChat
> **Generated:** 2026-03-03 (via ui-ux-pro-max)
> **Page Type:** TabBar Page — Conversation List

> ⚠️ Rules in this file **override** `design-system/echochat/MASTER.md`.

---

## Page Layout

- **结构:** Header + Scrollable List + CustomTabBar
- **Header:** 左侧"消息"标题 + 右侧搜索图标按钮
- **Empty State:** 居中图标 + 主文字 + 辅助提示

## Icon Mapping

| 位置 | uni-icons type | size | color |
|------|---------------|------|-------|
| 搜索按钮 | `search` | 20 | `#475569` |
| 空状态 | `chatbubble` | 64 | `#CBD5E1` |
| 置顶标记 | `top` | 14 | `#94A3B8` |

## Conversation Item Spec

```
[Avatar 96rpx + Badge] | [Name + Time]
                       | [Preview/Typing + Pin]
```

- **头像:** 96rpx 圆角 24rpx，无头像时使用首字母占位（Primary 背景）
- **未读 Badge:** 绝对定位在头像右上角，红色圆角
- **时间:** 右对齐，`24rpx` `#94A3B8`
- **预览:** 单行截断，有未读时 `#64748B` + `font-weight: 500`
- **正在输入:** 替换预览文字，`#2563EB`
- **置顶条目:** 背景色 `#F8FAFC` 区分
- **按压态:** `:active { background-color: #F1F5F9 }`
- **长按:** ActionSheet 操作（置顶/取消置顶、删除会话）
