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
| 存储 | PostgreSQL 17 + Redis 7 |
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

> 所有表和字段均添加 `COMMENT` 注释，枚举类字段详细标注各值含义。

#### auth 模块 — 用户与权限

```sql
-- ============================================================
-- auth_users: 用户主表
-- 存储系统所有用户（包括普通用户和管理员），通过角色表区分权限
-- ============================================================
CREATE TABLE auth_users (
    id              BIGSERIAL PRIMARY KEY,
    username        VARCHAR(50)  UNIQUE NOT NULL,
    email           VARCHAR(100) UNIQUE NOT NULL,
    password_hash   VARCHAR(255) NOT NULL,
    nickname        VARCHAR(50)  NOT NULL DEFAULT '',
    avatar          VARCHAR(500) NOT NULL DEFAULT '',
    gender          SMALLINT     NOT NULL DEFAULT 0,
    phone           VARCHAR(20)  DEFAULT NULL,
    status          SMALLINT     NOT NULL DEFAULT 1,
    last_login_at   TIMESTAMP  DEFAULT NULL,
    last_login_ip   VARCHAR(50)  DEFAULT NULL,
    created_at      TIMESTAMP  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP  NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  auth_users                IS '用户主表，存储所有用户信息（普通用户与管理员共用）';
COMMENT ON COLUMN auth_users.id             IS '用户唯一标识，自增主键';
COMMENT ON COLUMN auth_users.username       IS '用户名，全局唯一，用于登录';
COMMENT ON COLUMN auth_users.email          IS '邮箱地址，全局唯一，用于登录和通知';
COMMENT ON COLUMN auth_users.password_hash  IS '密码哈希值，使用 bcrypt 加密存储';
COMMENT ON COLUMN auth_users.nickname       IS '用户昵称，用于前端显示';
COMMENT ON COLUMN auth_users.avatar         IS '头像 URL 地址';
COMMENT ON COLUMN auth_users.gender         IS '性别：0=未知，1=男，2=女';
COMMENT ON COLUMN auth_users.phone          IS '手机号码，可选';
COMMENT ON COLUMN auth_users.status         IS '账号状态：1=正常，2=禁用（管理员封禁），3=注销（用户主动注销）';
COMMENT ON COLUMN auth_users.last_login_at  IS '最后一次登录时间';
COMMENT ON COLUMN auth_users.last_login_ip  IS '最后一次登录 IP 地址';
COMMENT ON COLUMN auth_users.created_at     IS '账号创建时间';
COMMENT ON COLUMN auth_users.updated_at     IS '信息最后更新时间';

-- ============================================================
-- auth_roles: 角色表
-- 系统预置角色，用于 RBAC 权限控制
-- ============================================================
CREATE TABLE auth_roles (
    id          SERIAL PRIMARY KEY,
    code        VARCHAR(50) UNIQUE NOT NULL,
    name        VARCHAR(50) NOT NULL,
    description VARCHAR(200) DEFAULT '',
    created_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  auth_roles             IS '角色表，定义系统中所有角色类型';
COMMENT ON COLUMN auth_roles.id          IS '角色唯一标识，自增主键';
COMMENT ON COLUMN auth_roles.code        IS '角色代码，唯一标识：user=普通用户，admin=管理员，super_admin=超级管理员';
COMMENT ON COLUMN auth_roles.name        IS '角色显示名称，如"普通用户""管理员""超级管理员"';
COMMENT ON COLUMN auth_roles.description IS '角色描述说明';
COMMENT ON COLUMN auth_roles.created_at  IS '创建时间';

-- 预置角色数据
INSERT INTO auth_roles (code, name, description) VALUES
    ('user',        '普通用户',   '系统普通用户，可以使用聊天、会议等功能'),
    ('admin',       '管理员',    '后台管理员，可以管理用户、监控会议等'),
    ('super_admin', '超级管理员', '最高权限管理员，可以管理角色和系统配置');

-- ============================================================
-- auth_user_roles: 用户角色关联表
-- 多对多关系，一个用户可拥有多个角色
-- ============================================================
CREATE TABLE auth_user_roles (
    user_id    BIGINT NOT NULL REFERENCES auth_users(id),
    role_id    INT    NOT NULL REFERENCES auth_roles(id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, role_id)
);

#### 权限扩展规划（后期迭代）

> 当前 MVP 阶段使用粗粒度 3 角色（user / admin / super_admin）即可满足需求。
> 后期管理员增多、需要区分职责时，引入细粒度权限点机制，扩展如下两张表：

```sql
-- 预留：权限点表（后期扩展）
-- CREATE TABLE auth_permissions (
--     id          SERIAL PRIMARY KEY,
--     code        VARCHAR(100) UNIQUE NOT NULL,     -- 权限点代码，如 admin:user:list
--     name        VARCHAR(100) NOT NULL,            -- 权限点名称，如"查看用户列表"
--     module      VARCHAR(50)  NOT NULL,            -- 所属模块：user / meeting / system
--     description VARCHAR(200) DEFAULT '',
--     created_at  TIMESTAMP NOT NULL DEFAULT NOW()
-- );
--
-- 预留：角色权限关联表（后期扩展）
-- CREATE TABLE auth_role_permissions (
--     role_id       INT NOT NULL REFERENCES auth_roles(id),
--     permission_id INT NOT NULL REFERENCES auth_permissions(id),
--     PRIMARY KEY (role_id, permission_id)
-- );
```

扩展时的权限点示例：
- `admin:user:list` / `admin:user:detail` / `admin:user:disable` / `admin:user:role`
- `admin:meeting:list` / `admin:meeting:close` / `admin:meeting:stats`
- `admin:system:config` / `admin:log:list` / `admin:dashboard`

中间件从 `RequireRole("admin")` 扩展为 `RequirePermission("admin:user:list")`，前端管理端菜单根据权限点动态渲染。

COMMENT ON TABLE  auth_user_roles            IS '用户角色关联表，建立用户与角色的多对多关系';
COMMENT ON COLUMN auth_user_roles.user_id    IS '关联的用户 ID';
COMMENT ON COLUMN auth_user_roles.role_id    IS '关联的角色 ID';
COMMENT ON COLUMN auth_user_roles.created_at IS '角色分配时间';
```

#### contact 模块 — 联系人与好友

```sql
-- ============================================================
-- contact_friendships: 好友关系表
-- 双向存储：A→B 和 B→A 各一条记录，便于查询"我的好友列表"
-- ============================================================
CREATE TABLE contact_friendships (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT   NOT NULL REFERENCES auth_users(id),
    friend_id   BIGINT   NOT NULL REFERENCES auth_users(id),
    remark      VARCHAR(50) DEFAULT '',
    group_id    BIGINT   DEFAULT NULL,
    status      SMALLINT NOT NULL DEFAULT 0,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, friend_id)
);

COMMENT ON TABLE  contact_friendships            IS '好友关系表，双向存储（A→B和B→A各一条记录）';
COMMENT ON COLUMN contact_friendships.id         IS '记录唯一标识';
COMMENT ON COLUMN contact_friendships.user_id    IS '发起方用户 ID';
COMMENT ON COLUMN contact_friendships.friend_id  IS '好友用户 ID';
COMMENT ON COLUMN contact_friendships.remark     IS '好友备注名，仅对当前用户可见';
COMMENT ON COLUMN contact_friendships.group_id   IS '所属好友分组 ID，关联 contact_groups 表';
COMMENT ON COLUMN contact_friendships.status     IS '好友关系状态：0=待确认（已发送申请），1=已接受（互为好友），2=已拒绝，3=已拉黑';
COMMENT ON COLUMN contact_friendships.created_at IS '记录创建时间（申请发送时间）';
COMMENT ON COLUMN contact_friendships.updated_at IS '最后更新时间（状态变更时间）';

-- ============================================================
-- contact_groups: 好友分组表
-- 每个用户可自定义好友分组
-- ============================================================
CREATE TABLE contact_groups (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT      NOT NULL REFERENCES auth_users(id),
    name       VARCHAR(50) NOT NULL,
    sort_order INT         NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  contact_groups            IS '好友分组表，用户可自定义分组管理好友';
COMMENT ON COLUMN contact_groups.id         IS '分组唯一标识';
COMMENT ON COLUMN contact_groups.user_id    IS '所属用户 ID';
COMMENT ON COLUMN contact_groups.name       IS '分组名称，如"同事""家人""朋友"等';
COMMENT ON COLUMN contact_groups.sort_order IS '排序权重，数值越小越靠前';
COMMENT ON COLUMN contact_groups.created_at IS '创建时间';
```

#### im 模块 — 即时通讯（统一会话模型）

单聊和群聊统一抽象为"会话"，本质都是"一组人在一个空间里收发消息"。这是微信、钉钉、Slack 等主流 IM 的标准模型。

```sql
-- ============================================================
-- im_conversations: 会话表
-- 统一抽象单聊和群聊，单聊时 name/avatar 为空（前端用对方信息展示）
-- ============================================================
CREATE TABLE im_conversations (
    id          BIGSERIAL PRIMARY KEY,
    type        SMALLINT    NOT NULL,
    name        VARCHAR(100) DEFAULT '',
    avatar      VARCHAR(500) DEFAULT '',
    owner_id    BIGINT      DEFAULT NULL,
    max_members INT         NOT NULL DEFAULT 200,
    status      SMALLINT    NOT NULL DEFAULT 1,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  im_conversations             IS '会话表，统一管理单聊和群聊会话';
COMMENT ON COLUMN im_conversations.id          IS '会话唯一标识';
COMMENT ON COLUMN im_conversations.type        IS '会话类型：1=单聊（两人私聊），2=群聊（多人群组）';
COMMENT ON COLUMN im_conversations.name        IS '会话名称，群聊时为群名，单聊时为空（前端取对方昵称展示）';
COMMENT ON COLUMN im_conversations.avatar      IS '会话头像 URL，群聊时为群头像，单聊时为空（前端取对方头像展示）';
COMMENT ON COLUMN im_conversations.owner_id    IS '群主用户 ID，仅群聊时有值，单聊时为 NULL';
COMMENT ON COLUMN im_conversations.max_members IS '最大成员数，单聊固定为2，群聊默认200';
COMMENT ON COLUMN im_conversations.status      IS '会话状态：1=正常，2=已解散（仅群聊可解散）';
COMMENT ON COLUMN im_conversations.created_at  IS '会话创建时间';
COMMENT ON COLUMN im_conversations.updated_at  IS '最后更新时间';

-- ============================================================
-- im_conversation_members: 会话成员表
-- 记录每个会话中的参与成员及其个性化设置
-- ============================================================
CREATE TABLE im_conversation_members (
    id              BIGSERIAL PRIMARY KEY,
    conversation_id BIGINT   NOT NULL REFERENCES im_conversations(id),
    user_id         BIGINT   NOT NULL REFERENCES auth_users(id),
    role            SMALLINT NOT NULL DEFAULT 0,
    nickname        VARCHAR(50) DEFAULT '',
    is_muted        BOOLEAN  NOT NULL DEFAULT FALSE,
    is_pinned       BOOLEAN  NOT NULL DEFAULT FALSE,
    last_read_msg_id BIGINT  DEFAULT 0,
    joined_at       TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (conversation_id, user_id)
);

COMMENT ON TABLE  im_conversation_members                  IS '会话成员表，记录成员列表及每人的个性化设置';
COMMENT ON COLUMN im_conversation_members.id               IS '记录唯一标识';
COMMENT ON COLUMN im_conversation_members.conversation_id  IS '所属会话 ID';
COMMENT ON COLUMN im_conversation_members.user_id          IS '成员用户 ID';
COMMENT ON COLUMN im_conversation_members.role             IS '成员角色：0=普通成员，1=管理员（群聊可设置），2=群主';
COMMENT ON COLUMN im_conversation_members.nickname         IS '群内昵称，仅在该群聊中生效，为空则使用用户全局昵称';
COMMENT ON COLUMN im_conversation_members.is_muted         IS '是否被禁言：false=正常发言，true=已被禁言（仅管理员/群主可操作）';
COMMENT ON COLUMN im_conversation_members.is_pinned        IS '是否置顶该会话：false=不置顶，true=置顶（个人设置，不影响他人）';
COMMENT ON COLUMN im_conversation_members.last_read_msg_id IS '最后已读消息 ID，用于计算未读消息数';
COMMENT ON COLUMN im_conversation_members.joined_at        IS '加入会话的时间';

-- ============================================================
-- im_messages: 消息表
-- 系统数据量最大的表，存储所有聊天消息内容
-- ============================================================
CREATE TABLE im_messages (
    id              BIGSERIAL PRIMARY KEY,
    conversation_id BIGINT   NOT NULL,
    sender_id       BIGINT   NOT NULL,
    type            SMALLINT NOT NULL DEFAULT 1,
    content         TEXT     NOT NULL DEFAULT '',
    extra           JSONB    DEFAULT '{}',
    status          SMALLINT NOT NULL DEFAULT 1,
    created_at      TIMESTAMP NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  im_messages                 IS '聊天消息表，存储所有会话的消息记录（系统数据量最大的表）';
COMMENT ON COLUMN im_messages.id              IS '消息唯一标识，全局自增';
COMMENT ON COLUMN im_messages.conversation_id IS '所属会话 ID';
COMMENT ON COLUMN im_messages.sender_id       IS '发送者用户 ID';
COMMENT ON COLUMN im_messages.type            IS '消息类型：1=文本消息，2=图片消息，3=文件消息，4=语音消息，5=系统通知消息';
COMMENT ON COLUMN im_messages.content         IS '消息内容，文本消息为文字，其他类型为描述文字或为空';
COMMENT ON COLUMN im_messages.extra           IS '附加数据（JSON），图片消息存 {url,width,height}，文件消息存 {url,name,size}，语音消息存 {url,duration}';
COMMENT ON COLUMN im_messages.status          IS '消息状态：1=正常，2=已撤回（发送者撤回），3=已删除（管理员删除）';
COMMENT ON COLUMN im_messages.created_at      IS '消息发送时间';

CREATE INDEX idx_im_messages_conv_time ON im_messages(conversation_id, created_at DESC);
COMMENT ON INDEX idx_im_messages_conv_time IS '会话消息时间索引，用于按时间倒序查询会话历史消息';
```

#### meeting 模块 — 音视频会议

```sql
-- ============================================================
-- meeting_rooms: 会议房间表
-- 存储所有会议信息，支持即时会议和预约会议两种类型
-- ============================================================
CREATE TABLE meeting_rooms (
    id            BIGSERIAL PRIMARY KEY,
    room_code     VARCHAR(20) UNIQUE NOT NULL,
    title         VARCHAR(200) NOT NULL,
    host_id       BIGINT      NOT NULL REFERENCES auth_users(id),
    type          SMALLINT    NOT NULL DEFAULT 1,
    password      VARCHAR(50) DEFAULT NULL,
    max_members   INT         NOT NULL DEFAULT 50,
    status        SMALLINT    NOT NULL DEFAULT 0,
    scheduled_at  TIMESTAMP DEFAULT NULL,
    started_at    TIMESTAMP DEFAULT NULL,
    ended_at      TIMESTAMP DEFAULT NULL,
    settings      JSONB       DEFAULT '{}',
    created_at    TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMP NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  meeting_rooms               IS '会议房间表，存储所有会议的基本信息和状态';
COMMENT ON COLUMN meeting_rooms.id            IS '会议唯一标识，自增主键';
COMMENT ON COLUMN meeting_rooms.room_code     IS '会议号（用户可见），如"123-456-789"，用于分享和加入会议';
COMMENT ON COLUMN meeting_rooms.title         IS '会议标题';
COMMENT ON COLUMN meeting_rooms.host_id       IS '会议创建者/主持人用户 ID';
COMMENT ON COLUMN meeting_rooms.type          IS '会议类型：1=即时会议（立即创建立即开始），2=预约会议（设定未来时间）';
COMMENT ON COLUMN meeting_rooms.password      IS '会议密码，NULL 表示无密码，任何人可直接加入';
COMMENT ON COLUMN meeting_rooms.max_members   IS '最大参会人数，默认50人';
COMMENT ON COLUMN meeting_rooms.status        IS '会议状态：0=未开始（仅预约会议），1=进行中，2=已结束';
COMMENT ON COLUMN meeting_rooms.scheduled_at  IS '预约时间，仅预约会议有值，即时会议为 NULL';
COMMENT ON COLUMN meeting_rooms.started_at    IS '实际开始时间';
COMMENT ON COLUMN meeting_rooms.ended_at      IS '实际结束时间';
COMMENT ON COLUMN meeting_rooms.settings      IS '会议设置（JSON），如 {mute_on_join: true, allow_recording: false, auto_start: true}';
COMMENT ON COLUMN meeting_rooms.created_at    IS '会议创建时间';
COMMENT ON COLUMN meeting_rooms.updated_at    IS '信息最后更新时间';

-- ============================================================
-- meeting_participants: 会议参与者表
-- 记录每场会议的参与者及其参会信息
-- ============================================================
CREATE TABLE meeting_participants (
    id           BIGSERIAL PRIMARY KEY,
    room_id      BIGINT   NOT NULL REFERENCES meeting_rooms(id),
    user_id      BIGINT   NOT NULL REFERENCES auth_users(id),
    role         SMALLINT NOT NULL DEFAULT 0,
    joined_at    TIMESTAMP DEFAULT NULL,
    left_at      TIMESTAMP DEFAULT NULL,
    duration     INT      DEFAULT 0,
    UNIQUE (room_id, user_id)
);

COMMENT ON TABLE  meeting_participants           IS '会议参与者表，记录每场会议的所有参与者信息';
COMMENT ON COLUMN meeting_participants.id        IS '记录唯一标识';
COMMENT ON COLUMN meeting_participants.room_id   IS '所属会议 ID';
COMMENT ON COLUMN meeting_participants.user_id   IS '参与者用户 ID';
COMMENT ON COLUMN meeting_participants.role      IS '参会角色：0=普通参与者，1=主持人（会议创建者），2=联合主持人（主持人指定）';
COMMENT ON COLUMN meeting_participants.joined_at IS '加入会议的时间';
COMMENT ON COLUMN meeting_participants.left_at   IS '离开会议的时间，NULL 表示仍在会议中';
COMMENT ON COLUMN meeting_participants.duration  IS '累计参会时长（秒），离开时自动计算';
```

会议类型状态流转：
- **即时会议**（type=1）：创建 → 进行中(status=1) → 已结束(status=2)
- **预约会议**（type=2）：创建 → 未开始(status=0) → 进行中(status=1) → 已结束(status=2)

#### notify 模块 — 消息通知

```sql
-- ============================================================
-- notify_notifications: 通知表
-- 存储所有推送给用户的通知消息
-- ============================================================
CREATE TABLE notify_notifications (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT      NOT NULL REFERENCES auth_users(id),
    type        VARCHAR(50) NOT NULL,
    title       VARCHAR(200) NOT NULL,
    content     TEXT        DEFAULT '',
    extra       JSONB       DEFAULT '{}',
    is_read     BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  notify_notifications            IS '通知消息表，存储推送给用户的所有类型通知';
COMMENT ON COLUMN notify_notifications.id         IS '通知唯一标识';
COMMENT ON COLUMN notify_notifications.user_id    IS '接收通知的用户 ID';
COMMENT ON COLUMN notify_notifications.type       IS '通知类型：meeting_invite=会议邀请，friend_request=好友申请，friend_accepted=好友已接受，meeting_reminder=会议提醒，system=系统通知';
COMMENT ON COLUMN notify_notifications.title      IS '通知标题';
COMMENT ON COLUMN notify_notifications.content    IS '通知正文内容';
COMMENT ON COLUMN notify_notifications.extra      IS '附加数据（JSON），如会议邀请存 {room_code, room_title}，好友申请存 {from_user_id, from_username}';
COMMENT ON COLUMN notify_notifications.is_read    IS '是否已读：false=未读，true=已读';
COMMENT ON COLUMN notify_notifications.created_at IS '通知创建时间';

CREATE INDEX idx_notify_user_read ON notify_notifications(user_id, is_read, created_at DESC);
COMMENT ON INDEX idx_notify_user_read IS '用户未读通知索引，优化"获取未读通知列表"查询';
```

#### admin 模块 — 管理操作日志

```sql
-- ============================================================
-- admin_operation_logs: 管理操作日志表
-- 记录后台管理员的所有操作行为，用于审计和追踪
-- ============================================================
CREATE TABLE admin_operation_logs (
    id          BIGSERIAL PRIMARY KEY,
    admin_id    BIGINT      NOT NULL REFERENCES auth_users(id),
    module      VARCHAR(50) NOT NULL,
    action      VARCHAR(50) NOT NULL,
    target_type VARCHAR(50) DEFAULT '',
    target_id   BIGINT      DEFAULT NULL,
    detail      JSONB       DEFAULT '{}',
    ip          VARCHAR(50) DEFAULT '',
    created_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  admin_operation_logs             IS '管理操作日志表，记录所有后台管理员的操作行为';
COMMENT ON COLUMN admin_operation_logs.id          IS '日志唯一标识';
COMMENT ON COLUMN admin_operation_logs.admin_id    IS '操作管理员的用户 ID';
COMMENT ON COLUMN admin_operation_logs.module      IS '操作所属模块：user=用户管理，meeting=会议管理，permission=权限管理，system=系统配置';
COMMENT ON COLUMN admin_operation_logs.action      IS '操作类型：create=创建，update=修改，delete=删除，disable=禁用，enable=启用，close=关闭';
COMMENT ON COLUMN admin_operation_logs.target_type IS '操作目标类型：user=用户，meeting=会议，role=角色，config=配置';
COMMENT ON COLUMN admin_operation_logs.target_id   IS '操作目标 ID，关联对应表的主键';
COMMENT ON COLUMN admin_operation_logs.detail      IS '操作详情（JSON），如 {before: {...}, after: {...}} 记录变更前后数据';
COMMENT ON COLUMN admin_operation_logs.ip          IS '操作者 IP 地址';
COMMENT ON COLUMN admin_operation_logs.created_at  IS '操作时间';
```

### 4.2 Redis 数据结构

```
# 用户认证（按 client_type 隔离前台和管理端）
echo:auth:token:{client_type}:{user_id}     → JWT Access Token (STRING, TTL 由配置决定)
echo:auth:refresh:{client_type}:{user_id}   → Refresh Token (STRING, TTL 由配置决定)
# client_type: frontend（前台用户端）/ admin（后台管理端）

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
    "time": "2026-02-27 18:06:40"
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
    "time": "2026-02-27 18:00:00"
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
