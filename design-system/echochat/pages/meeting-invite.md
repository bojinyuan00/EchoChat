# Meeting Invite (邀请弹窗) Page Overrides

> **PROJECT:** EchoChat
> **Generated:** 2026-04-24 (via Task 15 UI 打磨)
> **Page Type:** Bottom Sheet / Center Modal — Share Meeting
> **Component:** `frontend/src/components/meeting/InviteDialog.vue`

> ⚠️ Rules in this file **override** `design-system/echochat/MASTER.md`.

---

## Page Layout

- **容器类型:** 居中 Modal（桌面端）/ 底部 BottomSheet（移动端 `<= 750px`）
- **遮罩:** `rgba(0, 0, 0, 0.55)` + backdrop blur `8px`
- **容器:**
  - 桌面: `520rpx × auto`，`#FFFFFF`，圆角 `24rpx`，阴影 `0 24rpx 48rpx rgba(0,0,0,0.35)`
  - 移动: `100vw`，顶部圆角 `24rpx 24rpx 0 0`
- **Z-index:** 200（同 MemberPanel 同级，因是互斥显示）

## 内容分区

### 1. 头部

- 关闭按钮（右上角 `56rpx` 圆形按钮）+ 标题 "邀请成员加入"
- 副标题：会议标题 + 会议号格式化 `xxx-xxx-xxx`

### 2. 分享方式 3 选项（并列）

| 选项 | 图标 | 说明 | 操作 |
|------|-----|------|-----|
| 复制链接 | `link` 24px | 带 Token 链接，含密码（若有） | 复制到剪贴板 + toast |
| 复制会议号 | `hash` 24px | 纯会议号（需对方手动输入密码） | 复制 + toast |
| 邀请联系人 | `user-plus` 24px | 打开联系人选择器（复用 Phase 2a 组件） | 打开抽屉 |

- **布局:** 水平排列，每项尺寸 `144rpx × 144rpx`；图标上方居中，文案下方居中
- **Hover / Active:** 卡片背景 `#F3F4F6` → `#E5E7EB`；`scale(0.98)` on active

### 3. 信息展示

- **邀请链接区:** 单行滚动 input（readonly）+ 右侧 "复制" 按钮
  - 背景 `#F8FAFC`，圆角 `12rpx`，高 `72rpx`
- **密码提示:** 若会议有密码，额外展示 `Password:` 标签 + 密码 pill（可点击复制）
- **过期时间:** 小字提示 "链接 24 小时内有效"（若后端返回过期时间）

## Color Palette

| 用途 | Hex |
|------|-----|
| 背景 | `#FFFFFF` |
| 选项卡片 | `#F8FAFC` |
| 选项悬停 | `#F3F4F6` |
| 主 CTA | `#2563EB` |
| 文本主 | `#1E293B` |
| 辅助文本 | `#64748B` |
| Toast 成功 | `#10B981` |

## Typography

| 层级 | Size | Weight |
|------|------|--------|
| 标题 | `32rpx` | 600 |
| 副标题 | `24rpx` | 400 |
| 选项名 | `26rpx` | 500 |
| 输入框 | `26rpx` | 400，`monospace` for code |

## Motion

- **打开:**
  - 桌面: 遮罩 fade-in `180ms` + 容器 `scale(0.96) → 1 + opacity 0 → 1`，`220ms cubic-bezier(0.2,0.8,0.2,1)`
  - 移动: 底部滑入 `translateY(100%) → 0`，`260ms`
- **关闭:** 反向播放，时长 `160ms`
- **复制成功 toast:** 顶部滑入 `translateY(-20px) → 0` 停留 `1.2s` 再滑出

## Interactions

- **复制链接:**
  - `clipboard.writeText(joinLink)` → 成功显示 toast "链接已复制"；失败显示 `Modal` 文本兜底
  - 含密码的链接形如 `https://echochat.app/#/pages/meeting/join?code=123456789&token=xxx`
- **复制会议号:**
  - 复制无密码纯 9 位数（不含分隔符）
- **联系人选择器:**
  - 打开 `ContactPicker` 组件（复用 Phase 2a）
  - 选中后 `POST /api/v1/meeting/rooms/:code/invite` 发送邀请
  - 成功后局部 toast，选择器不关闭方便批量邀请
- **ESC / 点击遮罩:** 关闭
