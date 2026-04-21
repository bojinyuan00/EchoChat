-- ============================================================
-- Phase 2e-2 数据库迁移：会议 MVP（meeting_rooms / meeting_participants / meeting_chats）
-- 执行环境：在已有 Phase 2e-1（通知中心）数据库基础上增量升级
-- 全部语句使用 IF NOT EXISTS / 幂等形式，可重复执行
-- 时间字段沿用项目风格 TIMESTAMP(0)
-- ============================================================

CREATE TABLE IF NOT EXISTS meeting_rooms (
    id             BIGSERIAL     PRIMARY KEY,
    room_code      VARCHAR(20)   UNIQUE NOT NULL,
    title          VARCHAR(200)  NOT NULL,
    host_id        BIGINT        NOT NULL REFERENCES auth_users(id),
    type           SMALLINT      NOT NULL DEFAULT 1,
    password_hash  VARCHAR(255)  DEFAULT NULL,
    max_members    INT           NOT NULL DEFAULT 50,
    status         SMALLINT      NOT NULL DEFAULT 0,
    scheduled_at   TIMESTAMP(0)  DEFAULT NULL,
    started_at     TIMESTAMP(0)  DEFAULT NULL,
    ended_at       TIMESTAMP(0)  DEFAULT NULL,
    ended_reason   VARCHAR(20)   DEFAULT NULL,
    settings       JSONB         NOT NULL DEFAULT '{}',
    created_at     TIMESTAMP(0)  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMP(0)  NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  meeting_rooms               IS '会议房间表';
COMMENT ON COLUMN meeting_rooms.room_code     IS '用户可见的会议号 XXX-XXX-XXX，全局唯一';
COMMENT ON COLUMN meeting_rooms.title         IS '会议标题';
COMMENT ON COLUMN meeting_rooms.host_id       IS '主持人用户 ID，引用 auth_users.id';
COMMENT ON COLUMN meeting_rooms.type          IS '会议类型：1=即时（MVP 仅此），2=预约（Phase 2e-3）';
COMMENT ON COLUMN meeting_rooms.password_hash IS '入会密码 bcrypt 哈希，NULL 表示无密码';
COMMENT ON COLUMN meeting_rooms.max_members   IS '房间最大成员数；MVP 应用层强制 8，schema 默认 50 供后续扩展';
COMMENT ON COLUMN meeting_rooms.status        IS '0=未开始（仅预约），1=进行中，2=已结束';
COMMENT ON COLUMN meeting_rooms.scheduled_at  IS '预约开始时间，即时会议为 NULL';
COMMENT ON COLUMN meeting_rooms.started_at    IS '实际开始时间，host 首次加入时回填';
COMMENT ON COLUMN meeting_rooms.ended_at      IS '实际结束时间';
COMMENT ON COLUMN meeting_rooms.ended_reason  IS '结束原因：host_ended / empty_ttl / admin_force / system_error';
COMMENT ON COLUMN meeting_rooms.settings      IS 'JSON 配置：{mute_on_join, allow_chat, record_enabled 等}';

CREATE INDEX IF NOT EXISTS idx_meeting_rooms_host_status ON meeting_rooms (host_id, status, created_at DESC);

-- ============================================================
-- meeting_participants: 参与者记录
-- ON DELETE CASCADE：room 删除时自动清理参与者
-- ============================================================
CREATE TABLE IF NOT EXISTS meeting_participants (
    id          BIGSERIAL    PRIMARY KEY,
    room_id     BIGINT       NOT NULL REFERENCES meeting_rooms(id) ON DELETE CASCADE,
    user_id     BIGINT       NOT NULL REFERENCES auth_users(id),
    role        SMALLINT     NOT NULL DEFAULT 0,
    joined_at   TIMESTAMP(0) NOT NULL DEFAULT NOW(),
    left_at     TIMESTAMP(0) DEFAULT NULL,
    left_reason VARCHAR(20)  DEFAULT NULL,
    duration    INT          NOT NULL DEFAULT 0,
    UNIQUE (room_id, user_id)
);

COMMENT ON TABLE  meeting_participants             IS '会议参与者记录（一次入会一行，重入更新）';
COMMENT ON COLUMN meeting_participants.room_id     IS '所属会议房间 ID，引用 meeting_rooms.id，CASCADE 删除';
COMMENT ON COLUMN meeting_participants.user_id     IS '参与者用户 ID，引用 auth_users.id';
COMMENT ON COLUMN meeting_participants.role        IS '0=普通，1=主持人（MVP 仅此两档），2=联合主持人（第二期）';
COMMENT ON COLUMN meeting_participants.joined_at   IS '加入时间，记录创建即代表加入';
COMMENT ON COLUMN meeting_participants.left_at     IS '离开时间，NULL 表示仍在会议中';
COMMENT ON COLUMN meeting_participants.left_reason IS '离会原因：self / kicked / host_end / empty_ttl / disconnect';
COMMENT ON COLUMN meeting_participants.duration    IS '本次参会时长（秒），left_at 写入时同步计算';

CREATE INDEX IF NOT EXISTS idx_meeting_participants_room ON meeting_participants (room_id, joined_at ASC);
CREATE INDEX IF NOT EXISTS idx_meeting_participants_user ON meeting_participants (user_id, joined_at DESC);

-- ============================================================
-- meeting_chats: 会议内文字聊天
-- 清理策略见 CURRENT_STATUS Phase 2e-2 说明
-- ============================================================
CREATE TABLE IF NOT EXISTS meeting_chats (
    id         BIGSERIAL    PRIMARY KEY,
    room_id    BIGINT       NOT NULL REFERENCES meeting_rooms(id) ON DELETE CASCADE,
    user_id    BIGINT       NOT NULL REFERENCES auth_users(id),
    content    TEXT         NOT NULL,
    created_at TIMESTAMP(0) NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  meeting_chats            IS '会议内文字聊天消息（会议结束 24 小时后批量清理）';
COMMENT ON COLUMN meeting_chats.room_id    IS '所属会议房间 ID，引用 meeting_rooms.id，CASCADE 删除';
COMMENT ON COLUMN meeting_chats.user_id    IS '发送者用户 ID，引用 auth_users.id';
COMMENT ON COLUMN meeting_chats.content    IS '消息内容（纯文本，MVP 不支持富媒体）';
COMMENT ON COLUMN meeting_chats.created_at IS '消息发送时间';

CREATE INDEX IF NOT EXISTS idx_meeting_chats_room_created ON meeting_chats (room_id, created_at ASC);
