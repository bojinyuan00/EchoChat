# 管理端 — 用户管理 API

> 通用规范（认证方式、响应格式、错误码）见 [README.md](../README.md)
> 以下所有接口均需要 **JWT 认证 + admin/super_admin 角色**

---

## 接口列表

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | /api/v1/admin/users | admin | 获取用户列表 |
| GET | /api/v1/admin/users/:id | admin | 获取用户详情 |
| PUT | /api/v1/admin/users/:id/status | admin | 更新用户状态（受角色等级约束） |
| PUT | /api/v1/admin/users/:id/roles | admin / super_admin | 批量设置用户角色（受角色等级约束） |
| POST | /api/v1/admin/users | admin | 管理员创建用户 |
| GET | /api/v1/admin/roles | admin | 获取所有角色列表（含 level） |
| GET | /api/v1/admin/users/:id/meetings | admin | 获取用户会议记录（Phase 3 实现） |

---

## 1. 获取用户列表

`GET /api/v1/admin/users`

**权限：** admin

**查询参数：**

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| page | int | 是 | - | 页码（从 1 开始） |
| page_size | int | 是 | - | 每页数量（1-100） |
| keyword | string | 否 | 无 | 搜索关键词（模糊匹配用户名或邮箱） |
| status | int | 否 | 无 | 按状态筛选：1=正常，2=禁用，3=注销 |

**成功响应：**
```json
{
    "code": 0,
    "message": "success",
    "data": {
        "total": 100,
        "list": [
            {
                "id": 1,
                "username": "zhangsan",
                "email": "zhangsan@example.com",
                "nickname": "张三",
                "avatar": "",
                "gender": 1,
                "phone": "13800138000",
                "status": 1,
                "status_text": "正常",
                "roles": [
                    { "code": "user", "name": "普通用户", "level": 100 }
                ],
                "max_level": 100,
                "last_login_at": "2026-02-27 10:00:00",
                "last_login_ip": "192.168.1.100",
                "created_at": "2026-02-20 08:00:00",
                "updated_at": "2026-02-27 10:00:00"
            }
        ]
    }
}
```

---

## 2. 获取用户详情

`GET /api/v1/admin/users/:id`

**权限：** admin

**成功响应：**
```json
{
    "code": 0,
    "message": "success",
    "data": {
        "id": 1,
        "username": "zhangsan",
        "email": "zhangsan@example.com",
        "nickname": "张三",
        "avatar": "",
        "gender": 1,
        "phone": "13800138000",
        "status": 1,
        "status_text": "正常",
        "roles": [
            { "code": "user", "name": "普通用户", "level": 100 }
        ],
        "max_level": 100,
        "last_login_at": "2026-02-27 10:00:00",
        "last_login_ip": "192.168.1.100",
        "created_at": "2026-02-20 08:00:00",
        "updated_at": "2026-02-27 10:00:00"
    }
}
```

---

## 3. 更新用户状态

`PUT /api/v1/admin/users/:id/status`

**权限：** admin

**请求参数：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| status | int | 是 | 目标状态：1=正常（启用），2=禁用 |

**说明：**
- 禁用用户后，该用户的所有活跃 Token 将被清除
- 不能禁用自己的账号（返回 400）
- 操作者权限等级必须高于目标用户（level 数值更小），否则返回 403
- 禁用后用户尝试登录将返回 403

**成功响应：**
```json
{
    "code": 0,
    "message": "success",
    "trace_id": "...",
    "time": "2026-03-02 11:12:44"
}
```

---

## 4. 批量设置用户角色

`PUT /api/v1/admin/users/:id/roles`

**权限：** admin / super_admin

**请求参数：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| role_codes | string[] | 是 | 角色代码列表，如 `["user", "admin"]` |

**权限管控规则：**
- 操作者权限等级（level）必须严格高于目标用户（数值更小），否则返回 403
- 不能分配等级高于或等于操作者自身的角色，否则返回 403
- 角色列表为全量覆盖（先清除再设置）
- 注：`super_admin` 角色（level=1）不可通过 API 分配（任何人的 level 都 >= 1），超管只能通过数据库直接创建

**成功响应：**
```json
{
    "code": 0,
    "message": "success"
}
```

**错误响应：**
- 403：`权限不足，无法操作更高等级的用户` 或 `不能分配高于自身等级的角色`

---

## 5. 获取所有角色列表

`GET /api/v1/admin/roles`

**权限：** admin

**成功响应：**
```json
{
    "code": 0,
    "message": "success",
    "data": [
        { "code": "super_admin", "name": "超级管理员", "level": 1 },
        { "code": "admin", "name": "管理员", "level": 10 },
        { "code": "user", "name": "普通用户", "level": 100 }
    ]
}
```

---

## 6. 管理员创建用户

`POST /api/v1/admin/users`

**权限：** admin

**请求参数：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 是 | 用户名，3-50 字符 |
| email | string | 是 | 邮箱地址 |
| password | string | 是 | 初始密码，6-50 字符 |
| nickname | string | 否 | 昵称（默认使用用户名） |
| role_code | string | 否 | 指定角色，默认为 user |

**权限管控规则：**
- 不能为新用户分配等级高于或等于操作者自身的角色，否则返回 403
- 例：admin（level=10）只能分配 level > 10 的角色（如 user），不能分配 admin 或 super_admin

**错误响应：**
- 400：`用户名或邮箱已被注册` / `无效的角色代码`
- 403：`不能分配高于自身等级的角色`

**成功响应：**
```json
{
    "code": 0,
    "message": "success",
    "data": {
        "id": 6,
        "username": "created_by_admin",
        "email": "admin_created@example.com",
        "nickname": "管理员创建",
        "avatar": "",
        "gender": 0,
        "status": 1,
        "status_text": "正常",
        "roles": [
            { "code": "user", "name": "普通用户", "level": 100 }
        ],
        "max_level": 100,
        "created_at": "2026-03-02 11:12:33",
        "updated_at": "2026-03-02 11:12:33"
    },
    "trace_id": "...",
    "time": "2026-03-02 11:12:33"
}
```

---

## 7. 获取用户会议记录

`GET /api/v1/admin/users/:id/meetings`

**权限：** admin

**查询参数：**

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| status | string | 无 | ongoing=进行中，upcoming=即将开始，ended=已结束 |
| page | int | 1 | 页码 |
| page_size | int | 20 | 每页数量 |
