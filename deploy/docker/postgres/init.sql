-- ============================================================
-- EchoChat 数据库初始化脚本
-- 包含模块：auth（用户认证）、contact（联系人）
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
