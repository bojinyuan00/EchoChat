# Design System Master File

> **LOGIC:** When building a specific page, first check `design-system/echochat/pages/[page-name].md`.
> If that file exists, its rules **override** this Master file.
> If not, strictly follow the rules below.

---

**Project:** EchoChat
**Generated:** 2026-03-03 (via ui-ux-pro-max)
**Category:** Mobile Messaging Application

---

## Global Rules

### Color Palette

| Role | Hex | CSS Variable | Usage |
|------|-----|--------------|-------|
| Primary | `#2563EB` | `--color-primary` | 按钮、链接、选中态、自己的消息气泡 |
| Secondary | `#3B82F6` | `--color-secondary` | 次级操作、hover 态 |
| CTA/Accent | `#10B981` | `--color-cta` | 在线状态、成功提示 |
| Background | `#F8FAFC` | `--color-background` | 页面主背景 |
| Surface | `#FFFFFF` | `--color-surface` | 卡片、列表项背景 |
| Text | `#1E293B` | `--color-text` | 主要文字 |
| Muted | `#94A3B8` | `--color-muted` | 辅助文字、时间戳、placeholder |
| Border | `#E2E8F0` | `--color-border` | 分隔线 |
| Border Light | `#F1F5F9` | `--color-border-light` | 浅分隔线 |
| Danger | `#EF4444` | `--color-danger` | 错误、删除、未读 badge |

**Color Notes:** Professional blue + clean slate gray

### Typography (uni-app)

- **Body Font:** System default (uni-app 跨平台自动适配)
- **Line Height:** 1.5 for body, 1.3 for headings
- **Font Size Scale (rpx):**
  - xs: `22rpx` — 极小辅助文字
  - sm: `24rpx` — 时间戳、标签
  - base: `28rpx` — 正文、列表内容
  - md: `30rpx` — 列表标题
  - lg: `32rpx` — 页面标题
  - xl: `36rpx` — 大标题

### Spacing Variables (rpx)

| Token | Value | Usage |
|-------|-------|-------|
| `xs` | `8rpx` | 图标间距 |
| `sm` | `16rpx` | 行内间距 |
| `md` | `24rpx` | 列表项内边距 |
| `lg` | `32rpx` | 区域边距 |
| `xl` | `48rpx` | 大区域分隔 |

### Shadow Depths

| Level | Value | Usage |
|-------|-------|-------|
| `sm` | `0 1px 2px rgba(0,0,0,0.05)` | 微弱浮起 |
| `md` | `0 4px 6px rgba(0,0,0,0.07)` | 卡片 |
| `lg` | `0 10px 15px rgba(0,0,0,0.1)` | 弹出层 |

### Border Radius Scale (rpx)

| Token | Value | Usage |
|-------|-------|-------|
| `sm` | `8rpx` | 小元素 |
| `md` | `16rpx` | 按钮、输入框 |
| `lg` | `24rpx` | 卡片、头像 |
| `xl` | `36rpx` | 胶囊按钮、搜索框 |
| `full` | `50%` | 圆形按钮 |

---

## Icon System (uni-icons)

**图标方案:** `@dcloudio/uni-ui` 的 `uni-icons` 组件（跨平台兼容）

**严禁:** 使用 emoji 或 HTML 实体字符（&#xxxx;）作为图标

| 场景 | 图标名 | 尺寸 | 颜色 |
|------|--------|------|------|
| 搜索 | `search` | 20 | `#475569` |
| 返回 | `back` | 20 | `#1E293B` |
| 更多 | `more-filled` | 20 | `#475569` |
| 发送 | `paperplane` | 22 | `#FFFFFF` |
| 发送中 | `loop` | 16 | `#94A3B8` |
| 发送失败 | `info-filled` | 18 | `#EF4444` |
| 置顶 | `top` | 16 | `#94A3B8` |
| 聊天（空状态）| `chatbubble` | 64 | `#CBD5E1` |
| 删除 | `trash` | 20 | `#EF4444` |
| 清除 | `clear` | 20 | `#EF4444` |

**使用方式:**

```html
<uni-icons type="search" size="20" color="#475569" />
```

---

## Component Specs

### Touch Targets (CRITICAL for Mobile)

```css
/* 所有可点击元素最小尺寸 */
.touchable {
  min-width: 88rpx;
  min-height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: center;
}
```

### Buttons

```css
.btn-primary {
  background: #2563EB;
  color: #FFFFFF;
  padding: 24rpx 48rpx;
  border-radius: 16rpx;
  font-weight: 600;
  font-size: 28rpx;
  transition: opacity 200ms ease;
}

.btn-primary:active {
  opacity: 0.85;
}

.btn-danger {
  background: transparent;
  color: #EF4444;
  font-size: 30rpx;
}
```

### Cards / List Items

```css
.list-item {
  display: flex;
  align-items: center;
  padding: 24rpx 32rpx;
  background-color: #FFFFFF;
  border-bottom: 1rpx solid #F1F5F9;
  transition: background-color 150ms ease;
}

.list-item:active {
  background-color: #F1F5F9;
}
```

### Avatar

```css
.avatar {
  width: 96rpx;
  height: 96rpx;
  border-radius: 24rpx;
}

.avatar-placeholder {
  background-color: #2563EB;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #FFFFFF;
  font-weight: 600;
}

.avatar-sm { width: 72rpx; height: 72rpx; border-radius: 18rpx; }
.avatar-lg { width: 120rpx; height: 120rpx; border-radius: 30rpx; }
```

### Inputs

```css
.input-wrap {
  background-color: #F1F5F9;
  border-radius: 36rpx;
  padding: 0 28rpx;
  height: 72rpx;
  display: flex;
  align-items: center;
}

.input {
  flex: 1;
  font-size: 28rpx;
  color: #1E293B;
}
```

### Badge

```css
.badge {
  min-width: 36rpx;
  height: 36rpx;
  padding: 0 8rpx;
  background-color: #EF4444;
  border-radius: 18rpx;
  color: #FFFFFF;
  font-size: 20rpx;
  line-height: 36rpx;
  text-align: center;
}
```

---

## Style Guidelines

**Style:** Clean & Functional Messaging

**Keywords:** Clean, functional, chat-focused, mobile-first, fast interactions, minimal visual noise

**Best For:** Instant messaging, social chat, real-time communication

**Key Effects:**
- `:active` 状态替代 `:hover`（移动端优先）
- 所有状态切换使用 `transition: 150-200ms ease`
- 消息气泡圆角差异化（自己 vs 对方）
- 列表项按压态反馈

---

## Anti-Patterns (Do NOT Use)

- ❌ **Emojis as icons** — 使用 `uni-icons` 组件（跨平台兼容）
- ❌ **HTML 实体字符 (&#xxxx;) as icons** — 同上
- ❌ **Low contrast text** — 最低 4.5:1 对比度
- ❌ **Instant state changes** — 必须使用 transition (150-200ms)
- ❌ **Touch targets < 44px (88rpx)** — 所有可交互元素最小 88rpx
- ❌ **相邻触控元素间距 < 8px (16rpx)** — 保证误触防护
- ❌ **硬编码 px** — uni-app 中统一使用 rpx（状态栏除外）

---

## Pre-Delivery Checklist

Before delivering any UI code, verify:

- [ ] No emojis or HTML entities used as icons (use `uni-icons`)
- [ ] All icons from `@dcloudio/uni-ui` uni-icons
- [ ] Touch targets ≥ 88rpx (44px) on all interactive elements
- [ ] `:active` states for press feedback (mobile)
- [ ] Transitions (150-200ms) on all state changes
- [ ] Text contrast 4.5:1 minimum
- [ ] No horizontal scroll on any screen
- [ ] Content not hidden behind navigation bars
- [ ] `padding-bottom` accounts for safe-area-inset-bottom
- [ ] Avatar placeholder with first character fallback
- [ ] Empty states with icon + text + hint
