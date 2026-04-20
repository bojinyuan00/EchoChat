# Phase 2d 实施计划：消息类型扩展

> **状态：** ✅ 已完成
> **设计文档：** [Phase 2d 设计文档](./2026-03-04-phase2d-design.md)
> **分支：** `feature/phase2d-message-types`
> **最后更新：** 2026-03-03

---

## 实施概览

共 14 个 Task，按依赖关系分为 4 个阶段：

| 阶段 | Task | 说明 |
|------|------|------|
| 后端基础层 | Task 1-4 | 上传增强 → DTO/常量 → IM 适配 → 管理端后端 |
| 前端消息组件 | Task 5-9 | 组件体系 → 图片 → 语音 → 文件 → 输入栏 |
| 管理端前端 | Task 10-12 | 列表页 → 统计页 → 路由/Store/API |
| 收尾 | Task 13-14 | 群聊适配+测试 → 代码审查+文档 |

---

## Task 1：文件上传服务增强 ✅

**目标**：升级现有文件上传服务，支持 50MB 上传、图片缩略图生成、语音校验。

### 修改文件

| 文件 | 操作 | 说明 |
|------|------|------|
| backend/go-service/go.mod | 修改 | 添加 `github.com/disintegration/imaging` 依赖 |
| backend/go-service/app/file/service/file_service.go | 修改 | 新增 `UploadWithThumbnail`、`UploadVoice` 方法；`UploadResult` 扩展字段 |
| backend/go-service/app/file/controller/file_controller.go | 修改 | 新增 `UploadImage`、`UploadVoice` 处理器；`maxUploadSize` 提升至 50MB |
| backend/go-service/app/file/router.go | 修改 | 新增 `/upload/image`、`/upload/voice` 路由 |

### 详细步骤

1. `go get github.com/disintegration/imaging` 安装图片处理库
2. 扩展 `UploadResult` 添加 `ThumbnailURL`、`Width`、`Height`、`Duration`、`MimeType` 可选字段
3. 新增 `ImageUploadResult` 和 `VoiceUploadResult` 结构体
4. 实现 `UploadWithThumbnail`：
   - 接收图片文件 → 解码获取宽高 → `imaging.Resize(img, 200, 0)` 生成缩略图
   - 缩略图转 JPEG(quality=80) → 上传原图+缩略图到 MinIO → 返回双 URL + 尺寸
5. 实现 `UploadVoice`：
   - 接收语音文件 → 校验扩展名（.mp3/.wav/.aac/.m4a）→ 获取时长（从 header 或前端传参）
   - 上传到 MinIO → 返回 URL + duration + size
6. Controller 新增 `UploadImage`、`UploadVoice` 处理器（含大小校验）
7. `maxUploadSize` 从 `10 << 20` 改为 `50 << 20`
8. 路由注册新接口

### 验收标准

- [x] `POST /api/v1/upload` 支持 50MB 文件
- [x] `POST /api/v1/upload/image` 返回 url + thumbnail_url + width + height
- [x] `POST /api/v1/upload/voice` 返回 url + duration
- [x] 缩略图宽度 200px，JPEG 格式

---

## Task 2：消息 extra 结构定义 + DTO 扩展 + 常量启用 ✅

**目标**：定义 extra JSON 结构的 Go 类型、扩展消息 DTO、更新常量注释。

### 修改文件

| 文件 | 操作 | 说明 |
|------|------|------|
| backend/go-service/app/dto/im_dto.go | 修改 | SendMessageRequest + MessageDTO 增加 Extra 字段 |
| backend/go-service/app/dto/admin_dto.go | 修改 | 新增管理端消息相关 DTO |
| backend/go-service/app/constants/im.go | 修改 | 消息类型常量注释去掉"预留"标注 |

### 详细步骤

1. `SendMessageRequest` 增加 `Extra string` 字段
2. `MessageDTO` 增加 `Extra *string` 字段（omitempty）
3. `constants/im.go` 消息类型注释更新：去掉"预留"二字
4. 在 `admin_dto.go` 中新增管理端消息 DTO 结构体

### 验收标准

- [x] SendMessageRequest 和 MessageDTO 支持 extra 字段
- [x] 管理端消息 DTO 定义完成
- [x] 常量注释准确反映当前实现状态

---

## Task 3：IM Service 适配富媒体消息 ✅

**目标**：后端 IM 服务支持存储和推送富媒体消息。

### 修改文件

| 文件 | 操作 | 说明 |
|------|------|------|
| backend/go-service/app/im/service/im_service.go | 修改 | SendMessage 存储 extra；预览文案生成 |
| backend/go-service/app/im/handler/event_handler.go | 修改 | im.message.send 解析 extra 字段 |
| backend/go-service/app/im/dao/message_dao.go | 修改 | MessageDTO 转换包含 extra |

### 详细步骤

1. `EventHandler` 中 `im.message.send` 解析 `req.Extra` 并传给 Service
2. `SendMessage` 中创建 Message 时设置 `Extra` 字段
3. 新增 `generateLastMsgPreview(msgType int, content string, extra *string) string` 方法
4. `UpdateLastMessage` 调用预览文案生成方法
5. `buildMessagePushData` 返回的 MessageDTO 包含 extra
6. 历史消息查询 `toMessageDTO` 中包含 extra 字段

### 验收标准

- [x] type=2/3/5 消息能正常存储到数据库，extra 字段不为空
- [x] 推送 im.message.new 包含 extra 数据
- [x] 会话列表预览文案正确（[图片]/[语音 12"]/[文件] xxx.pdf）
- [x] 历史消息接口返回 extra 字段

---

## Task 4：管理端消息 DAO + Service + Controller ✅

**目标**：实现管理端完整的消息管理后端 API。

### 新增文件

| 文件 | 操作 | 说明 |
|------|------|------|
| backend/go-service/app/admin/dao/message_manage_dao.go | 新增 | 消息分页查询、统计聚合 |
| backend/go-service/app/admin/service/message_manage_service.go | 新增 | 消息管理业务逻辑 |
| backend/go-service/app/admin/controller/message_manage_controller.go | 新增 | REST API 处理器 |
| backend/go-service/app/admin/router.go | 修改 | 注册消息管理路由 |
| backend/go-service/app/provider/provider.go | 修改 | Wire 注入新依赖 |

### 详细步骤

1. `message_manage_dao.go`：
   - `GetMessageList`：分页+多条件筛选（keyword/type/sender/conversation/status/time）
   - `GetMessageByID`：单条详情
   - `UpdateMessageStatus`：更新状态（撤回/删除）
   - `GetMessageStats`：统计聚合（总量/今日/类型分布/每日趋势/活跃用户/活跃群组）
2. `message_manage_service.go`：
   - `GetMessageList` / `GetMessageDetail` / `DeleteMessage` / `RecallMessage` / `GetStats`
   - 撤回时推送 WS 事件给相关在线用户
3. `message_manage_controller.go`：五个接口处理器
4. Router 注册路由
5. Wire provider 注册

### 验收标准

- [x] `GET /api/v1/admin/messages` 支持完整筛选
- [x] `GET /api/v1/admin/messages/:id` 返回详细信息
- [x] `DELETE /api/v1/admin/messages/:id` 软删除
- [x] `PUT /api/v1/admin/messages/:id/recall` 撤回+推送
- [x] `GET /api/v1/admin/messages/stats` 返回统计数据

---

## Task 5：消息组件体系 + conversation 页改造 ✅

**目标**：建立消息类型组件体系，改造聊天页面按消息类型渲染不同组件。

### 新增/修改文件

| 文件 | 操作 | 说明 |
|------|------|------|
| frontend/src/components/msg/MsgText.vue | 新增 | 文本消息组件（从 conversation 抽取） |
| frontend/src/components/msg/MsgImage.vue | 新增 | 图片消息组件（网格+预览，初始骨架） |
| frontend/src/components/msg/MsgVoice.vue | 新增 | 语音消息组件（波形+播放，初始骨架） |
| frontend/src/components/msg/MsgFile.vue | 新增 | 文件消息组件（卡片+下载，初始骨架） |
| frontend/src/pages/chat/conversation.vue | 修改 | 引入组件体系，按 type 分支渲染 |
| frontend/src/pages/group/conversation.vue | 修改 | 同上 |

### 详细步骤

1. 从现有 conversation.vue 中提取文本消息渲染逻辑到 `MsgText.vue`
2. 创建 `MsgImage.vue`/`MsgVoice.vue`/`MsgFile.vue` 骨架组件（接收 msg prop，显示占位）
3. conversation.vue 消息列表区域改为根据 `msg.type` 使用 `<component :is="">` 动态组件
4. 确保文本消息渲染效果与改造前一致

### 验收标准

- [x] 文本消息渲染效果不变
- [x] 4 个消息组件均已创建并可接收 msg prop
- [x] conversation 页面按 type 分支渲染

---

## Task 6：图片消息完整流程 ✅

**目标**：实现图片消息从选取到展示的完整流程。

### 新增/修改文件

| 文件 | 操作 | 说明 |
|------|------|------|
| frontend/src/api/file.js | 新增 | 文件上传 API 封装 |
| frontend/src/utils/file.js | 新增 | 文件工具函数 |
| frontend/src/components/msg/MsgImage.vue | 修改 | 完整图片网格+大图预览 |
| frontend/src/components/chat/ImagePreviewer.vue | 新增 | 全屏图片预览 |
| frontend/src/store/chat.js | 修改 | 新增 sendImageMessage |

### 详细步骤

1. `api/file.js`：封装 `uploadImage(file)` 和 `uploadFile(file)` API 调用
2. `utils/file.js`：图片压缩工具、文件大小格式化
3. `MsgImage.vue`：完整实现多图网格布局 + 点击调起 ImagePreviewer
4. `ImagePreviewer.vue`：全屏 Swiper 预览 + 左右滑 + 关闭
5. `chat.js`：`sendImageMessage(conversationId, imageFiles)` → 压缩 → 逐张上传 → 组装 extra → WS 发送

### 验收标准

- [x] 图片选取+压缩+上传+发送全流程通畅
- [x] 聊天界面正确展示图片缩略图（网格布局）
- [x] 点击图片可全屏大图预览
- [x] 多图（2-9张）布局正确

---

## Task 7：语音消息完整流程 ✅

**目标**：实现语音消息录制、发送、展示、播放完整流程。

### 新增/修改文件

| 文件 | 操作 | 说明 |
|------|------|------|
| frontend/src/components/chat/VoiceRecorder.vue | 新增 | 语音录制组件 |
| frontend/src/components/msg/MsgVoice.vue | 修改 | 完整语音播放组件 |
| frontend/src/store/chat.js | 修改 | 新增 sendVoiceMessage |

### 详细步骤

1. `VoiceRecorder.vue`：长按录音 UI + 上滑取消 + 时长显示 + 波形动画
2. `MsgVoice.vue`：播放按钮 + 波形条 + 时长显示 + 播放进度动画
3. 全局单例播放器：同时只有一个语音在播放
4. `chat.js`：`sendVoiceMessage(conversationId, voiceTempPath, duration)` → 上传 → 发送

### 验收标准

- [x] 长按录音 + 松手发送 + 上滑取消
- [x] 录音时长 < 1s 提示太短
- [x] 录音时长 > 60s 自动停止
- [x] 聊天界面显示语音条 + 时长
- [x] 点击播放/暂停
- [x] 同一时间只有一个语音播放

---

## Task 8：文件消息完整流程 ✅

**目标**：实现文件消息选取、上传、发送、展示、下载/预览完整流程。

### 新增/修改文件

| 文件 | 操作 | 说明 |
|------|------|------|
| frontend/src/components/msg/MsgFile.vue | 修改 | 完整文件卡片组件 |
| frontend/src/store/chat.js | 修改 | 新增 sendFileMessage |
| frontend/src/utils/file.js | 修改 | 文件类型图标映射 + 大小格式化 |

### 详细步骤

1. `MsgFile.vue`：文件图标 + 文件名 + 大小 + 下载/预览按钮
2. 文件类型图标映射（PDF/Word/Excel/PPT/压缩包/图片/音频/通用）
3. 下载功能：H5 用 `window.open`，APP 用 `uni.downloadFile`
4. 预览功能：PDF/图片支持在线预览
5. `chat.js`：`sendFileMessage(conversationId, file)` → 校验大小 → 上传 → 发送

### 验收标准

- [x] 文件选择 + 大小校验（50MB）+ 上传 + 发送
- [x] 聊天界面显示文件卡片（图标+名称+大小）
- [x] 点击下载文件
- [x] PDF/图片可在线预览

---

## Task 9：输入栏改造 ✅

**目标**：改造聊天输入栏，新增"+"面板和麦克风按钮。

### 新增/修改文件

| 文件 | 操作 | 说明 |
|------|------|------|
| frontend/src/components/chat/MorePanel.vue | 新增 | "+"展开面板 |
| frontend/src/pages/chat/conversation.vue | 修改 | 输入栏集成"+"和麦克风 |
| frontend/src/pages/group/conversation.vue | 修改 | 同上 |

### 详细步骤

1. `MorePanel.vue`：网格布局面板，包含"图片"和"文件"两个入口
2. 输入栏右侧添加"+"按钮，点击展开/收起 MorePanel
3. 输入栏左侧添加麦克风按钮，点击切换为语音录制模式
4. 语音模式下隐藏文本输入框，显示 VoiceRecorder
5. 统一发送流程：文本/图片/语音/文件各走对应的 store 方法

### 验收标准

- [x] "+"按钮展开面板，包含图片和文件入口
- [x] 麦克风按钮切换语音录制模式
- [x] 各类型消息发送流程统一
- [x] 单聊和群聊输入栏行为一致

---

## Task 10：管理端消息列表页 ✅

**目标**：实现管理端消息管理列表页面。

### 新增文件

| 文件 | 操作 | 说明 |
|------|------|------|
| admin/src/views/message/list.vue | 新增 | 消息列表页 |
| admin/src/api/message.js | 新增 | 消息管理 API |

### 详细步骤

1. 顶部筛选区：关键词搜索 + 类型下拉 + 发送者 + 时间范围 + 状态
2. 表格列：ID / 发送者 / 类型 / 内容预览 / 状态 / 时间 / 操作
3. 操作列：查看详情 / 撤回 / 删除
4. 分页组件
5. 消息详情弹窗：展示完整内容 + extra 渲染（图片缩略图 / 语音播放 / 文件链接）

### 验收标准

- [x] 完整筛选功能
- [x] 分页正确
- [x] 撤回和删除操作正常
- [x] 消息详情弹窗内容完整

---

## Task 11：管理端消息统计页 ✅

**目标**：实现管理端消息统计仪表板。

### 新增文件

| 文件 | 操作 | 说明 |
|------|------|------|
| admin/src/views/message/stats.vue | 新增 | 消息统计仪表板 |

### 详细步骤

1. 顶部统计卡片：总消息数 + 今日消息数
2. 消息趋势折线图（近 7/30 天切换）
3. 消息类型分布饼图
4. 活跃用户排行水平柱状图（Top 10）
5. 活跃群组排行水平柱状图（Top 10）
6. 图表主题色与系统一致（Primary #2563EB）

### 验收标准

- [x] 4 个统计图表正确渲染
- [x] 7/30 天切换正常
- [x] 数据与 stats API 返回一致
- [x] 响应式布局

---

## Task 12：管理端路由 + Store + API ✅

**目标**：管理端消息管理模块基础设施。

### 修改文件

| 文件 | 操作 | 说明 |
|------|------|------|
| admin/src/router/index.js | 修改 | 注册消息管理路由 |
| admin/src/api/message.js | 修改 | 完善 API 封装（如未在 Task 10 完成） |

### 详细步骤

1. 路由配置：`/message/list` + `/message/stats`
2. 侧边栏菜单：新增"消息管理"分组
3. API 封装：确保所有管理端消息 API 已封装

### 验收标准

- [x] 侧边栏正确显示消息管理菜单
- [x] 路由跳转正常
- [x] API 封装完整

---

## Task 13：群聊适配 + 统一测试 ✅

**目标**：确保群聊页面与单聊页面的富媒体消息功能一致，统一测试。

### 修改文件

| 文件 | 操作 | 说明 |
|------|------|------|
| frontend/src/pages/group/conversation.vue | 修改 | 确认组件集成完整 |

### 详细步骤

1. 群聊 conversation.vue 确认所有消息组件和输入栏改造与单聊一致
2. 测试单聊图片/语音/文件消息发送接收
3. 测试群聊图片/语音/文件消息发送接收
4. 测试会话列表预览文案
5. 测试管理端消息列表和统计
6. 修复发现的问题

### 验收标准

- [x] 单聊+群聊富媒体消息全流程正常
- [x] 会话列表预览文案正确
- [x] 管理端功能正常

---

## Task 14：代码审查 + Playwright 测试 + 文档更新 ✅

**目标**：最终质量保障和文档同步。

### 详细步骤

1. 使用 `code-reviewer` 子代理进行代码审查
2. 使用 Playwright MCP 进行浏览器自动化测试
3. 修复审查和测试中发现的问题
4. 更新项目文档：
   - `docs/progress/CURRENT_STATUS.md`
   - `docs/plans/2026-03-04-phase2d-design.md`（状态更新）
   - `docs/plans/2026-03-04-phase2d-implementation.plan.md`（状态更新）
   - `docs/api/frontend/im.md`（新增上传 API）
   - `docs/api/admin/message.md`（新增管理端消息 API 文档）
   - `docs/architecture/system-architecture.md`（更新模块描述）
   - `.cursor/rules/project-context.mdc`（更新进度）
5. Git commit + push

### 验收标准

- [x] 代码审查通过
- [x] Playwright 测试通过
- [x] 所有文档同步更新
