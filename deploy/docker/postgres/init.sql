-- ============================================================
-- EchoChat 数据库初始化脚本
-- 第一阶段：auth 模块表结构
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
    description VARCHAR(200) DEFAULT '',
    created_at  TIMESTAMP(0) NOT NULL DEFAULT NOW()
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
    ('admin',       '管理员',     '后台管理员，可以管理用户、监控会议等'),
    ('super_admin', '超级管理员', '最高权限管理员，可以管理角色和系统配置');

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
-- 插入默认超级管理员
-- 密码: admin123456 (bcrypt hash)
-- 实际部署时应更换密码
-- ============================================================
INSERT INTO auth_users (username, email, password_hash, nickname, status)
VALUES ('admin', 'admin@echochat.com', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', '超级管理员', 1);

INSERT INTO auth_user_roles (user_id, role_id)
VALUES (1, (SELECT id FROM auth_roles WHERE code = 'super_admin'));
