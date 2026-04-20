# EchoChat 项目开发进度

> **最后更新**：2026-04-20（Phase 2d 完成 + UX 优化 + 已读状态刷新持久化）
> **当前阶段**：Phase 2d 全部完成 + Bug 修复 7 项 + UX 优化 2 项
> **当前分支**：`feature/phase2c-group-read-receipt`（注：Phase 2d 工作误用了 2c 分支，下一阶段 2e 直接新建分支开发）
> **实施计划**：`docs/plans/2026-03-04-phase2d-implementation.plan.md`
> **设计文档**：`docs/plans/2026-03-04-phase2d-design.md`
> **本次修复报告**：`test-report-phase2d-bugfix.md`

---

## 🐛 2026-04-20 Bug 修复：已读状态刷新后丢失（变回"未读"）

**问题**：单聊/群聊页面中大量消息显示"已读"的情况下，刷新浏览器 → 所有已读标签全部变为"未读" / "N人已读"消失，直到对方再次触发读消息才恢复 —— 严重影响真实性与信任度。

**根因（单句）**：前端 `readStatusMap`（对方已读位置）和 `groupReadCountMap`（群聊已读计数）仅由 WebSocket 事件 `im.message.read.ack / im.message.read.count` 填充，从未在页面加载时从后端拉取，刷新后 state 归零且无 API 补回 → `isRead()` 判定 `msg.id <= 0` 恒为 false。后端数据库 `im_conversation_members.last_read_msg_id` 与 `im_message_reads` 数据其实都存在，只是未暴露给前端初始化。

**修复方案（前后端联动，不新增 API，仅扩展 `HistoryMessageResponse`）**：

| 端 | 改动 |
|---|---|
| 后端 DTO | `HistoryMessageResponse` 新增 `peer_last_read_msg_id int64`（单聊）、`read_count_map map[int64]int`（群聊 omitempty） |
| 后端 Service | `GetHistoryMessages` 按会话类型分支：type=1 → `GetPeerUserID` + `GetMember` 拿对方 `last_read_msg_id`；type=2 → 过滤自己发送的消息 ID，调 `readRecorder.GetReadCountBatch` |
| 前端 Store | `loadHistoryMessages` 仅在「首次加载（messages.length===0）」时回填，避免后续"加载更多"时用历史数据覆盖 WS 增量的最新态 |

**涉及文件**：
- `backend/go-service/app/dto/im_dto.go`：DTO 扩展
- `backend/go-service/app/im/service/im_service.go`：GetHistoryMessages 回填逻辑
- `frontend/src/store/chat.js`：loadHistoryMessages 消费新字段

**Playwright 端到端验证**：
1. 数据库事实：会话 6 中 `bojinyuan(7)` last_read_msg_id=202（已读到 id=202）
2. duanlingyun 登录 → API 返回 `peer_last_read_msg_id=202` ✓
3. 会话页首屏：21 条自己发送的消息 → 20 条"已读" + 1 条新插入的 id=203"未读" ✓（与期望完全吻合）
4. 群聊验证：bojinyuan 登录 → 群 8 API 返回 `read_count_map={94:1, 95:1, ..., 139:2, 140:2}`（共 13 条自己发的消息）→ 页面正确显示 11 个"1人已读" + 2 个"2人已读" ✓
5. 视觉回归：同一截图中语音"未听红点"仍正常工作（表明两项 UX 优化无互相干扰）

**设计权衡**：
- 为什么不新增独立 API：避免每次进入会话多一次往返请求
- 为什么仅首次加载回填：后续 WS 事件已能增量更新，用历史数据覆盖反而可能倒退
- 为什么群聊只查"自己发的消息"：前端仅在 `isSelf=true` 时展示"N人已读"，避免无用 DB 查询

---

## 🎙️ 2026-04-20 UX 优化：语音消息「未听红点」（仿微信）

**需求**：通用"已读"状态基于滚动可见性，适合文字/图片（肉眼可直接阅读）；但**语音必须点击播放才算"听过"**，仅"已读"不能反映用户是否真的听过。参照微信设计，为「对方发来的语音」叠加独立的"未播放"视觉提示（红点）。

**设计决策（均为推荐方案）**：
| 维度 | 取舍 |
|---|---|
| 作用范围 | 仅「对方发来的」语音显示红点（自己发的无此语义） |
| 持久化 | 本地 localStorage（按 userId 隔离） —— 不同步后端，属于私人视图态 |
| 触发时机 | 用户点击播放即刻清除红点（不要求播放完整） |
| 作用层级 | 仅聊天详情页；会话列表不处理 |

**核心实现**：
- `chat store` 新增 `voicePlayedMap: {[msgId]: true}` + `markVoicePlayed / isVoicePlayed / loadVoicePlayedState / resetVoicePlayedState` 4 个 API
- localStorage key：`echo:voice-played:{userId}`，完整 JSON 覆盖写入
- `MsgVoice.vue` 自主消费 store（父级无需改动，单聊/群聊同构生效）
  - 模板：`<view v-if="showUnplayedDot" class="unplayed-dot" />`（10rpx 红圆，紧贴气泡右侧）
  - `showUnplayedDot = computed(() => !isSelf && msg.id && !chatStore.isVoicePlayed(msg.id))`
  - `onTogglePlay` 首行：`if (!isSelf && msg.id) chatStore.markVoicePlayed(msg.id)`
- `user.store.logout()`：动态导入 chat store 调用 `resetVoicePlayedState()` 防止串用户
- `chat.store.initWsListeners()`：按当前 `userStore.userInfo.id` 懒加载对应缓存

**涉及文件**：
- `frontend/src/store/chat.js`：新增 voicePlayed 状态 + 4 个方法
- `frontend/src/components/msg/MsgVoice.vue`：模板加红点 + 消费 store
- `frontend/src/store/user.js`：logout 清理运行时 state

**Playwright 验证**：
1. duanlingyun(id=13) 登录 → 会话 6（内含自己 3 条语音 + 对方 5 条语音）
2. 初始：8 条语音 → 5 个红点（全部对方发） + 0 个红点（自己 3 条 `voice-self`）✓
3. 点击任意两条对方语音 → 红点数 5→4→3 ✓
4. localStorage 写入 `echo:voice-played:13 = {195:true, 196:true}` ✓
5. 页面刷新 → 仍只剩 3 个红点（持久化生效）✓

---

## 💡 2026-04-20 UX 优化：聊天页「新消息悬浮提示 + 按需已读」

**问题**：用户停留在聊天页旧消息位置时，对方发来新消息 → 页面不会自动滚动到底 → 但 store 层一律自动 `markRead` → 对方看到"已读"但用户实际没看到消息，产生体验错位。

**方案（混合策略，业内通用）**：
- 贴底（距底 < 150px）时：新消息自动滚动到底部 + 标记已读（保留原有体验）
- 远离底部时：不滚动、不标记已读，右下角显示「↓ N 条新消息」悬浮胶囊按钮
- 点击悬浮按钮：滚到底 + 清零计数 + 标记已读
- 用户手动滚回底部：自动隐藏悬浮 + 标记已读

**技术实现（`maxScrollTop` 追踪策略）**：
- `@scroll` 事件中维护"历史最大 scrollTop"，`maxScrollTop - scrollTop < 150` 判断是否贴底
- 不依赖 `@scrolltolower` 作为唯一权威信号，对 `scroll-into-view` + 动态 scrollHeight 都正确响应
- Store 层移除 `_onNewMessage` 中对当前会话的自动 `markRead`，把决策权交给页面

**涉及文件**：
- `frontend/src/store/chat.js`：移除新消息到达时的自动 `markRead`
- `frontend/src/pages/chat/conversation.vue`：单聊滚动感知 + 悬浮按钮
- `frontend/src/pages/group/conversation.vue`：群聊同构改造

**验证结果**：Playwright 全流程通过三种场景（贴底自动滚、远离底部显示悬浮、点击悬浮滚到底清零）。

### 🐛 UX 优化后续修复（2026-04-20）：watch 逻辑误把"加载历史消息"计入 newMsgCount

**问题**：用户在页面中段滚动时触发「加载更多」→ 从数组**头部**插入 20 条历史消息 → `messages.value.length` 变化 → watch 误以为是"对方的新消息"并 `newMsgCount += 20`，悬浮按钮显示"40+ 条新消息"。

**根因**：watch 回调只看 `messages.length` 增量（delta），未区分"头部插入历史"与"尾部追加新消息"。读取 `messages[length-1]` 拿到的是已存在的最后一条消息，若恰好是对方发的即通过 fromOther 判断并累加。

**修复策略**：监听「末尾消息标识（id / client_msg_id）」是否变化，而非仅看长度：
- 头部插入历史消息 → tail 未变 → 直接返回，不做任何处理
- 尾部追加一条新消息 → tail 变化 → 才进入"fromSelf/fromOther/nearBottom"判断分支
- 自己发的消息 tail 也会变但会被 `fromSelf` 拦下

**真实双账号验证（duanlingyun ↔ bojinyuan，会话 id=6）**：

| 操作 | messages.length | scrollTop | scrollHeight | 悬浮按钮 |
|---|---|---|---|---|
| 初始贴底 | 20 | 1419 | 2066 | 无 |
| 滚至顶部触发加载更多 ×3 | 76 | 0 | 4823 | **无** ✅ |
| 对方发 1 条 | 77 | 1003 | 4877 | "1 条新消息" ✅ |
| 对方再发 2 条 | 79 | 1003 | 4877 | "3 条新消息" ✅ |
| 再次加载更多历史 | 79+10 | 0 | 4986 | **仍为 "3 条新消息"** ✅ |
| 点击悬浮 | 79+10 | 4339 (贴底) | 4986 | 消失 ✅ |

**涉及文件**：
- `frontend/src/pages/chat/conversation.vue`：`lastMsgCount` → `lastTailKey` 改造
- `frontend/src/pages/group/conversation.vue`：相同改造（群聊）

### 🐛 语音消息 H5 兼容修复（2026-04-20）

**问题链**（按排查顺序）：
1. **事件层**：PC Chrome 按住"按住说话"按钮无反应 —— 只绑定了 `@touchstart/move/end`，现代浏览器鼠标不合成 touch 事件
2. **录音 API 层**：修复事件绑定后报 `method 'uni.getRecorderManager' not supported` —— uni-app H5 端不支持原生 `getRecorderManager`
3. **上传层**：即便录到 Blob，`uni.uploadFile(blob:URL)` 推断出的 filename 无扩展名，被后端白名单拦下
4. **后端校验层**：`allowedVoiceExts` 只允许 `.mp3/.wav/.aac/.m4a`，不接受 H5 MediaRecorder 输出的 webm/ogg

**修复链**：
| 层 | 改动 |
|---|---|
| 事件 | VoiceRecorder 同时监听 touch + mouse 事件，`pressing` 重入守卫，`@mouseleave` 兜底取消 |
| 录音 | 新增 `H5Recorder` 类，基于 `MediaRecorder` + `getUserMedia` 实现 uni 录音接口（`onStart/onStop/onError/start/stop`），`createRecorder()` 工厂按平台自动选择 |
| 上传 | `uploadVoice` 增加 blob 参数分支，H5 走 `fetch` + `FormData` 精确控制 filename（基于 mimeType 推断扩展名，如 `voice-{ts}.webm`）|
| 后端 | `allowedVoiceExts` 新增 `.webm`、`.ogg`，错误消息同步更新 |

**验证结果（Playwright 自动化）**：

| 阶段 | class | 按钮文字 | 遮罩 | 计时 |
|---|---|---|---|---|
| 按压中 (4s) | `record-btn recording` | "松开发送" | ✓ 显示 | "3\"" |
| 按压中 (5.5s) | 同上 | 同上 | ✓ | "5\"" （跳动正常）|
| 松开后 | `record-btn` | "按住 说话" | 消失 | — |

数据库验证：连续 3 次录音上传成功（1/2/5 秒，size 18K/35K/75K），`im_messages` type=3（语音）记录正确写入，MinIO 中 `.webm` 文件可访问。

**涉及文件**：
- `frontend/src/components/chat/VoiceRecorder.vue`：核心改造（事件兼容 + H5Recorder 类）
- `frontend/src/api/file.js`：`uploadVoice` 增加 blob 分支，新增 `_doUploadBlob` 私有方法
- `frontend/src/pages/chat/conversation.vue` & `frontend/src/pages/group/conversation.vue`：`onVoiceRecorded` 按 mimeType 生成 fileName 并传 blob
- `backend/go-service/app/file/service/file_service.go`：`allowedVoiceExts` 扩展 + 错误消息更新

---

## 🔧 2026-04-20 Bug 修复清单

| # | 根因 | 修复 | 文件 |
|---|---|---|---|
| B1 | 文件上传 BASE_URL 硬编码为 `:8080`（与实际 `:8085` 不符）导致 `ERR_CONNECTION_REFUSED` | 改为复用 `@/utils/request` 中的统一 BASE_URL | `frontend/src/api/file.js` |
| B2 | MinIO bucket 默认 private，图片 URL 匿名访问返回 403 | 启动时自动设置 bucket public-read 策略（仅 `s3:GetObject`） | `backend/go-service/pkg/storage/minio.go` |
| B3 | MsgImage flex 布局塌陷，`<uni-image>` 宽度为 0 | `.grid-single/2col/3col` 增加显式宽高和 `display:block` | `frontend/src/components/msg/MsgImage.vue` |
| B4 | H5 下 `URL.createObjectURL(file)` 丢失文件名，显示为 `file-1776xxx` | 前端发送消息时优先使用用户选择的 `file.name` | `frontend/src/pages/{chat,group}/conversation.vue` |

**验证结果**：Playwright MCP 全流程通过（图片/文件上传、缩略图渲染、大图预览、管理端消息详情）。详见 `test-report-phase2d-bugfix.md`。

---

## 🛠️ 2026-04-20 工程化脚本改造（开发者体验）

引入社区通用的 **setup / start 职责分离** 模式（参考 Rails `bin/setup` + `bin/dev`、Django 模板、Next.js 企业模板）：

| 脚本 | 定位 | 使用频率 |
|---|---|---|
| `scripts/dev-setup.sh` | **首次环境初始化**：Docker 检查 + 容器拉起 + 重试式健康检查（pg/redis/minio）| 首次 clone / 重建卷 |
| `scripts/start.sh` | **日常启动**：Docker 中间件 + Go 后端 + 前台 + 管理端全量拉起，端口占用自动跳过 | 每天多次 |
| `scripts/stop.sh` | **日常停止**：优雅终止应用层（先 TERM 后 KILL 兜底），默认保留容器，支持 `--all` 全关 | 每天多次 |
| `scripts/status.sh` | **状态查看**：三应用端口 + 三容器状态一览 | 排障随时 |

核心特性：
- **端口占用自动跳过**：`lsof` 检测，已占用则 WARN 跳过，不重复启动
- **PID + 日志分离**：PID 写入 `.run/*.pid`，stdout 写入 `.run/logs/*.log`（已加入 `.gitignore`）
- **MinIO 健康等待**：`dev-setup.sh` 新增 `/minio/health/live` 重试检查，补齐原脚本遗漏
- **单项启停**：`./scripts/start.sh backend|frontend|admin|docker`

涉及文件：
- 新增 `scripts/start.sh`、`scripts/stop.sh`、`scripts/status.sh`
- 改造 `scripts/dev-setup.sh`（新增 MinIO 检查 + 头部使用时机注释）
- 更新 `README.md`（置顶一键脚本章节，阐述首次/日常使用顺序）
- 更新 `.gitignore`（忽略 `.run/`）

---

## 一、Phase 2b Task 完成状态

| Task | 描述 | 状态 | 备注 |
|------|------|------|------|
| Task 0 | IM Model + 数据库迁移 + 常量 | ✅ 完成 | 3 张表 + init.sql + AutoMigrate |
| Task 1 | WS 事件路由表机制 | ✅ 完成 | Hub.RegisterEvent/DispatchEvent |
| Task 2 | IM DAO 层 | ✅ 完成 | ConversationDAO + MessageDAO |
| Task 3 | IM Service 核心业务 + DTO | ✅ 完成 | 9 个业务方法 + 接口注入 |
| Task 4 | WS 事件处理器 + 离线推送 | ✅ 完成 | 4 个事件 + OfflinePusher |
| Task 5 | REST Controller + Router + Wire | ✅ 完成 | 7 个 REST API + 完整 Wire 集成 |
| Task 6 | 前台 Store + API + WS 事件 | ✅ 完成 | chat.js Store + API + TabBar badge |
| Task 7 | 会话列表页 + 聊天对话页 | ✅ 完成 | 2 个核心页面 |
| Task 8 | 设置页 + 搜索页 + 联系人改造 | ✅ 完成 | 2 个辅助页面 + 发消息跳转 |
| Task 9 | 文档更新 + 代码审查 | ✅ 完成 | 进度/架构文档同步 |
| UI 改造 | ui-ux-pro-max 规范改造 | ✅ 完成 | uni-icons 替换 emoji + 设计规范文件 |
| 代码审查修复 | 后端 7 项修复 | ✅ 完成 | P0×2 + P1×3 + 推送补全×2 |
| 用户测试修复 | 8 项 Bug 修复 | ✅ 完成 | 好友申请/接受、在线状态、UI 布局等 |

### 代码审查修复详情

| # | 优先级 | 修复内容 |
|---|--------|----------|
| Fix 1 | P0 | ClearHistory 改为个人视图操作（ClearBeforeMsgID），不再删除双方消息 |
| Fix 2 | P0 | Redis 未读数负数保护（Lua 脚本原子递减，下限为 0） |
| Fix 3 | P1 | GetConversationList N+1 查询优化（LEFT JOIN 一次获取 peerID） |
| Fix 4 | P1 | 消息搜索改用 GIN 全文索引（to_tsvector/plainto_tsquery 替代 LIKE） |
| Fix 5 | P1 | 撤回消息后更新会话预览（last_msg_content = "XX 撤回了一条消息"） |
| Fix 6 | - | im.message.new 推送补充 sender_name、sender_avatar |
| Fix 7 | - | im.message.recalled 推送补充 sender_id |

### 用户测试修复详情

| # | 修复内容 |
|---|----------|
| Fix 8 | 好友申请拒绝后重新申请失败 — FriendshipDAO 新增 ReactivateRejectedRequest 方法 |
| Fix 9 | 好友接受申请失败（反向记录 UNIQUE 冲突）— AcceptRequest 先查后改，避免重复插入 |
| Fix 10 | Redis 在线状态残留 — OnlineService 启动时清理旧在线数据（cleanStaleOnlineData） |
| Fix 11 | WS 断开时在线状态未清理 — 修正 onDisconnect 判断条件（closedByHub && isOnline） |
| Fix 12 | 前端 WS 连接未全局初始化 — App.vue onLaunch/onShow + login.vue 登录后建立连接 |
| Fix 13 | 后台管理端好友关系页用户 A 列名称错误 — 修正字段绑定 row.user_username |
| Fix 14 | 前台好友在线状态初始值缺失 — ContactService 注入 OnlineChecker，GetFriendList 返回 is_online |
| Fix 15 | 聊天页消息过多时输入框被挤出 — scroll-view 添加 height:0 + min-height:0 约束 |

---

## 二、Phase 2b 新增功能

### 即时通讯（IM）
- **消息收发**：WebSocket 全双工通讯，im.message.send → ACK + 推送
- **三态确认**：sending → sent/ACK → failed
- **消息撤回**：2 分钟内可撤回，推送 im.message.recalled
- **正在输入**：im.typing 事件，3 秒超时自动清除
- **离线消息**：WebSocket 重连后服务端主动推送未读会话摘要

### 会话管理
- **自动创建**：首次发消息时自动创建单聊会话
- **会话列表**：置顶优先 → 最后消息时间降序，LEFT JOIN 一次获取 peerID（N+1 优化）
- **会话操作**：置顶/取消、软删除（不影响对方）、清空聊天记录（个人视图 ClearBeforeMsgID）
- **未读管理**：DB unread_count + Redis STRING 全局未读数（Lua 脚本负数保护），TabBar badge 显示

### WebSocket 事件路由表
- **Hub.RegisterEvent**：业务模块注册事件处理器
- **Hub.DispatchEvent**：消息分发到匹配的处理器
- **事件清单**：im.message.send / im.message.recall / im.conversation.read / im.typing

### REST API（7 个）
| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /api/v1/im/conversations | 会话列表 |
| GET | /api/v1/im/messages | 历史消息（游标分页） |
| PUT | /api/v1/im/conversations/:id/pin | 置顶/取消 |
| DELETE | /api/v1/im/conversations/:id | 删除会话 |
| DELETE | /api/v1/im/conversations/:id/messages | 清空记录 |
| GET | /api/v1/im/messages/search | 全局搜索 |
| GET | /api/v1/im/unread | 全局未读数 |

### 前端页面（4 个）
- `pages/chat/index.vue` — 会话列表（TabBar 页面）
- `pages/chat/conversation.vue` — 聊天对话页
- `pages/chat/settings.vue` — 聊天设置页
- `pages/chat/search.vue` — 消息搜索页

### 数据库表（3 张）
- `im_conversations` — 会话表（含冗余 last_msg_* 字段）
- `im_conversation_members` — 会话成员表（个人视图：置顶/未读/软删除）
- `im_messages` — 消息表（游标分页索引 + GIN 全文搜索索引）

---

## 三、Phase 2a 完成总结

| Task | 描述 | 状态 | 备注 |
|------|------|------|------|
| Task 0-12 | WebSocket + 联系人 + 管理端 | ✅ 全部完成 | 13 个 Task + 8 项 Bug 修复 |

- WebSocket 实时通讯（Hub + Client + PubSub）
- 联系人管理 17 个 API
- 在线状态管理（Redis SET + TTL）
- 管理端扩展（在线监控 + 好友管理）

---

## 四、Phase 1 完成总结

| Task | 描述 | 状态 |
|------|------|------|
| Task 1-11 | 基础设施 + 认证 + 用户管理 | ✅ 全部完成 |

- Go 后端 15+ API、JWT 有状态认证、RBAC 角色权限
- 前台 uni-app 登录/注册/TabBar/个人中心
- 管理端 Vue 3 登录/仪表盘/用户列表/详情
- Docker Compose 一键启动

---

## 五、关键技术决策记录

### 后端（Go）
1. **框架组合**：Gin + GORM + Wire + Zap + Viper
2. **JWT 策略**：有状态 JWT，Token 按 clientType 隔离存储在 Redis
3. **WebSocket**：`gorilla/websocket` + Redis Pub/Sub 跨实例路由
4. **WS 事件路由**：Hub.eventHandlers map[string]EventHandler + RegisterEvent/DispatchEvent
5. **IM 跨模块**：FriendChecker + UserInfoGetter 接口注入（contact → im）
6. **IM 推送**：OfflineMessagePusher 接口注入（im → ws）
7. **在线状态**：混合方案（Redis SET + STRING TTL + Pub/Sub 推送）
8. **角色等级**：`auth_roles.level`（1=超管, 10=管理员, 100=普通用户）

### 前台用户端（frontend/）
1. **框架**：uni-app 3.0（Vue 3.4.21）
2. **状态管理**：Pinia 2.1.7 + pinia-plugin-persistedstate@3
3. **WebSocket**：`uni.connectSocket`（小程序）/ `WebSocket`（H5）
4. **IM Store**：chat.js（会话列表 + 消息缓存 + 三态确认 + 全局未读）
5. **设计系统**：ui-ux-pro-max 规范，MASTER.md + 页面覆盖规范
6. **图标方案**：@dcloudio/uni-ui uni-icons（easycom 自动引入，跨平台兼容）
7. **预处理器**：sass（uni-icons SCSS 依赖）

### 后台管理端（admin/）
1. **框架**：Vue 3.5+ + Vite 7.x + Element Plus
2. **HTTP 客户端**：Axios
3. **存储隔离**：localStorage key 前缀 `admin_`

---

## 六、目录结构概览

```
EchoChat/
├── backend/go-service/
│   ├── app/
│   │   ├── admin/               # 管理端
│   │   ├── auth/                # 认证模块
│   │   ├── contact/             # [Phase 2a] 联系人模块
│   │   ├── im/                  # [Phase 2b] 即时通讯模块
│   │   │   ├── controller/      # REST API 控制器
│   │   │   ├── dao/             # 数据访问（ConversationDAO + MessageDAO）
│   │   │   ├── handler/         # WS 事件处理器 + 离线推送
│   │   │   ├── model/           # 数据库模型
│   │   │   ├── service/         # 核心业务 + 接口定义
│   │   │   ├── router.go
│   │   │   └── provider.go
│   │   ├── ws/                  # [Phase 2a] WebSocket 模块
│   │   ├── constants/           # 含 im.go 常量
│   │   ├── dto/                 # 含 im_dto.go
│   │   └── provider/
│   ├── pkg/
│   │   ├── ws/                  # WS 核心（Hub 含事件路由表）
│   │   ├── db/ logs/ middleware/ utils/
│   └── router/router.go
├── frontend/                    # 前台（uni-app）
│   └── src/
│       ├── api/{auth,contact,user,im,group,file}.js
│       ├── constants/group.js    # [Phase 2c] 群聊角色/状态常量
│       ├── services/websocket.js
│       ├── store/{user,websocket,contact,chat,group}.js
│       ├── pages/chat/          # [Phase 2b] 5 个页面
│       │   ├── index.vue        # 会话列表（含群聊 Tab）
│       │   ├── conversation.vue # 单聊对话
│       │   ├── read-detail.vue  # [Phase 2c] 已读详情
│       │   ├── settings.vue     # 聊天设置
│       │   └── search.vue       # 消息搜索
│       ├── pages/group/         # [Phase 2c] 7 个页面
│       │   ├── conversation.vue # 群聊对话（含 @选择器 + 已读计数 + 禁言提示）
│       │   ├── create.vue       # 创建群聊
│       │   ├── settings.vue     # 群设置
│       │   ├── members.vue      # 成员管理
│       │   ├── invite.vue       # 邀请入群
│       │   ├── join-requests.vue # 入群审批
│       │   └── search.vue       # 搜索群聊
│       ├── pages/contact/       # [Phase 2a] 6 个页面
│       ├── components/msg/     # [Phase 2d] 消息类型组件
│       │   ├── MsgText.vue     # 文本消息
│       │   ├── MsgImage.vue    # 图片消息（网格+预览）
│       │   ├── MsgVoice.vue    # 语音消息（播放+波形）
│       │   └── MsgFile.vue     # 文件消息（卡片+下载）
│       ├── components/chat/    # [Phase 2d] 聊天辅助组件
│       │   ├── MorePanel.vue   # "+"展开面板
│       │   └── VoiceRecorder.vue # 语音录制
│       └── components/CustomTabBar.vue（含 badge）
├── admin/                       # 管理端（Vue 3 + Element Plus）
│   └── src/views/
│       ├── message/            # [Phase 2d] 消息管理
│       │   ├── list.vue        # 消息列表（多条件筛选+操作）
│       │   └── stats.vue       # 消息统计（ECharts 仪表板）
├── deploy/
├── design-system/
└── docs/
    ├── api/
    ├── plans/
    ├── progress/CURRENT_STATUS.md
    └── conventions/
```

---

## 七、开发测试指南

### 服务管理命令

> 建议开 4 个终端窗口，分别运行各服务。终止方式统一为 `Ctrl + C` 或 `kill` 命令。

#### 基础设施（Docker Compose）

```bash
# 一键启动全部基础设施（PostgreSQL + Redis + MinIO）
cd deploy && docker compose -f docker-compose.dev.yml up -d postgres redis minio

# 查看容器状态
cd deploy && docker compose -f docker-compose.dev.yml ps

# 一键停止全部基础设施
cd deploy && docker compose -f docker-compose.dev.yml stop
```

| 服务 | 地址 | 单独启动 | 单独停止 |
|------|------|----------|----------|
| PostgreSQL | localhost:5432 | `docker compose -f docker-compose.dev.yml up -d postgres` | `docker compose -f docker-compose.dev.yml stop postgres` |
| Redis | localhost:6379 | `docker compose -f docker-compose.dev.yml up -d redis` | `docker compose -f docker-compose.dev.yml stop redis` |
| MinIO API | localhost:9000 | `docker compose -f docker-compose.dev.yml up -d minio` | `docker compose -f docker-compose.dev.yml stop minio` |
| MinIO 控制台 | localhost:9001 | （同上） | （同上） |

#### 应用服务

| 服务 | 地址 | 启动命令 | 终止命令 |
|------|------|----------|----------|
| Go 后端 | http://localhost:8085 | `cd backend/go-service && go run cmd/server/main.go` | `Ctrl+C` 或 `kill $(lsof -ti :8085)` |
| 前台用户端 (H5) | http://localhost:5173 | `cd frontend && npm run dev:h5` | `Ctrl+C` 或 `kill $(lsof -ti :5173)` |
| 后台管理端 | http://localhost:3100 | `cd admin && npm run dev` | `Ctrl+C` 或 `kill $(lsof -ti :3100)` |

#### Go 后端快速重启（一行命令）

```bash
kill $(lsof -ti :8085) 2>/dev/null; sleep 2; cd backend/go-service && go run cmd/server/main.go
```

### 测试账号

| 账号 | 密码 | 角色 | 用途 |
|------|------|------|------|
| `super_admin` | `admin123456` | super_admin | 系统预置唯一超管 |
| `admin_test` | `admin123456` | user + admin | 管理端登录推荐 |
| `testuser1` | `test123456` | user + admin | 前台登录测试 |
| `testuser` | `test123456` | user | 前台登录测试 |

### Phase 2b 可测试功能

- **会话列表**：发消息自动创建会话 → 列表排序 → 置顶 → 长按删除
- **聊天**：发送文本 → 三态确认 → 撤回（2分钟内）→ 正在输入提示
- **离线消息**：断开重连 → 自动推送未读摘要 → TabBar badge 更新
- **消息搜索**：全局关键词搜索 → 跳转到对应会话
- **联系人入口**：好友详情页 → 发消息 → 跳转聊天页

---

## 八、Phase 2c — 群聊与已读回执

> **状态：** ✅ 已完成
> **设计文档：** `docs/plans/2026-03-04-phase2c-design.md`
> **实施计划：** `docs/plans/2026-03-04-phase2c-implementation.plan.md`
> **分支：** `feature/phase2c-group-read-receipt`

### 功能范围

| 模块 | 内容 |
|------|------|
| 群聊管理 | 建群/加入/退出/解散/搜索/三级角色/禁言/全体禁言/群公告/群昵称/免打扰 |
| 群消息 | 复用 im.message.* 事件 + @某人/@所有人 + 管理员撤回（无时限）+ 系统消息 |
| 已读回执 | 单聊会话级（last_read_msg_id）+ 群聊消息级（im_message_reads 表）+ 实时推送 |
| MinIO | Docker 容器 + Go SDK + 通用上传 API（群头像） |
| 管理端 | 群列表/群详情/解散群 |
| 前端 | 9 个新页面 + 群聊 Store + 会话列表 Tab 改造 |

### Task 完成状态

| Task | 描述 | 状态 |
|------|------|------|
| Task 0 | MinIO Docker + SDK + 通用上传 API | ✅ 完成 |
| Task 1 | 数据库迁移 + Model + 常量 | ✅ 完成 |
| Task 2 | Group DAO 层 | ✅ 完成 |
| Task 3 | Group Service 业务逻辑 + WS 推送 | ✅ 完成 |
| Task 4 | Group Controller + Router + Wire | ✅ 完成 |
| Task 5 | IM Service 扩展（群消息/@提醒/管理员撤回） | ✅ 完成 |
| Task 6 | 已读回执后端（单聊 + 群聊） | ✅ 完成 |
| Task 7 | 代码审查修复（Critical 4 项 + Important 4 项） | ✅ 完成 |
| Task 8 | 前端已读回执 UI（单聊标记 + 群聊计数 + 详情页） | ✅ 完成 |
| Task 9 | 前端群聊 Store + API 封装 + WS 事件监听 | ✅ 完成 |
| Task 10 | 群聊核心页面（Tab 切换 + 群聊对话页 + 创建群聊页） | ✅ 完成 |
| Task 11 | 群聊管理页面（群设置 + 成员管理 + 邀请入群） | ✅ 完成 |
| Task 12 | 群聊辅助功能（入群审批 + 搜索群聊） | ✅ 完成 |
| Task 13 | 管理端群组管理 + 文档更新 | ✅ 完成 |
| 代码审查修复 | 14 项修复（Critical×5 + Important×4 + Minor×2 + Suggestion×3） | ✅ 完成 |
| Playwright 测试 + 用户反馈修复 | 浏览器端到端测试 + 21 项修复（搜索/UI/交互/功能增强） | ✅ 完成 |

### 用户测试修复详情（Phase 2c）

| # | 修复内容 |
|---|----------|
| Fix T1 | create.vue 创建群后跳转使用 `result.id`（原 `result.group_id` 字段不存在） |
| Fix T2 | conversation/settings/members/join-requests 用 `user_nickname` 替代 `username`（对齐 DTO） |
| Fix T3 | 管理端群组详情弹窗 UI 全面重设计（卡片+头像+分区布局+成员mini头像） |
| Fix T4 | 群搜索结果已在群内的显示「已加入」标签，不再显示「申请加入」按钮 |
| Fix T5 | group_dao.go `SearchGroups` 改用 ILIKE 模糊匹配（原 to_tsvector 不支持混合词搜索） |
| Fix T6 | conversation.vue 增加 groupId=0 时从 chatStore 回退查找 group_id |
| Fix T7 | create.vue 最低选择人数从 2 改为 1（支持 2 人群聊） |
| Fix T8 | create.vue 支持搜索非好友用户并加入群聊（全站用户搜索 + 非好友标签） |
| Fix T9 | invite.vue 支持搜索非好友用户并邀请入群（同 create.vue 改造） |
| Fix T10 | admin/list.vue 成员表用 `username` 替代 `user_nickname`（对齐 admin DTO） |
| Fix T11 | 群聊已读回执优化：无人已读时显示「0人已读」（含点击跳转已读详情功能） |
| Fix T12 | 新好友聊天页 conversationId=0 时「加载更多」点击报错（hasMore 增加 ID 校验） |
| Fix T13 | 联系人 TabBar 添加好友申请未读数 badge（与消息 Tab 一致的交互体验） |
| Fix T14 | App.vue 启动时预加载好友申请数，确保 badge 立即可见 |
| Fix T15 | 单聊已读回执：后端 MarkRead 缺失 `im.message.read.ack` 推送（补全对方已读通知链路） |
| Fix T16 | 群成员列表页为所有角色添加身份标识（群主/管理员/成员） |
| Fix T17 | 全局修复 `e?.data?.message` → `e?.message`（8 个页面 18 处，确保后端错误信息正确展示） |
| Fix T18 | members.vue 管理操作弹窗改为自定义组件（头像+角色+图标操作列表），替代 uni.showActionSheet；三个点按钮从 @longpress 改为 @tap |
| Fix T19 | 已读详情页展示群内昵称：后端 DTO 新增 group_nickname 字段 + DAO 批量查询群昵称 + 前端主显群昵称/副显真实昵称 |
| Fix T20 | 消息免打扰 API 补全：后端缺失 DAO/Service/Controller/Router 完整链路（PUT /api/v1/im/conversations/:id/dnd） |
| Fix T21 | 联系人 Tab 页切换回来后数据不刷新：onMounted → 增加 onShow 生命周期钩子自动重新获取好友列表和待处理申请数 |

### 代码审查修复详情（Phase 2c）

| # | 优先级 | 修复内容 |
|---|--------|----------|
| Fix C1 | Critical | conversation.vue 角色类型不匹配：字符串改为 GROUP_ROLE 数字常量 |
| Fix C2 | Critical | settings.vue 群公告字段名 announcement 改为 notice（对齐后端 DTO） |
| Fix C3 | Critical | chat.js sendMessage 未传递 at_user_ids 到 WS payload |
| Fix C4 | Critical | create.vue 创建成功后导航错误：改为 /pages/group/conversation + 正确参数 |
| Fix C5 | Critical | admin/provider.go Wire Set 未注册 GroupManageService/Controller |
| Fix I1 | Important | store/group.js searchGroups 添加 append 参数解决分页加载竞态 |
| Fix I2 | Important | settings.vue 改为 fetchMembers() 刷新数据，不直接修改 computed 引用 |
| Fix I3 | Important | 新增 constants/group.js 前端角色常量定义，消除魔数 |
| Fix I4 | Important | group_manage_service.go 列表查询 N+1 优化：批量查询用户名和成员数 |
| Fix M1 | Minor | file.js JSON.parse 添加 try/catch 异常保护 |
| Fix M2 | Minor | conversation.vue isSelf 移除冗余临时状态条件 |
| Fix S1 | Suggestion | 群聊对话页新增禁言状态检测和输入栏禁用提示 |
| Fix S2 | Suggestion | join-requests.vue 注册 WS group.join.request 事件实时刷新 |
| Fix S3 | Suggestion | read-detail.vue 获取失败时添加 uni.showToast 用户提示 |

### Phase 2c 新增内容

#### 后端新增模块
- **file/** — 文件上传（MinIO SDK + 通用上传 API）
- **group/** — 群聊管理（Controller + Service + DAO + Model + Router）
  - 18 个群管理 REST API + 11 个 WS 群事件推送
- **im/ 扩展** — 群消息发送/撤回 + @提醒 + 单聊/群聊已读回执
- **admin/ 扩展** — 群组列表 + 群组详情 + 解散群聊

#### 前端新增页面（9 个）
| 页面 | 路径 | 功能 |
|------|------|------|
| 群聊对话页 | `pages/group/conversation.vue` | 群消息收发 + @选择器 + 已读计数 |
| 创建群聊页 | `pages/group/create.vue` | 好友/非好友多选 + 全站用户搜索 + 群名称输入 |
| 群设置页 | `pages/group/settings.vue` | 群信息修改 + 成员概览 + 退出/解散 |
| 群成员页 | `pages/group/members.vue` | 成员列表 + 全角色标识（群主/管理员/成员）+ 自定义操作弹窗 + 角色管理 + 禁言操作 |
| 邀请入群页 | `pages/group/invite.vue` | 好友/非好友多选 + 全站用户搜索 + 排除已在群内成员 |
| 入群审批页 | `pages/group/join-requests.vue` | 申请列表 + 通过/拒绝操作 |
| 搜索群聊页 | `pages/group/search.vue` | 关键词搜索 + 申请加入 + 已加入状态显示 |
| 已读详情页 | `pages/chat/read-detail.vue` | 已读/未读成员列表（群聊消息级）+ 群昵称优先展示 + 真实昵称副行 |
| 会话列表改造 | `pages/chat/index.vue` | Tab 切换（全部/单聊/群聊）+ @标记 + 免打扰标识 |

#### 管理端新增
| 页面 | 路径 | 功能 |
|------|------|------|
| 群组列表 | `views/group/list.vue` | 搜索 + 分页 + 详情弹窗 + 解散群聊 |

#### 数据库新增/变更
- `im_groups` — 群信息表（新增）
- `im_group_join_requests` — 入群申请表（新增）
- `im_message_reads` — 群消息已读表（新增）
- `im_conversation_members` — 扩展字段（role, nickname, is_muted, is_do_not_disturb, joined_at, at_me_count）
- `im_messages` — 扩展字段（at_user_ids BIGINT[]）

---

## 九、Phase 2d — 消息类型扩展

> **状态：** ✅ 已完成
> **设计文档：** `docs/plans/2026-03-04-phase2d-design.md`
> **实施计划：** `docs/plans/2026-03-04-phase2d-implementation.plan.md`
> **分支：** `feature/phase2d-message-types`

### 功能范围

| 模块 | 内容 |
|------|------|
| 文件上传增强 | 50MB 上传上限、图片缩略图（200px JPEG）、语音校验（mp3/wav/aac/m4a, 最长 60s） |
| 富媒体消息 | 图片（多图网格 + 大图预览）、语音（录制+播放+波形）、文件（卡片+下载+预览） |
| IM Service | extra JSON 存储、会话列表预览文案（[图片x3]/[语音 12"]/[文件] xxx.pdf） |
| 前端组件 | MsgText/MsgImage/MsgVoice/MsgFile + MorePanel + VoiceRecorder |
| 管理端 | 消息列表（多条件筛选+撤回+删除+详情）+ 消息统计仪表板（ECharts 四图表） |

### Task 完成状态

| Task | 描述 | 状态 |
|------|------|------|
| Task 1 | 文件上传服务增强（50MB + 缩略图 + 语音校验） | ✅ 完成 |
| Task 2 | 消息 extra 结构定义 + DTO 扩展 + 常量启用 | ✅ 完成 |
| Task 3 | IM Service 适配富媒体消息 | ✅ 完成 |
| Task 4 | 管理端消息 DAO + Service + Controller | ✅ 完成 |
| Task 5 | 消息组件体系 + conversation 页改造 | ✅ 完成 |
| Task 6 | 图片消息完整流程 | ✅ 完成 |
| Task 7 | 语音消息完整流程 | ✅ 完成 |
| Task 8 | 文件消息完整流程 | ✅ 完成 |
| Task 9 | 输入栏改造 | ✅ 完成 |
| Task 10 | 管理端消息列表页 | ✅ 完成 |
| Task 11 | 管理端消息统计页 | ✅ 完成 |
| Task 12 | 管理端路由 + Store + API | ✅ 完成 |
| Task 13 | 群聊适配 + 编译验证 | ✅ 完成 |
| Task 14 | 代码审查 + 文档更新 | ✅ 完成 |
| 代码审查修复 | C1: im_groups 表名修复 + C2: 管理端撤回 WS 推送补全 | ✅ 完成 |

### 后端新增/修改

#### 文件上传 API
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/v1/upload/image | 图片上传（含缩略图生成） |
| POST | /api/v1/upload/voice | 语音上传（含时长校验） |

#### 管理端消息 API
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/admin/messages | 消息列表（分页+多条件筛选） |
| GET | /api/v1/admin/messages/:id | 消息详情 |
| DELETE | /api/v1/admin/messages/:id | 删除消息（软删除） |
| PUT | /api/v1/admin/messages/:id/recall | 撤回消息（+WS 推送） |
| GET | /api/v1/admin/messages/stats | 消息统计 |

### 前端新增组件

| 组件 | 路径 | 功能 |
|------|------|------|
| MsgText.vue | components/msg/ | 文本消息渲染 |
| MsgImage.vue | components/msg/ | 图片网格 + 大图预览 |
| MsgVoice.vue | components/msg/ | 语音播放 + 波形 + 时长 |
| MsgFile.vue | components/msg/ | 文件卡片 + 下载 + 预览 |
| MorePanel.vue | components/chat/ | "+"展开面板 |
| VoiceRecorder.vue | components/chat/ | 长按录音 |

### 管理端新增页面

| 页面 | 路径 | 功能 |
|------|------|------|
| 消息列表 | views/message/list.vue | 多条件筛选 + 操作 + 详情弹窗 |
| 消息统计 | views/message/stats.vue | 趋势折线图 + 类型饼图 + 活跃排行 |

### Playwright 端到端测试（2026-04-17）

| 测试项 | 结果 | 说明 |
|------|------|------|
| 前台单聊输入栏改造 | ✅ 通过 | 语音切换 + MorePanel 展开正常 |
| 前台群聊输入栏改造 | ✅ 通过 | 与单聊一致的改造结构 |
| 管理端消息列表展示 | ✅ 通过 | 表格、分页、操作按钮正常 |
| 管理端筛选功能 | ✅ 通过 | 关键词/类型/发送者ID/重置全部正常 |
| 管理端撤回/删除操作 | ✅ 通过 | 确认弹窗 + 状态变更 + 按钮联动正常 |
| 管理端消息详情弹窗 | ✅ 通过 | 发送者信息 + 元数据 + 内容展示正常 |
| 管理端消息统计图表 | ✅ 通过 | ECharts 线图/饼图/条形图渲染正常 |

> 详细报告：`test-report-phase2d.md`

### 留待后续阶段

- 群头像上传 UI 完善
- 消息转发功能
- 视频消息支持
