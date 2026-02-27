# EchoChat 音视频会议直播系统 - 整体设计方案

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**目标：** 构建一套跨端可用、可扩展、可演进的实时音视频会议直播系统，第一期（MVP）实现用户体系、即时聊天、好友管理、多人音视频会议、消息通知五大核心功能。

**架构：** 精简单体 + 媒体微服务。Go 单体模块化服务处理所有业务逻辑，mediasoup Node 服务独立管理媒体资源。模块间按微服务边界组织代码，预留后期拆分能力。

**技术栈：** Go (Gin + GORM + Wire) / Node.js + mediasoup / uniapp (Vue 3) / Vue 3 + Element Plus / PostgreSQL / Redis / Docker Compose

---

## 一、需求确认汇总

| 维度 | 决策 |
|------|------|
| 后端 | 纯 Go 语言（移除 PHP） |
| 前台前端 | uniapp (Vue 3) + mediasoup-client |
| 后台前端 | 独立 Vue 3 + Vite + Element Plus（PC 端专注） |
| 媒体层 | Node.js + mediasoup（保留官方 Node 中间层） |
| 存储 | PostgreSQL 16 + Redis 7 |
| 认证 | 邮箱+密码、用户名+密码（微信授权后期迭代） |
| MVP 功能 | 用户体系、即时聊天、好友管理、音视频会议、消息通知 |
| 多端优先级 | H5 > 桌面端 > 微信小程序 > Android |
| 并发规模 | 中型（单场50人，总用户千级） |
| 部署 | Docker Compose（初期），预留 K8s 编排 |
| 团队 | 个人 + AI 辅助开发 |

---

## 二、系统架构总览

```
                    ┌─────────────────────────────────────┐
                    │           负载均衡 (Nginx)            │
                    └──────┬──────────────┬───────────────┘
                           │              │
              ┌────────────┴───┐   ┌──────┴────────────┐
              │  前台用户端     │   │  后台管理端         │
              │  uniapp        │   │  Vue3+ElementPlus  │
              │  (H5/App/小程序)│   │  (PC Web)          │
              └────────┬───────┘   └──────┬─────────────┘
                       │                  │
          WebSocket + HTTPS          HTTPS (RESTful)
                       │                  │
              ┌────────┴──────────────────┴─────────────┐
              │         Go 单体服务（模块化）              │
              │                                          │
              │  ┌────────┐ ┌──────┐ ┌────────────────┐  │
              │  │ auth   │ │  im  │ │  meeting       │  │
              │  │ 用户鉴权│ │ 聊天 │ │  会议/信令     │  │
              │  ├────────┤ ├──────┤ ├────────────────┤  │
              │  │contact │ │notify│ │  admin         │  │
              │  │ 联系人  │ │ 通知 │ │  后台管理      │  │
              │  └────────┘ └──────┘ └────────────────┘  │
              │          │              │                  │
              │     PostgreSQL       Redis                │
              └──────────┼──────────────┼─────────────────┘
                         │         HTTP/gRPC
              ┌──────────┴──────────────┴─────────────────┐
              │         mediasoup Node 服务                 │
              │  ┌──────────┐  ┌──────────┐               │
              │  │ Worker 1 │  │ Worker N │  (C++ SFU)    │
              │  └──────────┘  └──────────┘               │
              └────────────────────────────────────────────┘
```

### 架构选型：精简单体 + 媒体微服务

- **方案一（已采用）：** Go 单体模块化服务 + mediasoup Node 独立服务
- **演进路径：** 单体 → 分组服务（实时/业务分离） → 完全微服务
- **微服务预留规则：**
  - 模块间零直接引用，通过 interface 通信
  - 每个模块有独立的路由注册
  - 数据库表按模块前缀命名
  - Redis key 按模块命名空间

### 用户体系设计

采用统一用户表 + RBAC 权限体系，认证入口分离：

- 前台用户和后台管理员共用 `auth_users` 表
- 通过 `auth_roles` 和 `auth_user_roles` 区分角色（user / admin / super_admin）
- 前台 API 路由：`/api/v1/auth/*` — 仅需 JWT 验证
- 后台 API 路由：`/api/v1/admin/*` — JWT 验证 + 角色检查双重中间件

---

## 三、项目目录结构

```
EchoChat/
├── frontend/                          # 前台用户端 (uniapp)
│   ├── src/
│   │   ├── api/                       # API 请求封装
│   │   ├── components/                # 公共组件
│   │   ├── composables/               # 组合式函数
│   │   ├── pages/                     # 页面
│   │   │   ├── auth/                  # 登录/注册
│   │   │   ├── chat/                  # 聊天相关
│   │   │   ├── contact/               # 联系人
│   │   │   ├── meeting/               # 会议相关
│   │   │   ├── notification/          # 通知中心
│   │   │   └── profile/               # 个人中心
│   │   ├── store/                     # Pinia 状态管理
│   │   ├── utils/                     # 工具函数
│   │   ├── services/                  # 业务服务（WebSocket、mediasoup-client）
│   │   └── static/                    # 静态资源
│   └── package.json
│
├── admin/                             # 后台管理端 (Vue 3 + Element Plus)
│   ├── src/
│   │   ├── api/
│   │   ├── components/
│   │   ├── views/
│   │   │   ├── dashboard/             # 数据看板
│   │   │   ├── user/                  # 用户管理
│   │   │   ├── meeting/               # 会议监控
│   │   │   ├── permission/            # 权限管理
│   │   │   ├── system/                # 系统配置
│   │   │   └── layout/                # 布局框架
│   │   ├── router/
│   │   └── store/
│   └── package.json
│
├── backend/
│   └── go-service/                    # Go 单体服务
│       ├── cmd/
│       │   └── server/main.go         # 入口
│       ├── config/                    # 配置
│       ├── app/
│       │   ├── auth/                  # 用户鉴权模块
│       │   │   ├── controller/
│       │   │   │   ├── auth_controller.go
│       │   │   │   └── admin_auth_controller.go
│       │   │   ├── service/
│       │   │   ├── dao/
│       │   │   └── model/
│       │   ├── im/                    # 即时通讯模块
│       │   ├── contact/               # 联系人模块
│       │   ├── meeting/               # 会议模块
│       │   ├── notify/                # 通知模块
│       │   ├── admin/                 # 后台管理模块
│       │   ├── constants/             # 常量定义
│       │   └── dto/                   # 数据传输对象
│       ├── pkg/                       # 公共工具包
│       │   ├── db/                    # 数据库连接
│       │   ├── redis/                 # Redis 客户端
│       │   ├── ws/                    # WebSocket 管理
│       │   ├── middleware/            # 中间件
│       │   ├── utils/                 # 工具函数
│       │   └── logs/                  # 日志
│       ├── go.mod
│       └── go.sum
│
├── media-server/                      # mediasoup Node 服务
│   ├── src/
│   │   ├── worker-manager.js          # Worker 管理
│   │   ├── room.js                    # 房间媒体资源管理
│   │   ├── peer.js                    # 参与者媒体管理
│   │   └── api.js                     # 对外 HTTP API
│   ├── config/
│   └── package.json
│
├── deploy/                            # 部署配置
│   ├── docker/                        # Dockerfile 集合
│   ├── docker-compose.yml
│   ├── docker-compose.dev.yml
│   ├── nginx/
│   └── k8s/                           # 预留 K8s 配置
│
├── docs/                              # 文档
│   ├── plans/                         # 设计方案与实施计划
│   ├── api/                           # API 文档
│   └── architecture/                  # 架构文档
│
├── scripts/                           # 脚本
│   ├── init-db.sql
│   └── dev-setup.sh
│
└── README.md
```

---

## 四、数据库设计

### 4.1 PostgreSQL 核心表

#### auth 模块 — 用户与权限

```sql
CREATE TABLE auth_users (
    id              BIGSERIAL PRIMARY KEY,
    username        VARCHAR(50)  UNIQUE NOT NULL,
    email           VARCHAR(100) UNIQUE NOT NULL,
    password_hash   VARCHAR(255) NOT NULL,
    nickname        VARCHAR(50)  NOT NULL DEFAULT '',
    avatar          VARCHAR(500) NOT NULL DEFAULT '',
    gender          SMALLINT     NOT NULL DEFAULT 0,       -- 0:未知 1:男 2:女
    phone           VARCHAR(20)  DEFAULT NULL,
    status          SMALLINT     NOT NULL DEFAULT 1,       -- 1:正常 2:禁用 3:注销
    last_login_at   TIMESTAMPTZ  DEFAULT NULL,
    last_login_ip   VARCHAR(50)  DEFAULT NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE auth_roles (
    id          SERIAL PRIMARY KEY,
    code        VARCHAR(50) UNIQUE NOT NULL,               -- user, admin, super_admin
    name        VARCHAR(50) NOT NULL,
    description VARCHAR(200) DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE auth_user_roles (
    user_id    BIGINT NOT NULL REFERENCES auth_users(id),
    role_id    INT    NOT NULL REFERENCES auth_roles(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, role_id)
);
```

#### contact 模块 — 联系人与好友

```sql
CREATE TABLE contact_friendships (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT   NOT NULL REFERENCES auth_users(id),
    friend_id   BIGINT   NOT NULL REFERENCES auth_users(id),
    remark      VARCHAR(50) DEFAULT '',
    group_id    BIGINT   DEFAULT NULL,
    status      SMALLINT NOT NULL DEFAULT 0,               -- 0:待确认 1:已接受 2:已拒绝 3:已拉黑
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, friend_id)
);

CREATE TABLE contact_groups (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT      NOT NULL REFERENCES auth_users(id),
    name       VARCHAR(50) NOT NULL,
    sort_order INT         NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

#### im 模块 — 即时通讯（统一会话模型）

单聊和群聊统一抽象为"会话"，本质都是"一组人在一个空间里收发消息"。这是微信、钉钉、Slack 等主流 IM 的标准模型。

```sql
CREATE TABLE im_conversations (
    id          BIGSERIAL PRIMARY KEY,
    type        SMALLINT    NOT NULL,                      -- 1:单聊 2:群聊
    name        VARCHAR(100) DEFAULT '',
    avatar      VARCHAR(500) DEFAULT '',
    owner_id    BIGINT      DEFAULT NULL,
    max_members INT         NOT NULL DEFAULT 200,
    status      SMALLINT    NOT NULL DEFAULT 1,            -- 1:正常 2:已解散
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE im_conversation_members (
    id              BIGSERIAL PRIMARY KEY,
    conversation_id BIGINT   NOT NULL REFERENCES im_conversations(id),
    user_id         BIGINT   NOT NULL REFERENCES auth_users(id),
    role            SMALLINT NOT NULL DEFAULT 0,           -- 0:普通成员 1:管理员 2:群主
    nickname        VARCHAR(50) DEFAULT '',
    is_muted        BOOLEAN  NOT NULL DEFAULT FALSE,
    is_pinned       BOOLEAN  NOT NULL DEFAULT FALSE,
    last_read_msg_id BIGINT  DEFAULT 0,
    joined_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (conversation_id, user_id)
);

CREATE TABLE im_messages (
    id              BIGSERIAL PRIMARY KEY,
    conversation_id BIGINT   NOT NULL,
    sender_id       BIGINT   NOT NULL,
    type            SMALLINT NOT NULL DEFAULT 1,           -- 1:文本 2:图片 3:文件 4:语音 5:系统消息
    content         TEXT     NOT NULL DEFAULT '',
    extra           JSONB    DEFAULT '{}',
    status          SMALLINT NOT NULL DEFAULT 1,           -- 1:正常 2:已撤回 3:已删除
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_im_messages_conv_time ON im_messages(conversation_id, created_at DESC);
```

#### meeting 模块 — 音视频会议

```sql
CREATE TABLE meeting_rooms (
    id            BIGSERIAL PRIMARY KEY,
    room_code     VARCHAR(20) UNIQUE NOT NULL,             -- 会议号
    title         VARCHAR(200) NOT NULL,
    host_id       BIGINT      NOT NULL REFERENCES auth_users(id),
    type          SMALLINT    NOT NULL DEFAULT 1,          -- 1:即时会议 2:预约会议
    password      VARCHAR(50) DEFAULT NULL,
    max_members   INT         NOT NULL DEFAULT 50,
    status        SMALLINT    NOT NULL DEFAULT 0,          -- 0:未开始 1:进行中 2:已结束
    scheduled_at  TIMESTAMPTZ DEFAULT NULL,                -- 预约时间（即时会议为NULL）
    started_at    TIMESTAMPTZ DEFAULT NULL,
    ended_at      TIMESTAMPTZ DEFAULT NULL,
    settings      JSONB       DEFAULT '{}',                -- 会议设置
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE meeting_participants (
    id           BIGSERIAL PRIMARY KEY,
    room_id      BIGINT   NOT NULL REFERENCES meeting_rooms(id),
    user_id      BIGINT   NOT NULL REFERENCES auth_users(id),
    role         SMALLINT NOT NULL DEFAULT 0,              -- 0:参与者 1:主持人 2:联合主持人
    joined_at    TIMESTAMPTZ DEFAULT NULL,
    left_at      TIMESTAMPTZ DEFAULT NULL,
    duration     INT      DEFAULT 0,
    UNIQUE (room_id, user_id)
);
```

会议类型状态流转：
- 即时会议：创建 → 进行中(1) → 已结束(2)
- 预约会议：创建 → 未开始(0) → 进行中(1) → 已结束(2)

#### notify 模块 — 消息通知

```sql
CREATE TABLE notify_notifications (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT      NOT NULL REFERENCES auth_users(id),
    type        VARCHAR(50) NOT NULL,                      -- meeting_invite, friend_request, system
    title       VARCHAR(200) NOT NULL,
    content     TEXT        DEFAULT '',
    extra       JSONB       DEFAULT '{}',
    is_read     BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_notify_user_read ON notify_notifications(user_id, is_read, created_at DESC);
```

#### admin 模块 — 管理操作日志

```sql
CREATE TABLE admin_operation_logs (
    id          BIGSERIAL PRIMARY KEY,
    admin_id    BIGINT      NOT NULL REFERENCES auth_users(id),
    module      VARCHAR(50) NOT NULL,
    action      VARCHAR(50) NOT NULL,
    target_type VARCHAR(50) DEFAULT '',
    target_id   BIGINT      DEFAULT NULL,
    detail      JSONB       DEFAULT '{}',
    ip          VARCHAR(50) DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### 4.2 Redis 数据结构

```
# 用户认证
echo:auth:token:{user_id}           → JWT Token (STRING, TTL 7天)
echo:auth:refresh:{user_id}         → Refresh Token (STRING, TTL 30天)

# 用户在线状态
echo:user:online                    → 在线用户集合 (SET)
echo:user:status:{user_id}          → 用户状态 JSON (STRING, TTL 自动过期)

# 即时通讯
echo:im:unread:{user_id}            → 各会话未读数 (HASH: conv_id → count)
echo:im:typing:{conversation_id}    → 正在输入的用户 (SET, TTL 5秒)

# 会议实时状态
echo:meeting:room:{room_code}       → 房间实时状态 JSON (STRING)
echo:meeting:members:{room_code}    → 房间在线成员 (HASH: user_id → 状态JSON)
echo:meeting:transport:{room_code}  → Transport/Producer 映射 (HASH)

# WebSocket 连接映射
echo:ws:user:{user_id}              → 连接所在的服务实例ID (STRING)
echo:ws:conn:{conn_id}              → 连接信息 JSON (STRING)
```

---

## 五、通信协议与 API 设计

### 5.1 通信方式总览

| 通信路径 | 协议 | 用途 |
|---------|------|------|
| 客户端 ↔ Go 服务 | WebSocket | 实时消息、信令、状态同步 |
| 客户端 ↔ Go 服务 | HTTPS RESTful | 用户操作、数据查询、管理功能 |
| 管理端 ↔ Go 服务 | HTTPS RESTful | 后台管理 CRUD |
| Go 服务 ↔ mediasoup Node | HTTP | 媒体资源创建/管理 |
| 客户端 ↔ mediasoup Worker | WebRTC (DTLS/RTP) | 音视频媒体流直连 |

### 5.2 WebSocket 消息协议

统一 JSON 格式：

```json
{
    "event": "im.message.send",
    "seq": 1001,
    "data": {},
    "timestamp": 1740700000
}
```

事件命名规范：`{模块}.{对象}.{动作}`

```
# 即时通讯
im.message.send / im.message.new / im.message.revoke / im.message.read
im.typing.start / im.typing.stop

# 会议信令
meeting.room.join / meeting.room.leave / meeting.room.info
meeting.member.join / meeting.member.leave / meeting.member.mute / meeting.member.video

# mediasoup 信令
meeting.transport.create / meeting.transport.connect
meeting.produce.start / meeting.produce.stop
meeting.consume.start / meeting.consume.resume

# 用户状态
user.status.online / user.status.offline

# 通知
notify.new / notify.meeting.invite / notify.friend.request
```

服务端响应格式：

```json
{
    "event": "im.message.send.ack",
    "seq": 1001,
    "code": 0,
    "message": "ok",
    "data": { "msg_id": 10086 }
}
```

### 5.3 前台用户 RESTful API

```
# 认证模块
POST   /api/v1/auth/register
POST   /api/v1/auth/login
POST   /api/v1/auth/logout
POST   /api/v1/auth/refresh-token
GET    /api/v1/auth/profile
PUT    /api/v1/auth/profile
PUT    /api/v1/auth/password

# 联系人模块
GET    /api/v1/contacts
POST   /api/v1/contacts/request
POST   /api/v1/contacts/accept
POST   /api/v1/contacts/reject
DELETE /api/v1/contacts/:id
PUT    /api/v1/contacts/:id/remark
GET    /api/v1/contacts/groups
POST   /api/v1/contacts/groups

# 即时通讯模块
GET    /api/v1/conversations
POST   /api/v1/conversations
GET    /api/v1/conversations/:id
GET    /api/v1/conversations/:id/messages
POST   /api/v1/conversations/:id/members
DELETE /api/v1/conversations/:id/members/:uid

# 会议模块
POST   /api/v1/meetings
POST   /api/v1/meetings/schedule
GET    /api/v1/meetings/:code
POST   /api/v1/meetings/:code/join
POST   /api/v1/meetings/:code/leave
GET    /api/v1/meetings/upcoming
GET    /api/v1/meetings/ongoing
GET    /api/v1/meetings/history

# 通知模块
GET    /api/v1/notifications
PUT    /api/v1/notifications/:id/read
PUT    /api/v1/notifications/read-all
```

### 5.4 后台管理 RESTful API

```
# 管理员认证
POST   /api/v1/admin/auth/login

# 用户管理
GET    /api/v1/admin/users
GET    /api/v1/admin/users/:id
PUT    /api/v1/admin/users/:id/status
PUT    /api/v1/admin/users/:id/role
POST   /api/v1/admin/users
GET    /api/v1/admin/users/:id/meetings

# 会议管理
GET    /api/v1/admin/meetings
GET    /api/v1/admin/meetings/:id
PUT    /api/v1/admin/meetings/:id/close
GET    /api/v1/admin/meetings/stats

# 系统管理
GET    /api/v1/admin/dashboard
GET    /api/v1/admin/logs
GET    /api/v1/admin/system/config
PUT    /api/v1/admin/system/config
```

### 5.5 Go ↔ mediasoup Node 通信 API

```
POST   /media/router/create
POST   /media/transport/create
POST   /media/transport/connect
POST   /media/producer/create
POST   /media/producer/close
POST   /media/consumer/create
POST   /media/consumer/resume
GET    /media/router/:id/capabilities
DELETE /media/room/:id
```

### 5.6 统一响应格式与错误码

```json
{
    "code": 0,
    "message": "ok",
    "data": {},
    "timestamp": 1740700000
}
```

| 错误码 | 含义 |
|--------|------|
| 0 | 成功 |
| 1001 | 参数错误 |
| 1002 | 未认证 |
| 1003 | 权限不足 |
| 2001 | 用户相关错误 |
| 3001 | IM 相关错误 |
| 4001 | 会议相关错误 |
| 5001 | 系统内部错误 |

---

## 六、前端页面结构

### 6.1 前台用户端页面（uniapp）

```
pages/
├── auth/
│   ├── login.vue                # 登录页
│   ├── register.vue             # 注册页
│   └── forgot-password.vue      # 忘记密码
├── chat/
│   ├── index.vue                # 会话列表
│   ├── conversation.vue         # 聊天对话页
│   └── group-create.vue         # 创建群聊
├── contact/
│   ├── index.vue                # 联系人列表
│   ├── add-friend.vue           # 添加好友
│   ├── friend-requests.vue      # 好友申请列表
│   └── detail.vue               # 好友资料页
├── meeting/
│   ├── index.vue                # 会议首页（即将开始 + 进行中 + 快速入口）
│   ├── create.vue               # 创建即时会议
│   ├── schedule.vue             # 预约会议
│   ├── room.vue                 # 会议房间（音视频画面）
│   ├── history.vue              # 历史会议
│   └── detail.vue               # 会议详情
├── notification/
│   └── index.vue                # 通知中心
├── profile/
│   ├── index.vue                # 个人中心
│   └── edit.vue                 # 编辑资料
└── index/
    └── index.vue                # 启动页/首页
```

TabBar 导航（底部4标签）：消息 | 联系人 | 会议 | 我的

### 6.2 后台管理端页面（Vue 3 + Element Plus）

```
views/
├── login.vue                     # 管理员登录
├── dashboard/index.vue           # 数据看板
├── user/
│   ├── list.vue                  # 用户列表
│   └── detail.vue                # 用户详情（含会议记录Tab）
├── meeting/
│   ├── list.vue                  # 会议列表
│   ├── detail.vue                # 会议详情
│   └── monitor.vue               # 实时监控
├── permission/
│   ├── role.vue                  # 角色管理
│   └── assign.vue                # 权限分配
├── system/
│   ├── config.vue                # 系统配置
│   └── logs.vue                  # 操作日志
└── layout/
    ├── index.vue                 # 布局框架
    ├── sidebar.vue               # 侧边栏
    └── header.vue                # 顶部栏
```

---

## 七、关键技术方案

### 7.1 WebSocket 连接管理

- 心跳保活：30秒一次 ping/pong
- 断线自动重连：指数退避（1s → 2s → 4s → 8s → 最大30s）
- 消息队列缓冲：断线期间消息先缓存，重连后重发
- 多页面共享：通过 Pinia store 管理连接状态

### 7.2 mediasoup-client 集成

```
services/
├── websocket.js           # WebSocket 连接管理（单例）
├── mediasoup-client.js    # mediasoup Device 管理
└── media-utils.js         # 音视频工具
```

会议加入完整信令流程：
1. HTTP: POST /api/v1/meetings/:code/join → 验证权限
2. WS: meeting.room.join → Go 通知 Node 创建 Router
3. WS: meeting.transport.create → Go 转发 Node 创建 Transport
4. mediasoup-client: device.load(routerRtpCapabilities)
5. sendTransport.produce(videoTrack/audioTrack) → WS 信令
6. 服务端通知其他成员 → 其他成员 consume
7. 音视频流直连 mediasoup Worker

### 7.3 状态管理（Pinia）

```
store/
├── user.js              # 用户信息、Token、登录状态
├── chat.js              # 会话列表、当前会话、消息缓存
├── contact.js           # 好友列表、好友请求
├── meeting.js           # 会议状态、参与者、本地媒体
├── notification.js      # 通知列表、未读数
└── websocket.js         # WebSocket 连接状态、消息分发
```

### 7.4 Docker Compose 编排

```yaml
services:
  go-service:        # Go 后端    → 8080
  media-server:      # mediasoup  → 3000 + 40000-40100/udp (RTP)
  postgres:          # PostgreSQL → 5432
  redis:             # Redis      → 6379
  nginx:             # 反向代理    → 80/443
```

---

## 八、开发分期规划

### 第一期（MVP）
- 用户注册/登录（邮箱+密码、用户名+密码）
- 即时聊天（单聊 + 群聊，文字/图片/文件）
- 联系人/好友管理
- 多人音视频会议（即时会议 + 预约会议）
- 消息通知系统

### 第二期
- 屏幕共享
- 微信授权登录
- 互动直播（主播开播、观众观看、弹幕互动）
- 会议录制与回放

### 第三期
- 微服务拆分
- Kubernetes 部署编排
- 跨服务器会议（多 Worker 集群）
- AI 辅助功能（语音转文字、会议纪要等）
