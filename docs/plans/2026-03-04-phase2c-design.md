# Phase 2c 设计文档：群聊与已读回执

> **状态：** ✅ 已完成（含代码审查修复 14 项 + 浏览器测试修复 21 项）
> **分支：** `feature/phase2c-group-read-receipt`
> **前置依赖：** Phase 2b 全部完成（单聊即时通讯）
> **最后更新：** 2026-03-04

---

## 一、设计目标

基于 Phase 2b 的单聊 IM 基础设施，实现群聊核心功能和已读回执系统，同时引入 MinIO 文件存储服务作为基础设施扩展。

**核心交付物：**
- 群聊管理（创建/加入/退出/解散/搜索/角色管理/禁言/公告）
- 群消息收发（复用 im.message.* 事件 + @提醒 + 管理员撤回）
- 已读回执（单聊会话级 + 群聊消息级 + 实时推送 + 详情查看）
- MinIO 文件存储（Docker 容器 + Go SDK + 通用上传 API）
- 管理端群聊管理（群列表/详情/解散/移除成员）
- 前端 9 个新页面 + 群聊 Store + API 封装

**不包含（留待后续阶段）：**
- 图片/语音/文件消息
- 管理端消息管理
- 消息类型扩展

---

## 二、需求决策记录

### 2.1 群聊功能决策

| 决策项 | 选定方案 | 说明 |
|--------|----------|------|
| 创建方式 | 选好友 + 搜索用户 ID 拉人 | 非好友也可拉入群 |
| 成员上限 | 200 人（默认） | im_groups.max_members 可配置 |
| 角色体系 | 三级：群主(2) + 管理员(1) + 普通成员(0) | 微信模式 |
| 管理操作 | 全功能 | 改群名/头像/公告、拉人/踢人、禁言/全体禁言、转让群主、解散、设管理员 |
| 入群方式 | 被拉入（无需审批）+ 主动申请（需审批） | 群主/管理员审批 |
| 退出机制 | 主动退出 + 显示系统消息 | "XX 退出/加入/被移出了群聊" |
| 历史消息 | 钉钉模式 | 新成员可查看所有历史消息 |
| 群搜索 | 全局搜索公开群 + 自己的群 | 默认可搜索，群主可设为不可搜索 |
| 群头像 | 默认图标 + 群主手动上传 | 使用 MinIO 存储 |
| 群昵称 | 成员可设置群内昵称 | 仅在该群内显示 |
| 免打扰 | 微信模式 | 仍计未读（灰色数字），不推送通知 |
| @提醒 | @某人 + @所有人 | 所有成员可 @所有人，输入 @ 弹出成员选择 |
| @通知 | 会话列表标记 | 显示 "[N条] @了我" |
| 正在输入 | 群聊不显示 | 仅单聊保留 |
| 消息撤回 | 成员 2 分钟，管理员无时限 | 群主/管理员可撤回任何人消息 |
| 撤回展示 | 区分操作者 | "管理员 XX 撤回了 YY 的一条消息" |
| 群公告 | 需要 | 群主/管理员发布，发布时推送系统消息通知全员 |

### 2.2 已读回执决策

| 决策项 | 选定方案 | 说明 |
|--------|----------|------|
| 单聊粒度 | 会话级别 | 复用 last_read_msg_id，消息 ID ≤ 该值即已读 |
| 群聊粒度 | 消息级别 | im_message_reads 表记录每人每条消息的已读 |
| 展示方式 | 钉钉模式 | 单聊 "已读/未读"，群聊 "X人已读" 可点击查看列表 |
| 存储方案 | 纯 PostgreSQL | im_message_reads 表（联合主键 message_id + user_id） |
| 推送策略 | 差异化 | 单聊实时推送已读变化，群聊仅推送已读计数变化 |
| 详情查看 | REST 按需拉取 | 点击 "X人已读" 时 GET /api/v1/im/messages/:id/reads |

### 2.3 基础设施决策

| 决策项 | 选定方案 | 说明 |
|--------|----------|------|
| 文件存储 | MinIO（Docker 镜像） | 通用上传 API，后续消息类型扩展可复用 |
| 前端架构 | 群聊独立页面体系 | 不复用单聊页面，完全独立 |
| 会话列表 | 分 Tab | 单聊 Tab + 群聊 Tab |
| WS 事件 | 消息复用 + 管理独立 | im.message.* 复用，group.* 新增 |
| 消息搜索 | 统一全局搜索 | 单聊 + 群聊消息在同一搜索结果中 |
| 分支策略 | 从 phase2b 拉新分支 | feature/phase2c-group-read-receipt |
| 管理端 | 群列表 + 详情 + 解散 + 移除 | 4 个管理 API + 2 个前端页面 |

---

## 三、架构设计

### 3.1 新增模块

| 模块 | 位置 | 职责 |
|------|------|------|
| group | app/group/ | 群聊管理（创建/加入/退出/角色/禁言/搜索/审批） |
| file | app/file/ | 文件上传（MinIO SDK 封装 + 通用上传 API） |
| storage | pkg/storage/ | MinIO 客户端初始化 + 配置 |

### 3.2 模块目录结构

```
backend/go-service/
├── app/
│   ├── group/                        # [新增] 群聊管理模块
│   │   ├── controller/
│   │   │   └── group_controller.go   # REST API 控制器（~15 个接口）
│   │   ├── service/
│   │   │   └── group_service.go      # 群创建/管理/审批/搜索业务逻辑
│   │   ├── dao/
│   │   │   └── group_dao.go          # im_groups + 成员角色 + 入群申请 CRUD
│   │   ├── handler/
│   │   │   └── group_handler.go      # WS 群管理事件处理
│   │   ├── model/
│   │   │   ├── group.go              # im_groups 模型
│   │   │   └── join_request.go       # im_group_join_requests 模型
│   │   ├── router.go
│   │   └── provider.go
│   ├── file/                         # [新增] 文件上传模块
│   │   ├── controller/
│   │   │   └── file_controller.go    # POST /api/v1/upload
│   │   ├── service/
│   │   │   └── file_service.go       # MinIO 上传/删除封装
│   │   ├── router.go
│   │   └── provider.go
│   ├── im/                           # [扩展]
│   │   ├── dao/
│   │   │   └── read_dao.go           # [新增] im_message_reads 操作
│   │   ├── model/
│   │   │   └── message_read.go       # [新增] im_message_reads 模型
│   │   └── service/
│   │       └── im_service.go         # [扩展] 群消息/已读回执/管理员撤回
│   └── constants/
│       └── group.go                  # [新增] 群聊相关常量
├── pkg/
│   └── storage/
│       └── minio.go                  # [新增] MinIO 客户端初始化

frontend/src/
├── pages/
│   ├── group/                        # [新增] 群聊页面（独立体系）
│   │   ├── index.vue                 # 群聊会话列表
│   │   ├── conversation.vue          # 群聊对话页
│   │   ├── create.vue                # 创建群聊
│   │   ├── settings.vue              # 群聊设置
│   │   ├── members.vue               # 群成员列表
│   │   ├── invite.vue                # 邀请入群
│   │   ├── join-requests.vue         # 入群申请审批
│   │   └── search.vue                # 搜索公开群
│   └── chat/
│       └── read-detail.vue           # [新增] 已读回执详情页
├── store/
│   └── group.js                      # [新增] 群聊 Pinia Store
├── api/
│   ├── group.js                      # [新增] 群聊 API
│   └── file.js                       # [新增] 文件上传 API

admin/src/
├── views/
│   └── group/                        # [新增] 管理端群聊管理
│       ├── list.vue                  # 群列表
│       └── detail.vue                # 群详情（含成员管理）
├── api/
│   └── group.js                      # [新增] 管理端群聊 API
```

### 3.3 跨模块接口注入

延续 Phase 2a/2b 的接口注入模式，通过 Wire Bind 在 `app/provider/wire.go` 中统一绑定。

| 接口 | 定义模块 | 实现方 | 用途 |
|------|----------|--------|------|
| GroupMemberChecker | im/service | group/dao.GroupDAO | 检查用户是否为群成员 |
| GroupInfoGetter | im/service | group/dao.GroupDAO | 获取群信息（名称/头像/成员数） |
| GroupRoleChecker | im/service | group/dao.GroupDAO | 检查用户群角色（管理员撤回权限） |
| 已有接口 | - | - | FriendChecker / UserInfoGetter / FriendIDsGetter 等继续复用 |

---

## 四、数据库设计

### 4.1 新增表

#### im_groups（群聊信息表）

```sql
CREATE TABLE im_groups (
    id              BIGSERIAL PRIMARY KEY,
    conversation_id BIGINT NOT NULL UNIQUE REFERENCES im_conversations(id),
    name            VARCHAR(100) NOT NULL DEFAULT '',
    avatar          VARCHAR(500) DEFAULT '',
    owner_id        BIGINT NOT NULL,
    notice          TEXT DEFAULT '',
    max_members     INT NOT NULL DEFAULT 200,
    is_searchable   BOOLEAN NOT NULL DEFAULT TRUE,
    is_all_muted    BOOLEAN NOT NULL DEFAULT FALSE,
    status          SMALLINT NOT NULL DEFAULT 1,
    created_at      TIMESTAMP(0) NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP(0) NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  im_groups                IS '群聊信息表';
COMMENT ON COLUMN im_groups.id             IS '群唯一标识';
COMMENT ON COLUMN im_groups.conversation_id IS '关联 im_conversations.id';
COMMENT ON COLUMN im_groups.name           IS '群名称';
COMMENT ON COLUMN im_groups.avatar         IS '群头像 URL（MinIO）';
COMMENT ON COLUMN im_groups.owner_id       IS '群主用户 ID';
COMMENT ON COLUMN im_groups.notice         IS '群公告内容';
COMMENT ON COLUMN im_groups.max_members    IS '最大成员数，默认 200';
COMMENT ON COLUMN im_groups.is_searchable  IS '是否可被搜索发现';
COMMENT ON COLUMN im_groups.is_all_muted   IS '是否全体禁言';
COMMENT ON COLUMN im_groups.status         IS '群状态：1=正常，2=已解散';

CREATE INDEX idx_im_groups_owner ON im_groups(owner_id);
CREATE INDEX idx_im_groups_name ON im_groups USING gin(to_tsvector('simple', name));
-- 注意：实际搜索使用 ILIKE 模糊匹配（to_tsvector 不支持混合中英文词汇的分词）
```

#### im_group_join_requests（入群申请表）

```sql
CREATE TABLE im_group_join_requests (
    id          BIGSERIAL PRIMARY KEY,
    group_id    BIGINT NOT NULL REFERENCES im_groups(id),
    user_id     BIGINT NOT NULL,
    message     TEXT DEFAULT '',
    reviewer_id BIGINT DEFAULT NULL,
    status      SMALLINT NOT NULL DEFAULT 0,
    created_at  TIMESTAMP(0) NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP(0) NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  im_group_join_requests              IS '入群申请表';
COMMENT ON COLUMN im_group_join_requests.id            IS '申请唯一标识';
COMMENT ON COLUMN im_group_join_requests.group_id      IS '目标群 ID';
COMMENT ON COLUMN im_group_join_requests.user_id       IS '申请人用户 ID';
COMMENT ON COLUMN im_group_join_requests.message       IS '申请附言';
COMMENT ON COLUMN im_group_join_requests.reviewer_id   IS '审批人用户 ID';
COMMENT ON COLUMN im_group_join_requests.status        IS '状态：0=待审批，1=通过，2=拒绝';

CREATE INDEX idx_group_join_req_group ON im_group_join_requests(group_id, status);
CREATE INDEX idx_group_join_req_user ON im_group_join_requests(user_id);
```

#### im_message_reads（消息已读记录表）

```sql
CREATE TABLE im_message_reads (
    message_id BIGINT NOT NULL,
    user_id    BIGINT NOT NULL,
    read_at    TIMESTAMP(0) NOT NULL DEFAULT NOW(),
    PRIMARY KEY (message_id, user_id)
);

COMMENT ON TABLE  im_message_reads            IS '群聊消息已读记录表（消息级别）';
COMMENT ON COLUMN im_message_reads.message_id IS '消息 ID';
COMMENT ON COLUMN im_message_reads.user_id    IS '已读用户 ID';
COMMENT ON COLUMN im_message_reads.read_at    IS '已读时间';

CREATE INDEX idx_msg_reads_user ON im_message_reads(user_id, read_at);
```

### 4.2 扩展现有表

#### im_conversation_members — 新增 5 个字段

```sql
ALTER TABLE im_conversation_members
    ADD COLUMN role              SMALLINT NOT NULL DEFAULT 0,
    ADD COLUMN nickname          VARCHAR(50) DEFAULT '',
    ADD COLUMN is_muted          BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN is_do_not_disturb BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN joined_at         TIMESTAMP(0) DEFAULT NULL;

COMMENT ON COLUMN im_conversation_members.role              IS '成员角色：0=普通成员，1=管理员，2=群主';
COMMENT ON COLUMN im_conversation_members.nickname          IS '群内昵称（仅群聊有效）';
COMMENT ON COLUMN im_conversation_members.is_muted          IS '是否被禁言';
COMMENT ON COLUMN im_conversation_members.is_do_not_disturb IS '是否消息免打扰（微信模式：仍计未读但灰色显示）';
COMMENT ON COLUMN im_conversation_members.joined_at         IS '加入群聊时间';
```

#### im_messages — 新增 1 个字段

```sql
ALTER TABLE im_messages
    ADD COLUMN at_user_ids BIGINT[] DEFAULT NULL;

COMMENT ON COLUMN im_messages.at_user_ids IS '@提醒用户 ID 列表，NULL=无@，包含 0 表示 @所有人';
```

### 4.3 数据库表关系图

```
im_conversations (type=2 群聊)
    │ 1:1
    ├── im_groups (群信息：名称/头像/公告/群主)
    │       │ 1:N
    │       └── im_group_join_requests (入群申请)
    │ 1:N
    ├── im_conversation_members (成员：+role/nickname/is_muted/is_do_not_disturb)
    │ 1:N
    └── im_messages (消息：+at_user_ids)
            │ 1:N
            └── im_message_reads (群聊消息已读记录)
```

---

## 五、API 设计

### 5.1 群聊 REST API（16 个）

| # | 方法 | 路径 | 描述 | 权限 |
|---|------|------|------|------|
| 1 | POST | /api/v1/groups | 创建群聊 | 登录用户 |
| 2 | GET | /api/v1/groups/:id | 群详情 | 群成员 |
| 3 | PUT | /api/v1/groups/:id | 更新群信息 | 群主/管理员 |
| 4 | DELETE | /api/v1/groups/:id | 解散群聊 | 群主 |
| 5 | GET | /api/v1/groups/:id/members | 成员列表 | 群成员 |
| 6 | POST | /api/v1/groups/:id/members | 邀请/拉人入群 | 群主/管理员 |
| 7 | DELETE | /api/v1/groups/:id/members/:uid | 踢人 | 群主/管理员 |
| 8 | PUT | /api/v1/groups/:id/members/:uid/role | 设置/取消管理员 | 群主 |
| 9 | PUT | /api/v1/groups/:id/members/:uid/mute | 禁言/解除禁言 | 群主/管理员 |
| 10 | POST | /api/v1/groups/:id/leave | 退出群聊 | 群成员 |
| 11 | PUT | /api/v1/groups/:id/transfer | 转让群主 | 群主 |
| 12 | PUT | /api/v1/groups/:id/members/me/nickname | 修改群昵称 | 群成员 |
| 13 | POST | /api/v1/groups/:id/join-requests | 申请入群 | 登录用户 |
| 14 | GET | /api/v1/groups/:id/join-requests | 入群申请列表 | 群主/管理员 |
| 15 | PUT | /api/v1/groups/:id/join-requests/:rid | 审批入群申请 | 群主/管理员 |
| 16 | GET | /api/v1/groups/search | 搜索公开群 | 登录用户 |

### 5.2 已读回执 REST API（2 个）

| # | 方法 | 路径 | 描述 |
|---|------|------|------|
| 1 | GET | /api/v1/im/messages/:id/reads | 消息已读详情（谁读了，分页） |
| 2 | GET | /api/v1/im/messages/:id/read-count | 消息已读/未读计数 |

### 5.3 文件上传 REST API（1 个）

| # | 方法 | 路径 | 描述 |
|---|------|------|------|
| 1 | POST | /api/v1/upload | 通用文件上传（返回 MinIO URL） |

### 5.4 管理端 REST API（4 个）

| # | 方法 | 路径 | 描述 |
|---|------|------|------|
| 1 | GET | /api/v1/admin/groups | 群列表（分页 + 筛选） |
| 2 | GET | /api/v1/admin/groups/:id | 群详情（含成员列表） |
| 3 | DELETE | /api/v1/admin/groups/:id | 管理员解散群 |
| 4 | DELETE | /api/v1/admin/groups/:id/members/:uid | 管理员移除成员 |

### 5.5 WebSocket 事件

#### 消息事件（复用 im.message.*，扩展群聊支持）

| 事件 | 方向 | 变更说明 |
|------|------|----------|
| im.message.send | C→S | data 新增 at_user_ids 字段 |
| im.message.send.ack | S→C | 无变化 |
| im.message.new | S→C | data 新增 conv_type、at_user_ids 字段 |
| im.message.recall | C→S | 群聊时管理员可撤回他人消息（无时限） |
| im.message.recalled | S→C | 群聊时区分 "管理员 XX 撤回了 YY 的消息" |

#### 已读回执事件（新增）

| 事件 | 方向 | 说明 |
|------|------|------|
| im.message.read | C→S | 上报已读：单聊走 last_read_msg_id，群聊写 im_message_reads |
| im.message.read.ack | S→C | 单聊：实时推送已读状态给消息发送者 |
| im.message.read.count | S→C | 群聊：推送已读计数变化 {msg_id, read_count} |

#### 群管理事件（新增 group.* 系列）

| 事件 | 方向 | 说明 |
|------|------|------|
| group.member.join | S→C | 成员加入通知（含系统消息 "XX 加入了群聊"） |
| group.member.leave | S→C | 成员退出通知（含系统消息 "XX 退出了群聊"） |
| group.member.kicked | S→C | 成员被踢通知（含系统消息 "XX 被移出了群聊"） |
| group.info.update | S→C | 群信息变更通知（名称/头像） |
| group.notice.update | S→C | 群公告变更通知（含系统消息） |
| group.dissolved | S→C | 群解散通知 |
| group.mute.update | S→C | 禁言状态变更通知（个人/全体） |
| group.join.request | S→C | 新入群申请（推送给群主/管理员） |
| group.join.approved | S→C | 入群申请通过（推送给申请人） |

---

## 六、核心业务流程

### 6.1 创建群聊

```
1. 用户选择好友 + 搜索用户 ID → 确定初始成员列表
2. POST /api/v1/groups {name, member_ids[]}
3. GroupService:
   a. 创建 im_conversations (type=2)
   b. 创建 im_groups (关联 conversation_id)
   c. 批量创建 im_conversation_members (创建者 role=2 群主, 其余 role=0)
   d. 插入系统消息 "XX 创建了群聊，邀请了 A、B、C 加入"
   e. PubSub 推送 group.member.join 给所有初始成员
4. 返回群信息 + 会话 ID
```

### 6.2 群消息发送

```
1. 发送者 WS: im.message.send {conversation_id, content, at_user_ids}
2. IMService.SendMessage:
   a. 查询 im_conversations.type → 群聊分支
   b. 校验群成员身份 + 禁言检查（is_muted / is_all_muted，管理员豁免全体禁言）
   c. 写入 im_messages (含 at_user_ids)
   d. 更新 im_conversations.last_msg_*
   e. 遍历群成员：
      - 排除发送者
      - 非免打扰成员：递增 unread_count + Redis 全局未读
      - 免打扰成员：递增 unread_count，不递增 Redis 全局未读
      - 被 @的成员：记录 @计数（可存 im_conversation_members 或 Redis）
   f. PubSub.PublishToUsers → 推送 im.message.new 给所有在线成员
3. 发送者收到 im.message.send.ack
```

### 6.3 已读回执（单聊）

```
1. 用户打开单聊会话
2. 前端 WS: im.message.read {conversation_id}
3. IMService:
   a. 获取会话最新消息 ID → 更新 last_read_msg_id
   b. 清零 unread_count + Redis 全局未读
   c. PubSub 推送 im.message.read.ack 给对方
      {conversation_id, reader_id, last_read_msg_id}
4. 对方前端收到后，对 ID ≤ last_read_msg_id 的自己发送的消息标记为 "已读"
```

### 6.4 已读回执（群聊）

```
1. 用户打开群聊会话，前端获取可视区域消息 ID 列表
2. 前端 WS: im.message.read {conversation_id, message_ids[]}
3. IMService:
   a. 批量写入 im_message_reads (ON CONFLICT DO NOTHING)
   b. 更新 last_read_msg_id + 清零 unread_count
   c. 查询受影响消息的 sender_id（去重）
   d. 对每个 sender_id 推送 im.message.read.count
      {msg_id, read_count, total_members}
4. 发送者前端更新 "X人已读" 显示
5. 用户点击 "X人已读" → GET /api/v1/im/messages/:id/reads → 显示详情列表
```

### 6.5 @提醒流程

```
1. 发送者在输入框键入 @ → 弹出成员选择列表
2. 选择成员（或 @所有人）→ 消息中插入 @昵称 标记
3. 发送消息时 at_user_ids 包含被 @用户的 ID（0 表示 @所有人）
4. 接收方前端：
   a. 收到 im.message.new 检查 at_user_ids
   b. 如果包含自己的 ID 或 0 → 标记该会话 "被@"
   c. 会话列表显示 "[N条] @了我"
```

---

## 七、前端页面规划

### 7.1 新增页面（9 个）

| 页面 | 路径 | 功能描述 |
|------|------|----------|
| 群聊会话列表 | pages/group/index.vue | 群聊 Tab 页，显示群会话列表 |
| 群聊对话页 | pages/group/conversation.vue | 群消息展示 + 发送 + @选择器 |
| 创建群聊 | pages/group/create.vue | 选好友 + 全站搜索非好友用户 + 填群名（最少 1 人即可创建） |
| 群设置 | pages/group/settings.vue | 群信息/公告/成员概览/免打扰/退群/解散 |
| 群成员列表 | pages/group/members.vue | 完整成员列表 + 全角色标识（群主/管理员/成员）+ 自定义操作弹窗（含头像角色信息）+ 管理操作 |
| 邀请入群 | pages/group/invite.vue | 好友多选 + 全站搜索非好友用户 + 排除已在群内成员 |
| 入群审批 | pages/group/join-requests.vue | 群主/管理员审批入群申请 |
| 搜索群聊 | pages/group/search.vue | 搜索公开群 + 申请加入 + 已加入状态智能显示 |
| 已读详情 | pages/chat/read-detail.vue | 已读/未读人员列表（单聊+群聊共用） |

### 7.2 修改现有页面

| 页面 | 修改内容 |
|------|----------|
| pages/chat/index.vue | 改为 Tab 切换：单聊 Tab + 群聊 Tab |
| pages/chat/conversation.vue | 添加单聊已读状态展示（"已读"/"未读"） |
| pages/chat/search.vue | 全局搜索扩展支持群聊消息结果 |
| components/CustomTabBar.vue | TabBar 适配群聊入口 |

### 7.3 管理端新增页面（2 个）

| 页面 | 路径 | 功能描述 |
|------|------|----------|
| 群列表 | admin/src/views/group/list.vue | 群列表（搜索/分页/状态筛选） |
| 群详情 | admin/src/views/group/detail.vue | 群信息 + 成员列表 + 解散/移除操作 |

---

## 八、Docker Compose 变更

在 `deploy/docker/docker-compose.dev.yml` 中新增 MinIO 服务：

```yaml
minio:
  image: minio/minio:latest
  command: server /data --console-address ":9001"
  ports:
    - "9000:9000"
    - "9001:9001"
  environment:
    MINIO_ROOT_USER: echochat
    MINIO_ROOT_PASSWORD: echochat123456
  volumes:
    - minio_data:/data
  restart: unless-stopped
```

Go 后端 `config.yaml` 新增：

```yaml
minio:
  endpoint: "localhost:9000"
  access_key: "echochat"
  secret_key: "echochat123456"
  bucket: "echochat"
  use_ssl: false
```

---

## 九、常量定义规划

```go
// app/constants/group.go

// 群成员角色
const (
    GroupRoleMember = 0  // 普通成员
    GroupRoleAdmin  = 1  // 管理员
    GroupRoleOwner  = 2  // 群主
)

// 群状态
const (
    GroupStatusActive    = 1  // 正常
    GroupStatusDissolved = 2  // 已解散
)

// 入群申请状态
const (
    JoinRequestPending  = 0  // 待审批
    JoinRequestApproved = 1  // 已通过
    JoinRequestRejected = 2  // 已拒绝
)

// 系统消息类型（im_messages.type 扩展）
const (
    MessageTypeSystem = 10  // 系统通知消息
)
```

---

## 十、DTO 扩展规划

### 10.1 群聊 DTO

```go
// CreateGroupRequest 创建群聊请求
type CreateGroupRequest struct {
    Name      string  `json:"name" binding:"required"`
    MemberIDs []int64 `json:"member_ids" binding:"required,min=1"`
}

// GroupDTO 群详情
type GroupDTO struct {
    ID             int64  `json:"id"`
    ConversationID int64  `json:"conversation_id"`
    Name           string `json:"name"`
    Avatar         string `json:"avatar"`
    OwnerID        int64  `json:"owner_id"`
    OwnerNickname  string `json:"owner_nickname"`
    Notice         string `json:"notice"`
    MemberCount    int    `json:"member_count"`
    MaxMembers     int    `json:"max_members"`
    IsSearchable   bool   `json:"is_searchable"`
    IsAllMuted     bool   `json:"is_all_muted"`
    Status         int    `json:"status"`
    CreatedAt      string `json:"created_at"`
}

// GroupMemberDTO 群成员信息
type GroupMemberDTO struct {
    UserID       int64  `json:"user_id"`
    Username     string `json:"username"`
    Nickname     string `json:"nickname"`      // 群昵称（优先）或全局昵称
    Avatar       string `json:"avatar"`
    Role         int    `json:"role"`
    IsMuted      bool   `json:"is_muted"`
    IsOnline     bool   `json:"is_online"`
    JoinedAt     string `json:"joined_at"`
}
```

### 10.2 已读回执 DTO

```go
// ReadDetailDTO 消息已读详情
type ReadDetailDTO struct {
    UserID   int64  `json:"user_id"`
    Nickname string `json:"nickname"`
    Avatar   string `json:"avatar"`
    ReadAt   string `json:"read_at"`
}

// ReadCountDTO 消息已读计数
type ReadCountDTO struct {
    MessageID    int64 `json:"message_id"`
    ReadCount    int   `json:"read_count"`
    UnreadCount  int   `json:"unread_count"`
    TotalMembers int   `json:"total_members"`
}
```

### 10.3 ConversationDTO 扩展

```go
// ConversationDTO 扩展：增加群聊字段
type ConversationDTO struct {
    // 原有字段...
    GroupID        *int64  `json:"group_id,omitempty"`        // 群聊时的群 ID
    GroupName      string  `json:"group_name,omitempty"`      // 群名称
    GroupAvatar    string  `json:"group_avatar,omitempty"`    // 群头像
    MemberCount    int     `json:"member_count,omitempty"`    // 群成员数
    IsDoNotDisturb bool    `json:"is_do_not_disturb"`         // 是否免打扰
    AtMeCount      int     `json:"at_me_count,omitempty"`     // 被@计数
}
```

---

## 十一、补充设计细节

### 11.1 @提醒计数存储

会话列表需要显示 "[N条] @了我"，需要在 `im_conversation_members` 表新增字段存储：

```sql
ALTER TABLE im_conversation_members
    ADD COLUMN at_me_count INT DEFAULT 0;

COMMENT ON COLUMN im_conversation_members.at_me_count IS '被@提醒未读计数，打开会话后清零';
```

**操作流程：**
- 收到含 at_user_ids 的消息时：若当前用户在列表中 → `at_me_count += 1`
- @所有人（at_user_ids 含 0）：群内所有成员（排除发送者）`at_me_count += 1`
- 用户打开会话：`at_me_count = 0`（同清零 unread_count 时一并清零）

### 11.2 系统消息内容格式

系统消息（type=10）的 `content` 字段使用**纯文本格式**，不采用结构化 JSON。理由：
- 简单直观，前端直接渲染无需解析
- 系统消息种类有限且模板固定
- 不需要国际化（当前仅中文）

**系统消息模板：**

| 场景 | content 示例 |
|------|-------------|
| 创建群 | "XX 创建了群聊，邀请了 A、B、C 加入" |
| 邀请入群 | "XX 邀请了 A、B 加入群聊" |
| 退出群 | "XX 退出了群聊" |
| 被踢出 | "XX 被移出了群聊" |
| 转让群主 | "XX 将群主转让给了 YY" |
| 设管理员 | "XX 被设为管理员" |
| 取消管理员 | "XX 被取消了管理员" |
| 群公告 | "XX 修改了群公告" |
| 管理员撤回 | "管理员 XX 撤回了 YY 的一条消息" |
| 解散群 | "群主 XX 解散了群聊" |
| 禁言 | "XX 被禁言" |
| 全体禁言 | "管理员 XX 开启了全体禁言" |

### 11.3 群聊离线消息推送

沿用 Phase 2b 的离线推送机制（`OfflineMessagePusher`）：

1. 用户 WebSocket 连接后，服务端检查该用户的所有未读会话（含群聊）
2. 推送 `im.offline.sync` 事件，包含未读会话摘要列表
3. 群聊会话的摘要包含：conversation_id、type=2、group_name、group_avatar、unread_count、last_msg_content、at_me_count
4. 前端收到后更新群聊会话列表和 TabBar badge

### 11.4 文件上传策略

群头像上传不需要 `FileUploader` 接口注入。流程：

```
1. 前端调用 POST /api/v1/upload 上传图片 → 获取 MinIO URL
2. 前端调用 PUT /api/v1/groups/:id {avatar: "minio-url"} 更新群头像
```

两步操作，`file` 模块和 `group` 模块完全解耦，无需接口注入。

### 11.5 群聊错误常量规划

```go
// app/constants/group.go 错误定义（在 group/service 中使用）

var (
    ErrGroupNotFound        = errors.New("群聊不存在")
    ErrGroupDissolved       = errors.New("群聊已解散")
    ErrNotGroupMember       = errors.New("你不是该群成员")
    ErrNotGroupOwner        = errors.New("仅群主可执行此操作")
    ErrNotGroupAdmin        = errors.New("仅群主或管理员可执行此操作")
    ErrGroupFull            = errors.New("群成员已满")
    ErrAlreadyMember        = errors.New("该用户已是群成员")
    ErrCannotKickHigherRole = errors.New("不能操作同级或更高权限的成员")
    ErrOwnerCannotLeave     = errors.New("群主不能退出群聊，请先转让群主")
    ErrCannotMuteSelf       = errors.New("不能禁言自己")
    ErrAlreadyMuted         = errors.New("该成员已被禁言")
    ErrUserMuted            = errors.New("你已被禁言，无法发送消息")
    ErrGroupAllMuted        = errors.New("当前群已开启全体禁言")
    ErrPendingRequestExists = errors.New("已有待处理的入群申请")
    ErrJoinRequestNotFound  = errors.New("入群申请不存在")
)
```
