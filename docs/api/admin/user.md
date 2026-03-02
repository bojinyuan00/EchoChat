# 管理端 — 用户管理 API

> 通用规范（认证方式、响应格式、错误码）见 [README.md](../README.md)
> 以下所有接口均需要 **JWT 认证 + admin/super_admin 角色**

---

## 接口列表

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | /api/v1/admin/users | admin | 获取用户列表 |
| GET | /api/v1/admin/users/:id | admin | 获取用户详情 |
| PUT | /api/v1/admin/users/:id/status | admin | 更新用户状态 |
| PUT | /api/v1/admin/users/:id/role | super_admin | 分配用户角色 |
| POST | /api/v1/admin/users | admin | 管理员创建用户 |
| GET | /api/v1/admin/users/:id/meetings | admin | 获取用户会议记录 |

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
                "roles": ["user"],
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
        "roles": ["user"],
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

**说明：** 禁用用户后，该用户的所有活跃 Token 将被清除，正在进行的 WebSocket 连接将被断开。

---

## 4. 分配用户角色

`PUT /api/v1/admin/users/:id/role`

**权限：** admin / super_admin

**请求参数：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| role_code | string | 是 | 角色代码：user / admin / super_admin |

---

## 5. 管理员创建用户

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

---

## 6. 获取用户会议记录

`GET /api/v1/admin/users/:id/meetings`

**权限：** admin

**查询参数：**

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| status | string | 无 | ongoing=进行中，upcoming=即将开始，ended=已结束 |
| page | int | 1 | 页码 |
| page_size | int | 20 | 每页数量 |
