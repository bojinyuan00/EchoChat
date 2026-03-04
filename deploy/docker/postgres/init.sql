-- ============================================================
-- EchoChat 数据库初始化脚本
-- 包含模块：auth（用户认证）、contact（联系人）、im（即时通讯）、group（群聊）
-- ============================================================

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
    last_login_at   TIMESTAMP(0) DEFAULT NULL,
    last_login_ip   VARCHAR(50)  DEFAULT NULL,
    created_at      TIMESTAMP(0) NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP(0) NOT NULL DEFAULT NOW()
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
    level       INT NOT NULL DEFAULT 100,
    description VARCHAR(200) DEFAULT '',
    created_at  TIMESTAMP(0) NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  auth_roles             IS '角色表，定义系统中所有角色类型';
COMMENT ON COLUMN auth_roles.id          IS '角色唯一标识，自增主键';
COMMENT ON COLUMN auth_roles.code        IS '角色代码，唯一标识：user=普通用户，admin=管理员，super_admin=超级管理员';
COMMENT ON COLUMN auth_roles.name        IS '角色显示名称，如"普通用户""管理员""超级管理员"';
COMMENT ON COLUMN auth_roles.level       IS '角色等级，值越小权限越高：1=超管, 10=管理员, 100=普通用户，预留间隔便于扩展';
COMMENT ON COLUMN auth_roles.description IS '角色描述说明';
COMMENT ON COLUMN auth_roles.created_at  IS '创建时间';

-- 预置角色数据（level 值越小权限越高）
INSERT INTO auth_roles (code, name, level, description) VALUES
    ('user',        '普通用户',   100, '系统普通用户，可以使用聊天、会议等功能'),
    ('admin',       '管理员',     10,  '后台管理员，可以管理用户、监控会议等'),
    ('super_admin', '超级管理员', 1,   '最高权限管理员，可以管理角色和系统配置');

-- ============================================================
-- auth_user_roles: 用户角色关联表
-- 多对多关系，一个用户可拥有多个角色
-- ============================================================
CREATE TABLE auth_user_roles (
    user_id    BIGINT NOT NULL REFERENCES auth_users(id),
    role_id    INT    NOT NULL REFERENCES auth_roles(id),
    created_at TIMESTAMP(0) NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, role_id)
);

COMMENT ON TABLE  auth_user_roles            IS '用户角色关联表，建立用户与角色的多对多关系';
COMMENT ON COLUMN auth_user_roles.user_id    IS '关联的用户 ID';
COMMENT ON COLUMN auth_user_roles.role_id    IS '关联的角色 ID';
COMMENT ON COLUMN auth_user_roles.created_at IS '角色分配时间';

-- ============================================================
-- 插入默认超级管理员（系统预置唯一账号）
-- 用户名: super_admin  密码: admin123456 (bcrypt hash)
-- 实际部署时应更换密码
-- ============================================================
INSERT INTO auth_users (username, email, password_hash, nickname, status)
VALUES ('super_admin', 'super_admin@echochat.com', '$2a$10$Osjn5JVXuEHhtPwBW5Lyo.4gnyYtpFAZYrmbf5fzvN.M5DqcSggb2', '超级管理员', 1);

INSERT INTO auth_user_roles (user_id, role_id)
VALUES (1, (SELECT id FROM auth_roles WHERE code = 'super_admin'));

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
    message     VARCHAR(200) DEFAULT '',
    created_at  TIMESTAMP(0) NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP(0) NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, friend_id)
);

COMMENT ON TABLE  contact_friendships            IS '好友关系表，双向存储（A→B和B→A各一条记录）';
COMMENT ON COLUMN contact_friendships.id         IS '记录唯一标识';
COMMENT ON COLUMN contact_friendships.user_id    IS '发起方用户 ID';
COMMENT ON COLUMN contact_friendships.friend_id  IS '好友用户 ID';
COMMENT ON COLUMN contact_friendships.remark     IS '好友备注名，仅对当前用户可见';
COMMENT ON COLUMN contact_friendships.group_id   IS '所属好友分组 ID，关联 contact_groups 表';
COMMENT ON COLUMN contact_friendships.status     IS '好友关系状态：0=待确认（已发送申请），1=已接受（互为好友），2=已拒绝，3=已拉黑';
COMMENT ON COLUMN contact_friendships.message    IS '好友申请附言';
COMMENT ON COLUMN contact_friendships.created_at IS '记录创建时间（申请发送时间）';
COMMENT ON COLUMN contact_friendships.updated_at IS '最后更新时间（状态变更时间）';

CREATE INDEX idx_friendships_user_status ON contact_friendships (user_id, status);
CREATE INDEX idx_friendships_friend_status ON contact_friendships (friend_id, status);

-- ============================================================
-- contact_groups: 好友分组表
-- 每个用户可自定义好友分组
-- ============================================================
CREATE TABLE contact_groups (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT      NOT NULL REFERENCES auth_users(id),
    name       VARCHAR(50) NOT NULL,
    sort_order INT         NOT NULL DEFAULT 0,
    created_at TIMESTAMP(0) NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  contact_groups            IS '好友分组表，用户可自定义分组管理好友';
COMMENT ON COLUMN contact_groups.id         IS '分组唯一标识';
COMMENT ON COLUMN contact_groups.user_id    IS '所属用户 ID';
COMMENT ON COLUMN contact_groups.name       IS '分组名称，如"同事""家人""朋友"等';
COMMENT ON COLUMN contact_groups.sort_order IS '排序权重，数值越小越靠前';
COMMENT ON COLUMN contact_groups.created_at IS '创建时间';

CREATE INDEX idx_contact_groups_user ON contact_groups (user_id);

-- ============================================================
-- im_conversations: 会话表
-- 存储单聊（type=1）和群聊（type=2，预留）会话
-- 冗余 last_msg_* 字段以避免会话列表查询时 JOIN 消息表
-- ============================================================
CREATE TABLE im_conversations (
    id                 BIGSERIAL PRIMARY KEY,
    type               SMALLINT     NOT NULL DEFAULT 1,
    creator_id         BIGINT       NOT NULL,
    last_message_id    BIGINT       DEFAULT NULL,
    last_msg_content   TEXT         DEFAULT '',
    last_msg_time      TIMESTAMP(0) DEFAULT NULL,
    last_msg_sender_id BIGINT       DEFAULT NULL,
    created_at         TIMESTAMP(0) NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMP(0) NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  im_conversations                    IS '会话表，支持单聊（type=1）和群聊（type=2，预留）';
COMMENT ON COLUMN im_conversations.id                 IS '会话唯一标识，自增主键';
COMMENT ON COLUMN im_conversations.type               IS '会话类型：1=单聊，2=群聊（预留）';
COMMENT ON COLUMN im_conversations.creator_id         IS '会话创建者用户 ID';
COMMENT ON COLUMN im_conversations.last_message_id    IS '最后一条消息 ID，用于快速定位';
COMMENT ON COLUMN im_conversations.last_msg_content   IS '最后消息预览文本，用于会话列表展示';
COMMENT ON COLUMN im_conversations.last_msg_time      IS '最后消息时间，用于会话列表排序';
COMMENT ON COLUMN im_conversations.last_msg_sender_id IS '最后消息发送者 ID';
COMMENT ON COLUMN im_conversations.created_at         IS '会话创建时间';
COMMENT ON COLUMN im_conversations.updated_at         IS '会话信息更新时间';

CREATE INDEX idx_im_conversations_updated ON im_conversations (updated_at DESC);

-- ============================================================
-- im_conversation_members: 会话成员表
-- 每个会话的每个成员一条记录，存储个人视图（置顶/未读/软删除）
-- 单聊两条记录，群聊 N 条记录
-- ============================================================
CREATE TABLE im_conversation_members (
    id               BIGSERIAL PRIMARY KEY,
    conversation_id  BIGINT       NOT NULL REFERENCES im_conversations(id),
    user_id          BIGINT       NOT NULL,
    is_pinned        BOOLEAN      DEFAULT FALSE,
    is_deleted       BOOLEAN      DEFAULT FALSE,
    unread_count     INT          DEFAULT 0,
    last_read_msg_id     BIGINT       DEFAULT 0,
    clear_before_msg_id  BIGINT       DEFAULT 0,
    role                 SMALLINT     NOT NULL DEFAULT 0,
    nickname             VARCHAR(50)  DEFAULT '',
    is_muted             BOOLEAN      NOT NULL DEFAULT FALSE,
    is_do_not_disturb    BOOLEAN      NOT NULL DEFAULT FALSE,
    joined_at            TIMESTAMP(0) DEFAULT NULL,
    at_me_count          INT          DEFAULT 0,
    created_at           TIMESTAMP(0) NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMP(0) NOT NULL DEFAULT NOW(),
    UNIQUE (conversation_id, user_id)
);

COMMENT ON TABLE  im_conversation_members                        IS '会话成员表，存储每个成员对会话的个人视图';
COMMENT ON COLUMN im_conversation_members.id                     IS '记录唯一标识';
COMMENT ON COLUMN im_conversation_members.conversation_id        IS '所属会话 ID';
COMMENT ON COLUMN im_conversation_members.user_id                IS '成员用户 ID';
COMMENT ON COLUMN im_conversation_members.is_pinned              IS '是否置顶该会话';
COMMENT ON COLUMN im_conversation_members.is_deleted             IS '是否删除该会话（软删除，不影响对方视图）';
COMMENT ON COLUMN im_conversation_members.unread_count           IS '该成员在此会话中的未读消息数';
COMMENT ON COLUMN im_conversation_members.last_read_msg_id       IS '该成员最后已读消息 ID';
COMMENT ON COLUMN im_conversation_members.clear_before_msg_id    IS '清空聊天记录时的消息截止 ID（个人视图，不影响对方）';
COMMENT ON COLUMN im_conversation_members.role                   IS '成员角色：0=普通成员，1=管理员，2=群主';
COMMENT ON COLUMN im_conversation_members.nickname               IS '群内昵称（仅群聊有效）';
COMMENT ON COLUMN im_conversation_members.is_muted               IS '是否被禁言';
COMMENT ON COLUMN im_conversation_members.is_do_not_disturb      IS '是否消息免打扰（仍计未读但灰色显示，不推送通知）';
COMMENT ON COLUMN im_conversation_members.joined_at              IS '加入群聊时间';
COMMENT ON COLUMN im_conversation_members.at_me_count            IS '被@提醒未读计数，打开会话后清零';
COMMENT ON COLUMN im_conversation_members.created_at             IS '加入会话时间';
COMMENT ON COLUMN im_conversation_members.updated_at             IS '最后更新时间';

CREATE INDEX idx_im_conv_members_user ON im_conversation_members (user_id, is_deleted);
CREATE INDEX idx_im_conv_members_conv ON im_conversation_members (conversation_id);

-- ============================================================
-- im_messages: 消息表
-- 存储所有聊天消息，通过 conversation_id 关联会话
-- 游标分页查询核心索引：(conversation_id, created_at DESC)
-- ============================================================
CREATE TABLE im_messages (
    id              BIGSERIAL PRIMARY KEY,
    conversation_id BIGINT       NOT NULL REFERENCES im_conversations(id),
    sender_id       BIGINT       NOT NULL,
    type            SMALLINT     NOT NULL DEFAULT 1,
    content         TEXT         NOT NULL,
    extra           JSONB        DEFAULT NULL,
    status          SMALLINT     NOT NULL DEFAULT 1,
    client_msg_id   VARCHAR(64)  DEFAULT '',
    at_user_ids     BIGINT[]     DEFAULT NULL,
    created_at      TIMESTAMP(0) NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  im_messages                   IS '聊天消息表，存储所有消息内容';
COMMENT ON COLUMN im_messages.id                IS '消息唯一标识，自增主键';
COMMENT ON COLUMN im_messages.conversation_id   IS '所属会话 ID';
COMMENT ON COLUMN im_messages.sender_id         IS '发送者用户 ID';
COMMENT ON COLUMN im_messages.type              IS '消息类型：1=文本，2=图片（预留），3=语音（预留），4=视频（预留），5=文件（预留）';
COMMENT ON COLUMN im_messages.content           IS '消息内容（文本消息为纯文本，其他类型为 URL 或描述）';
COMMENT ON COLUMN im_messages.extra             IS '扩展数据（JSON 格式），预留图片尺寸、语音时长等元信息';
COMMENT ON COLUMN im_messages.status            IS '消息状态：1=正常，2=已撤回，3=已删除';
COMMENT ON COLUMN im_messages.client_msg_id     IS '客户端消息唯一 ID，用于幂等去重防止网络重试重复发送';
COMMENT ON COLUMN im_messages.at_user_ids       IS '@提醒用户 ID 列表，NULL=无@，包含 0 表示 @所有人';
COMMENT ON COLUMN im_messages.created_at        IS '消息发送时间';

CREATE INDEX idx_im_messages_conv_time ON im_messages (conversation_id, created_at DESC);
CREATE INDEX idx_im_messages_conv_id ON im_messages (conversation_id, id DESC);
CREATE INDEX idx_im_messages_content_search ON im_messages USING gin(to_tsvector('simple', content));

-- ============================================================
-- im_groups: 群聊信息表
-- 与 im_conversations (type=2) 一对一关联，存储群聊独有属性
-- ============================================================
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

COMMENT ON TABLE  im_groups                 IS '群聊信息表';
COMMENT ON COLUMN im_groups.id              IS '群唯一标识';
COMMENT ON COLUMN im_groups.conversation_id IS '关联 im_conversations.id';
COMMENT ON COLUMN im_groups.name            IS '群名称';
COMMENT ON COLUMN im_groups.avatar          IS '群头像 URL（MinIO）';
COMMENT ON COLUMN im_groups.owner_id        IS '群主用户 ID';
COMMENT ON COLUMN im_groups.notice          IS '群公告内容';
COMMENT ON COLUMN im_groups.max_members     IS '最大成员数，默认 200';
COMMENT ON COLUMN im_groups.is_searchable   IS '是否可被搜索发现';
COMMENT ON COLUMN im_groups.is_all_muted    IS '是否全体禁言';
COMMENT ON COLUMN im_groups.status          IS '群状态：1=正常，2=已解散';

CREATE INDEX idx_im_groups_owner ON im_groups(owner_id);
CREATE INDEX idx_im_groups_name ON im_groups USING gin(to_tsvector('simple', name));

-- ============================================================
-- im_group_join_requests: 入群申请表
-- 记录用户主动申请加入群聊的审批流程
-- ============================================================
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

-- ============================================================
-- im_message_reads: 群聊消息已读记录表（消息级别）
-- 复合主键 (message_id, user_id)，存储每条群消息的已读用户
-- ============================================================
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
