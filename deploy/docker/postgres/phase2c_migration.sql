-- ============================================================
-- Phase 2c 数据库迁移：群聊 + 已读回执
-- 执行环境：在已有 Phase 2b 数据库结构基础上增量升级
-- ============================================================

-- ============================================================
-- im_groups: 群聊信息表
-- 与 im_conversations (type=2) 一对一关联，存储群聊独有属性
-- ============================================================
CREATE TABLE IF NOT EXISTS im_groups (
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

CREATE INDEX IF NOT EXISTS idx_im_groups_owner ON im_groups(owner_id);
CREATE INDEX IF NOT EXISTS idx_im_groups_name ON im_groups USING gin(to_tsvector('simple', name));

-- ============================================================
-- im_group_join_requests: 入群申请表
-- 记录用户主动申请加入群聊的审批流程
-- ============================================================
CREATE TABLE IF NOT EXISTS im_group_join_requests (
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

CREATE INDEX IF NOT EXISTS idx_group_join_req_group ON im_group_join_requests(group_id, status);
CREATE INDEX IF NOT EXISTS idx_group_join_req_user ON im_group_join_requests(user_id);

-- ============================================================
-- im_message_reads: 群聊消息已读记录表（消息级别）
-- 复合主键 (message_id, user_id)，存储每条群消息的已读用户
-- ============================================================
CREATE TABLE IF NOT EXISTS im_message_reads (
    message_id BIGINT NOT NULL,
    user_id    BIGINT NOT NULL,
    read_at    TIMESTAMP(0) NOT NULL DEFAULT NOW(),
    PRIMARY KEY (message_id, user_id)
);

COMMENT ON TABLE  im_message_reads            IS '群聊消息已读记录表（消息级别）';
COMMENT ON COLUMN im_message_reads.message_id IS '消息 ID';
COMMENT ON COLUMN im_message_reads.user_id    IS '已读用户 ID';
COMMENT ON COLUMN im_message_reads.read_at    IS '已读时间';

CREATE INDEX IF NOT EXISTS idx_msg_reads_user ON im_message_reads(user_id, read_at);

-- ============================================================
-- im_conversation_members: 新增群聊相关字段
-- role/nickname/is_muted/is_do_not_disturb/joined_at/at_me_count
-- ============================================================
ALTER TABLE im_conversation_members
    ADD COLUMN IF NOT EXISTS role              SMALLINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS nickname          VARCHAR(50) DEFAULT '',
    ADD COLUMN IF NOT EXISTS is_muted          BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS is_do_not_disturb BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS joined_at         TIMESTAMP(0) DEFAULT NULL,
    ADD COLUMN IF NOT EXISTS at_me_count       INT DEFAULT 0;

COMMENT ON COLUMN im_conversation_members.role              IS '成员角色：0=普通成员，1=管理员，2=群主';
COMMENT ON COLUMN im_conversation_members.nickname          IS '群内昵称（仅群聊有效）';
COMMENT ON COLUMN im_conversation_members.is_muted          IS '是否被禁言';
COMMENT ON COLUMN im_conversation_members.is_do_not_disturb IS '是否消息免打扰（仍计未读但灰色显示，不推送通知）';
COMMENT ON COLUMN im_conversation_members.joined_at         IS '加入群聊时间';
COMMENT ON COLUMN im_conversation_members.at_me_count       IS '被@提醒未读计数，打开会话后清零';

-- ============================================================
-- im_messages: 新增 @提醒用户列表字段
-- ============================================================
ALTER TABLE im_messages
    ADD COLUMN IF NOT EXISTS at_user_ids BIGINT[] DEFAULT NULL;

COMMENT ON COLUMN im_messages.at_user_ids IS '@提醒用户 ID 列表，NULL=无@，包含 0 表示 @所有人';
