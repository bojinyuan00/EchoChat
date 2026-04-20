# Phase 2d 设计文档：消息类型扩展

> **状态：** ✅ 已完成
> **分支：** `feature/phase2d-message-types`
> **前置依赖：** Phase 2c 全部完成（群聊与已读回执）
> **最后更新：** 2026-03-03

---

## 一、设计目标

基于 Phase 2b/2c 的单聊+群聊文本消息基础设施，扩展三种富媒体消息类型和管理端消息管理功能。

**核心交付物：**
- 图片消息（type=2）：前端压缩+多图发送（最多9张）、后端生成 JPEG 缩略图、聊天内缩略图展示+大图预览
- 语音消息（type=3）：微信风格按住录音/松手发送/上滑取消、最长60秒、时长波形条展示+播放
- 文件消息（type=5）：任意文件上传（最大50MB）、类型图标+大小展示、点击下载+在线预览（PDF/图片）
- 文件上传服务增强：大小限制提升至50MB、图片专用上传（缩略图生成）、语音专用上传（格式+时长校验）
- 管理端消息管理：消息列表（完整筛选搜索）+ 消息统计仪表板（趋势图+类型分布+活跃排行）+ 敏感消息删除/撤回

**不包含（留待后续阶段）：**
- 视频消息（type=4，录制/压缩/缩略图复杂度高）
- 表情包 / 自定义贴纸
- 消息转发 / 合并转发
- 消息引用 / 回复

---

## 二、需求决策记录

### 2.1 消息类型决策

| 决策项 | 选择 | 理由 |
|--------|------|------|
| 消息类型范围 | 图片+语音+文件（无视频） | 视频录制/压缩/缩略图复杂度高，留后续阶段 |
| 多图策略 | 单条消息包含多图（最多9张） | 微信风格，extra JSON 存图片列表 |
| 图片缩略图 | 后端 Go 图片库生成 | 保证各端一致性，前端仅做发送前压缩 |
| 图片格式 | 统一转 JPEG | 兼容性最好，体积适中 |
| 图片压缩 | 前端发送前压缩（宽>1920px 等比缩放，质量80%） | 减少上传流量 |
| 缩略图尺寸 | 宽度不超过 200px，等比缩放，JPEG quality 80 | 列表加载快，质量可接受 |

### 2.2 语音消息决策

| 决策项 | 选择 | 理由 |
|--------|------|------|
| 交互方式 | 微信风格：按住录音/松手发送/上滑取消 | 用户认知成本低 |
| 录音时长 | 最长 60 秒 | 微信标准 |
| 录音格式 | MP3 | 跨端兼容性最好 |
| 录音技术 | uni-app `uni.getRecorderManager()` | uni-app 原生 API，多端兼容 |

### 2.3 文件消息决策

| 决策项 | 选择 | 理由 |
|--------|------|------|
| 大小限制 | 50MB（上调原 10MB） | 支持 PPT/中等大小文件 |
| 文件能力 | 上传+类型图标+大小显示+下载+在线预览（PDF/图片） | 完整用户体验 |
| 文件类型限制 | 不限制（后端仅做大小校验） | 通用文件传输场景 |

### 2.4 上传与发送流程决策

| 决策项 | 选择 | 理由 |
|--------|------|------|
| 上传流程 | 先上传后发送：前端 REST 上传 → 获取 URL → WS 发消息 | 简单可靠，解耦文件上传和消息发送 |
| 上传接口 | 按类型分离：通用/图片/语音三个接口 | 图片需要缩略图、语音需要时长校验，逻辑各异 |

### 2.5 UI/UX 决策

| 决策项 | 选择 | 理由 |
|--------|------|------|
| 会话列表预览 | 方括号标记：`[图片]`、`[语音 12"]`、`[文件] report.pdf` | 微信风格，用户习惯 |
| 图片展示 | 聊天内缩略图（网格）+ 点击大图预览（Swiper） | 标准 IM 体验 |
| 语音展示 | 时长+波形条+播放按钮 | 微信风格 |
| 文件展示 | 文件图标+文件名+大小+下载/预览按钮 | 清晰直观 |
| 表情包/贴纸 | 不做 | 专注核心媒体消息 |

### 2.6 管理端决策

| 决策项 | 选择 | 理由 |
|--------|------|------|
| 搜索筛选 | 完整：关键词+类型+发送者+会话/群+时间+状态 | 满足运营需求 |
| 统计维度 | 完整：趋势图+类型分布+活跃用户排行+群活跃度 | 数据运营可视化 |
| 图表库 | ECharts | admin 项目基于 Vue 3 + Element Plus，ECharts 集成良好 |

---

## 三、架构设计

### 3.1 模块目录结构

```
backend/go-service/
├── app/
│   ├── file/                          # 文件上传模块（增强）
│   │   ├── controller/
│   │   │   └── file_controller.go     # 新增 UploadImage / UploadVoice 接口
│   │   ├── service/
│   │   │   └── file_service.go        # 新增 UploadWithThumbnail / UploadVoice 方法
│   │   ├── router.go                  # 新增 /upload/image 和 /upload/voice 路由
│   │   └── provider.go
│   ├── im/                            # 即时通讯模块（适配）
│   │   ├── service/
│   │   │   └── im_service.go          # SendMessage 适配 extra + 预览文案
│   │   └── handler/
│   │       └── event_handler.go       # im.message.send 适配 extra 字段
│   ├── admin/                         # 管理端（新增消息管理）
│   │   ├── controller/
│   │   │   └── message_manage_controller.go  # [新增]
│   │   ├── service/
│   │   │   └── message_manage_service.go     # [新增]
│   │   ├── dao/
│   │   │   └── message_manage_dao.go         # [新增]
│   │   └── router.go                         # 注册消息管理路由
│   ├── dto/
│   │   ├── im_dto.go                  # 扩展 SendMessageRequest + MessageDTO 的 extra
│   │   └── admin_dto.go               # 新增管理端消息 DTO
│   └── constants/
│       └── im.go                      # 消息类型常量注释更新（2/3/5 不再标注"预留"）
├── frontend/src/
│   ├── api/
│   │   └── file.js                    # [新增] 文件上传 API 封装
│   ├── components/
│   │   ├── msg/                       # [新增] 消息类型组件
│   │   │   ├── MsgText.vue            # 文本消息组件
│   │   │   ├── MsgImage.vue           # 图片消息组件（网格+大图预览）
│   │   │   ├── MsgVoice.vue           # 语音消息组件（波形条+播放）
│   │   │   └── MsgFile.vue            # 文件消息组件（卡片+下载）
│   │   └── chat/                      # [新增] 聊天辅助组件
│   │       ├── MorePanel.vue          # "+"展开面板（图片/文件入口）
│   │       ├── VoiceRecorder.vue      # 语音录制组件
│   │       └── ImagePreviewer.vue     # 图片全屏预览
│   ├── pages/
│   │   ├── chat/conversation.vue      # [修改] 引入消息组件+输入栏改造
│   │   └── group/conversation.vue     # [修改] 同上
│   ├── store/
│   │   └── chat.js                    # [修改] 扩展发送方法
│   └── utils/
│       └── file.js                    # [新增] 文件工具（压缩/格式判断/大小格式化）
├── admin/src/
│   ├── api/
│   │   └── message.js                 # [新增] 管理端消息 API
│   ├── views/
│   │   └── message/                   # [新增] 消息管理模块
│   │       ├── list.vue               # 消息列表（筛选+分页+操作）
│   │       └── stats.vue              # 消息统计仪表板
│   └── router/
│       └── index.js                   # 注册消息管理路由
```

### 3.2 跨模块依赖关系

```
file.FileService ← im.IMService (通过接口注入，暂不直接依赖)
                              ↓
                    im.IMService → convDAO/msgDAO (存储消息 + extra)
                              ↓
                    im.EventHandler → im.IMService (WS 事件调用)
                              ↓
                    admin.MessageManageService → im.MessageDAO (查询消息)
                                              → auth.UserDAO (查询发送者)
```

消息发送流程中，**文件上传与消息发送是解耦的**：
1. 前端通过 REST API 上传文件到 MinIO，获取 URL
2. 前端通过 WS 发送消息，payload 中携带 `extra` 字段（含文件 URL）
3. 后端 IM Service 直接存储，不再访问 file 模块

---

## 四、消息 extra 字段 JSON 结构设计

`im_messages.extra` 字段类型为 JSONB（已存在），按消息类型存储不同元数据。

### 4.1 图片消息（type=2）

```json
{
  "images": [
    {
      "url": "http://localhost:9000/echochat/uploads/2026/03/05/abc.jpg",
      "thumbnail_url": "http://localhost:9000/echochat/uploads/2026/03/05/abc_thumb.jpg",
      "width": 1920,
      "height": 1080,
      "size": 245760,
      "file_name": "photo.jpg"
    },
    {
      "url": "http://localhost:9000/echochat/uploads/2026/03/05/def.jpg",
      "thumbnail_url": "http://localhost:9000/echochat/uploads/2026/03/05/def_thumb.jpg",
      "width": 1280,
      "height": 720,
      "size": 180000,
      "file_name": "photo2.jpg"
    }
  ]
}
```

### 4.2 语音消息（type=3）

```json
{
  "voice": {
    "url": "http://localhost:9000/echochat/uploads/2026/03/05/voice_xxx.mp3",
    "duration": 12,
    "size": 48000,
    "file_name": "voice_20260305_120000.mp3"
  }
}
```

### 4.3 文件消息（type=5）

```json
{
  "file": {
    "url": "http://localhost:9000/echochat/uploads/2026/03/05/doc_xxx.pdf",
    "file_name": "年度报告.pdf",
    "size": 1048576,
    "mime_type": "application/pdf",
    "ext": ".pdf"
  }
}
```

---

## 五、API 设计

### 5.1 文件上传 REST API（扩展）

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| POST | /api/v1/upload | 需认证 | 通用文件上传（最大50MB） |
| POST | /api/v1/upload/image | 需认证 | 图片上传（压缩+生成缩略图，返回双 URL） |
| POST | /api/v1/upload/voice | 需认证 | 语音上传（校验格式+时长<=60s） |

#### POST /api/v1/upload/image

**请求**：multipart/form-data，字段名 `file`

**成功响应**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "url": "http://localhost:9000/echochat/uploads/2026/03/05/xxx.jpg",
    "thumbnail_url": "http://localhost:9000/echochat/uploads/2026/03/05/xxx_thumb.jpg",
    "file_name": "photo.jpg",
    "size": 245760,
    "width": 1920,
    "height": 1080,
    "mime_type": "image/jpeg"
  }
}
```

#### POST /api/v1/upload/voice

**请求**：multipart/form-data，字段名 `file`

**成功响应**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "url": "http://localhost:9000/echochat/uploads/2026/03/05/voice_xxx.mp3",
    "file_name": "voice_20260305.mp3",
    "size": 48000,
    "duration": 12,
    "mime_type": "audio/mpeg"
  }
}
```

### 5.2 管理端消息管理 REST API（新增）

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | /api/v1/admin/messages | admin | 消息列表（分页+多条件筛选） |
| GET | /api/v1/admin/messages/:id | admin | 消息详情（含发送者信息） |
| DELETE | /api/v1/admin/messages/:id | admin | 删除消息 |
| PUT | /api/v1/admin/messages/:id/recall | admin | 管理员撤回消息 |
| GET | /api/v1/admin/messages/stats | admin | 消息统计数据 |

#### GET /api/v1/admin/messages

**查询参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| keyword | string | 搜索关键词（模糊匹配 content） |
| type | int | 消息类型筛选（1/2/3/5/10） |
| sender_id | int64 | 发送者 ID |
| conversation_id | int64 | 会话 ID |
| status | int | 消息状态（1=正常/2=已撤回/3=已删除） |
| start_time | string | 开始时间（YYYY-MM-DD） |
| end_time | string | 结束时间（YYYY-MM-DD） |
| page | int | 页码（默认1） |
| page_size | int | 每页条数（默认20） |

**成功响应**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": 1,
        "conversation_id": 5,
        "sender_id": 4,
        "sender_nickname": "张三",
        "sender_avatar": "",
        "type": 2,
        "type_label": "图片",
        "content": "",
        "extra": { "images": [...] },
        "status": 1,
        "status_label": "正常",
        "created_at": "2026-03-05 10:30:00"
      }
    ],
    "total": 1000,
    "page": 1,
    "page_size": 20
  }
}
```

#### GET /api/v1/admin/messages/stats

**查询参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| days | int | 统计天数（默认7，最大90） |

**成功响应**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total_count": 12580,
    "today_count": 350,
    "type_distribution": [
      { "type": 1, "label": "文本", "count": 10000 },
      { "type": 2, "label": "图片", "count": 1500 },
      { "type": 3, "label": "语音", "count": 500 },
      { "type": 5, "label": "文件", "count": 300 },
      { "type": 10, "label": "系统消息", "count": 280 }
    ],
    "daily_trend": [
      { "date": "2026-03-01", "count": 280 },
      { "date": "2026-03-02", "count": 320 }
    ],
    "active_users": [
      { "user_id": 4, "nickname": "张三", "count": 120 }
    ],
    "active_groups": [
      { "group_id": 1, "name": "技术讨论群", "count": 500 }
    ]
  }
}
```

### 5.3 WebSocket 事件变更

**`im.message.send` 请求扩展**：

```json
{
  "event": "im.message.send",
  "data": {
    "conversation_id": 8,
    "type": 2,
    "content": "",
    "client_msg_id": "xxx",
    "extra": "{\"images\":[{\"url\":\"...\",\"thumbnail_url\":\"...\",\"width\":1920,\"height\":1080,\"size\":245760,\"file_name\":\"photo.jpg\"}]}"
  }
}
```

**`im.message.new` 推送扩展**：

推送数据增加 `extra` 字段（JSON string），前端解析后根据 `type` 渲染对应组件。

**`im.message.send.ack` 响应扩展**：

ACK 的 `data`（MessageDTO）增加 `extra` 字段。

---

## 六、核心业务流程

### 6.1 图片消息发送流程

1. 用户点击"+"面板中的"图片"，调用 `uni.chooseImage` 选择图片（最多9张）
2. 前端对每张图片进行压缩（宽>1920px 等比缩放，质量80%）
3. 逐张调用 `POST /api/v1/upload/image` 上传
4. 后端接收图片 → 生成 JPEG 缩略图（宽200px，quality 80）→ 原图+缩略图存 MinIO
5. 返回 `{ url, thumbnail_url, width, height, size, file_name, mime_type }`
6. 所有图片上传完成后，前端组装 `extra.images` 数组
7. 通过 WS 发送 `im.message.send`：`type=2`，`content=""`，`extra` 为 JSON 字符串
8. 后端存储消息，`UpdateLastMessage` 预览文案为 `[图片]` 或 `[图片x3]`
9. 推送 `im.message.new` 给接收方，携带 `extra`

### 6.2 语音消息发送流程

1. 用户长按麦克风按钮，调用 `uni.getRecorderManager().start({ format: 'mp3', duration: 60000 })`
2. 录音过程中显示录音动画（时长 + 波形），上滑显示"取消发送"
3. 松手停止录音 → 录音回调返回临时文件路径
4. 时长 < 1秒：提示"录音时间太短"并丢弃
5. 调用 `POST /api/v1/upload/voice` 上传语音文件
6. 后端校验格式和时长（<=60s）→ 存 MinIO → 返回 `{ url, duration, size, file_name }`
7. 前端组装 `extra.voice` 对象
8. 通过 WS 发送：`type=3`，`content=""`，`extra` 为 JSON 字符串
9. 后端存储，预览文案 `[语音 12"]`

### 6.3 文件消息发送流程

1. 用户点击"+"面板中的"文件"，调用 `uni.chooseFile` 或 `uni.chooseMessageFile`
2. 校验文件大小（<=50MB）
3. 调用 `POST /api/v1/upload` 通用上传接口
4. 返回 `{ url, file_name, size }`
5. 前端补充 `mime_type` 和 `ext`，组装 `extra.file` 对象
6. 通过 WS 发送：`type=5`，`content=""`，`extra` 为 JSON 字符串
7. 后端存储，预览文案 `[文件] report.pdf`

### 6.4 管理员撤回消息流程

1. 管理员在消息列表选中消息 → 点击"撤回"
2. 调用 `PUT /api/v1/admin/messages/:id/recall`
3. 后端将消息 status 改为 2（已撤回）
4. 推送 `im.message.recalled` 给相关在线用户
5. 会话最后消息预览更新为"管理员撤回了一条消息"

---

## 七、会话列表预览文案规则

在 `IMService` 中 `UpdateLastMessage` 时根据消息类型生成预览文案：

| type | content 字段用途 | last_msg_content 预览 |
|------|-----------------|----------------------|
| 1（文本） | 消息正文 | 消息正文（截断） |
| 2（图片） | 空字符串 | `[图片]` / `[图片x3]`（多图标数量） |
| 3（语音） | 空字符串 | `[语音 12"]`（duration 从 extra 解析） |
| 5（文件） | 空字符串 | `[文件] report.pdf`（file_name 从 extra 解析） |
| 10（系统） | 系统消息正文 | 系统消息正文 |

---

## 八、前端页面规划

### 8.1 新增组件（前台用户端）

| 组件 | 路径 | 说明 |
|------|------|------|
| MsgText.vue | components/msg/ | 文本消息组件（从 conversation 提取） |
| MsgImage.vue | components/msg/ | 图片消息：1张=大图，2-4张=2列网格，5-9张=3列网格；点击放大 |
| MsgVoice.vue | components/msg/ | 语音消息：播放按钮+波形条+时长（宽度按时长缩放） |
| MsgFile.vue | components/msg/ | 文件消息：文件图标+文件名+大小+下载/预览按钮 |
| MorePanel.vue | components/chat/ | 输入栏"+"展开面板：图片、文件两个入口 |
| VoiceRecorder.vue | components/chat/ | 语音录制：长按录音+上滑取消+时长显示+波形动画 |
| ImagePreviewer.vue | components/chat/ | 图片全屏预览：Swiper 左右滑+双指缩放 |

### 8.2 修改页面（前台用户端）

| 页面 | 修改内容 |
|------|----------|
| pages/chat/conversation.vue | 消息列表按 type 渲染不同组件；输入栏增加"+"和麦克风按钮 |
| pages/group/conversation.vue | 同上 |
| store/chat.js | 扩展 sendMessage 支持 extra；新增 sendImageMessage/sendVoiceMessage/sendFileMessage |

### 8.3 新增页面（管理端）

| 页面 | 路径 | 说明 |
|------|------|------|
| 消息列表 | views/message/list.vue | 多条件筛选+分页表格+撤回/删除操作 |
| 消息统计 | views/message/stats.vue | ECharts 图表：趋势折线图+类型饼图+活跃排行柱状图 |

---

## 九、DTO 扩展规划

### 9.1 发送消息请求扩展

```go
// SendMessageRequest 发送消息请求（WS 事件 im.message.send 的 data 字段）
type SendMessageRequest struct {
    ConversationID int64   `json:"conversation_id"`
    TargetUserID   int64   `json:"target_user_id"`
    Type           int     `json:"type"`
    Content        string  `json:"content"`
    ClientMsgID    string  `json:"client_msg_id"`
    AtUserIDs      []int64 `json:"at_user_ids"`
    Extra          string  `json:"extra"`           // [Phase 2d 新增] 扩展数据 JSON 字符串
}
```

### 9.2 消息 DTO 扩展

```go
// MessageDTO 消息传输对象（返回给前端）
type MessageDTO struct {
    ID             int64   `json:"id"`
    ConversationID int64   `json:"conversation_id"`
    SenderID       int64   `json:"sender_id"`
    Type           int     `json:"type"`
    Content        string  `json:"content"`
    Extra          *string `json:"extra,omitempty"` // [Phase 2d 新增] 扩展数据 JSON
    Status         int     `json:"status"`
    ClientMsgID    string  `json:"client_msg_id"`
    AtUserIDs      []int64 `json:"at_user_ids"`
    CreatedAt      string  `json:"created_at"`
}
```

### 9.3 文件上传结果 DTO 扩展

```go
// ImageUploadResult 图片上传结果
type ImageUploadResult struct {
    URL          string `json:"url"`
    ThumbnailURL string `json:"thumbnail_url"`
    FileName     string `json:"file_name"`
    Size         int64  `json:"size"`
    Width        int    `json:"width"`
    Height       int    `json:"height"`
    MimeType     string `json:"mime_type"`
}

// VoiceUploadResult 语音上传结果
type VoiceUploadResult struct {
    URL      string `json:"url"`
    FileName string `json:"file_name"`
    Size     int64  `json:"size"`
    Duration int    `json:"duration"`
    MimeType string `json:"mime_type"`
}
```

### 9.4 管理端消息 DTO

```go
// AdminMessageListRequest 管理端消息列表查询
type AdminMessageListRequest struct {
    Keyword        string `form:"keyword"`
    Type           *int   `form:"type"`
    SenderID       *int64 `form:"sender_id"`
    ConversationID *int64 `form:"conversation_id"`
    Status         *int   `form:"status"`
    StartTime      string `form:"start_time"`
    EndTime        string `form:"end_time"`
    Page           int    `form:"page"`
    PageSize       int    `form:"page_size"`
}

// AdminMessageDTO 管理端消息条目
type AdminMessageDTO struct {
    ID             int64   `json:"id"`
    ConversationID int64   `json:"conversation_id"`
    SenderID       int64   `json:"sender_id"`
    SenderNickname string  `json:"sender_nickname"`
    SenderAvatar   string  `json:"sender_avatar"`
    Type           int     `json:"type"`
    TypeLabel      string  `json:"type_label"`
    Content        string  `json:"content"`
    Extra          *string `json:"extra,omitempty"`
    Status         int     `json:"status"`
    StatusLabel    string  `json:"status_label"`
    CreatedAt      string  `json:"created_at"`
}

// AdminMessageStatsRequest 管理端消息统计请求
type AdminMessageStatsRequest struct {
    Days int `form:"days"` // 统计天数（默认7，最大90）
}

// AdminMessageStatsResponse 管理端消息统计响应
type AdminMessageStatsResponse struct {
    TotalCount       int64                `json:"total_count"`
    TodayCount       int64                `json:"today_count"`
    TypeDistribution []TypeDistItem       `json:"type_distribution"`
    DailyTrend       []DailyTrendItem     `json:"daily_trend"`
    ActiveUsers      []ActiveUserItem     `json:"active_users"`
    ActiveGroups     []ActiveGroupItem    `json:"active_groups"`
}

type TypeDistItem struct {
    Type  int    `json:"type"`
    Label string `json:"label"`
    Count int64  `json:"count"`
}

type DailyTrendItem struct {
    Date  string `json:"date"`
    Count int64  `json:"count"`
}

type ActiveUserItem struct {
    UserID   int64  `json:"user_id"`
    Nickname string `json:"nickname"`
    Count    int64  `json:"count"`
}

type ActiveGroupItem struct {
    GroupID int64  `json:"group_id"`
    Name    string `json:"name"`
    Count   int64  `json:"count"`
}
```

---

## 十、技术要点

### 10.1 Go 缩略图生成

- 依赖：`github.com/disintegration/imaging`
- 流程：`imaging.Open(file)` → `imaging.Resize(img, 200, 0, imaging.Lanczos)` → `imaging.Encode(buf, thumb, imaging.JPEG, imaging.JPEGQuality(80))`
- 缩略图文件名：原始 UUID + `_thumb` 后缀，如 `abc123_thumb.jpg`
- 原图同时进行 JPEG 转码（如果不是 JPEG 格式）

### 10.2 前端图片压缩

- H5 平台使用 Canvas 压缩：创建 Canvas → drawImage → toBlob('image/jpeg', 0.8)
- 小程序/App 平台使用 `uni.compressImage({ src, quality: 80 })`
- 压缩策略：宽度>1920px 等比缩放至 1920px，质量 80%

### 10.3 语音录制

- uni-app RecorderManager API
- `start({ format: 'mp3', duration: 60000, sampleRate: 44100, numberOfChannels: 1 })`
- 监听 `onStop` 获取录音文件临时路径
- 时长计算：回调中的 `duration` 字段（毫秒），前端转秒显示

### 10.4 语音播放

- uni-app InnerAudioContext API
- `uni.createInnerAudioContext()` → `src = voiceUrl` → `play()`
- 全局单例：同一时间只允许一个语音播放
- 播放状态：空闲 → 加载中 → 播放中 → 播放完成

### 10.5 文件下载与预览

- 下载：`uni.downloadFile({ url })` → `uni.saveFile()` 或直接 `window.open(url)` (H5)
- 预览：PDF 和图片通过 `uni.openDocument` 或浏览器新标签打开
- 文件类型图标映射：根据扩展名匹配图标（PDF/Word/Excel/PPT/ZIP/图片/音频/通用）

### 10.6 管理端 ECharts 集成

- `npm install echarts` 安装
- 使用 Vue 3 Composition API + `onMounted` 初始化
- 响应式：`window.resize` 时 `chart.resize()`
- 主题色与系统保持一致（Primary #2563EB）

---

## 十一、补充设计细节

### 11.1 消息内容安全

- 文件上传不做内容安全审查（留后续阶段）
- 管理端提供人工审查和撤回能力
- 上传文件类型不限制，但大小限制 50MB

### 11.2 多图布局规则

| 图片数量 | 布局 | 单图最大宽度 |
|----------|------|-------------|
| 1 张 | 自适应宽度（最大 500rpx） | 500rpx |
| 2 张 | 2列1行 | 240rpx |
| 3 张 | 3列1行 | 160rpx |
| 4 张 | 2列2行 | 240rpx |
| 5-6 张 | 3列2行 | 160rpx |
| 7-9 张 | 3列3行 | 160rpx |

### 11.3 文件类型图标映射

| 扩展名 | 图标 | 颜色 |
|--------|------|------|
| .pdf | PDF 图标 | #E53E3E |
| .doc/.docx | Word 图标 | #2B6CB0 |
| .xls/.xlsx | Excel 图标 | #276749 |
| .ppt/.pptx | PPT 图标 | #C05621 |
| .zip/.rar/.7z | 压缩包图标 | #6B46C1 |
| .jpg/.png/.gif | 图片图标 | #38A169 |
| .mp3/.wav/.aac | 音频图标 | #D69E2E |
| 其他 | 通用文件图标 | #718096 |

### 11.4 语音波形条设计

- 宽度：按时长线性缩放，最小 120rpx（1秒），最大 500rpx（60秒）
- 高度：固定 60rpx
- 波形：使用 CSS 模拟随机高度的竖条（5-6 个竖条）
- 播放时：从左到右逐条变色动画
