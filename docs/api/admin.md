# 后台管理模块 API (Admin)

> 通用规范（认证方式、响应格式、错误码）见 [README.md](README.md)
> 以下所有接口（除管理员登录外）均需要 **JWT 认证 + admin/super_admin 角色**

---

## 接口列表

### 认证

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| POST | /api/v1/admin/auth/login | 公开 | 管理员登录 |

### 用户管理

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | /api/v1/admin/users | admin | 获取用户列表 |
| GET | /api/v1/admin/users/:id | admin | 获取用户详情 |
| PUT | /api/v1/admin/users/:id/status | admin | 更新用户状态 |
| PUT | /api/v1/admin/users/:id/role | super_admin | 分配用户角色 |
| POST | /api/v1/admin/users | admin | 管理员创建用户 |
| GET | /api/v1/admin/users/:id/meetings | admin | 获取用户会议记录 |

### 会议管理

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | /api/v1/admin/meetings | admin | 获取会议列表 |
| GET | /api/v1/admin/meetings/:id | admin | 获取会议详情 |
| PUT | /api/v1/admin/meetings/:id/close | admin | 强制结束会议 |
| GET | /api/v1/admin/meetings/stats | admin | 获取会议统计 |

### 系统管理

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | /api/v1/admin/dashboard | admin | 获取仪表盘数据 |
| GET | /api/v1/admin/logs | admin | 获取操作日志 |
| GET | /api/v1/admin/system/config | super_admin | 获取系统配置 |
| PUT | /api/v1/admin/system/config | super_admin | 更新系统配置 |

---

## 认证

### 1. 管理员登录

`POST /api/v1/admin/auth/login`

**权限：** 公开

**请求参数：** 与用户登录相同

**说明：** 登录后会额外验证用户是否拥有 admin 或 super_admin 角色，如果没有对应角色则返回 1003（权限不足）。

---

## 用户管理

### 2. 获取用户列表

`GET /api/v1/admin/users`

**权限：** admin

**查询参数：**

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| keyword | string | 无 | 搜索关键词（匹配用户名/邮箱/昵称） |
| status | int | 无 | 按状态筛选：1=正常，2=禁用，3=注销 |
| role | string | 无 | 按角色筛选：user / admin / super_admin |
| page | int | 1 | 页码 |
| page_size | int | 20 | 每页数量 |

---

### 3. 获取用户详情

`GET /api/v1/admin/users/:id`

**权限：** admin

**成功响应：**
```json
{
    "code": 0,
    "message": "ok",
    "data": {
        "id": 1,
        "username": "zhangsan",
        "email": "zhangsan@example.com",
        "nickname": "张三",
        "avatar": "https://...",
        "gender": 1,
        "phone": "13800138000",
        "status": 1,
        "roles": ["user"],
        "last_login_at": "2026-02-27T10:00:00Z",
        "last_login_ip": "192.168.1.100",
        "created_at": "2026-02-20T08:00:00Z"
    }
}
```

---

### 4. 更新用户状态

`PUT /api/v1/admin/users/:id/status`

**权限：** admin

**请求参数：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| status | int | 是 | 目标状态：1=正常（启用），2=禁用 |

**说明：** 禁用用户后，该用户的所有活跃 Token 将被清除，正在进行的 WebSocket 连接将被断开。

---

### 5. 分配用户角色

`PUT /api/v1/admin/users/:id/role`

**权限：** super_admin

**请求参数：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| role_code | string | 是 | 角色代码：user / admin / super_admin |

---

### 6. 管理员创建用户

`POST /api/v1/admin/users`

**权限：** admin

**请求参数：** 同用户注册接口，额外支持：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| role_code | string | 否 | 指定角色，默认为 user |

---

### 7. 获取用户会议记录

`GET /api/v1/admin/users/:id/meetings`

**权限：** admin

**查询参数：**

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| status | string | 无 | ongoing=进行中，upcoming=即将开始，ended=已结束 |
| page | int | 1 | 页码 |
| page_size | int | 20 | 每页数量 |

---

## 会议管理

### 8. 获取会议列表

`GET /api/v1/admin/meetings`

**权限：** admin

**查询参数：** 支持按 status、type、keyword（会议标题）筛选，支持分页

---

### 9. 获取会议详情

`GET /api/v1/admin/meetings/:id`

**权限：** admin

---

### 10. 强制结束会议

`PUT /api/v1/admin/meetings/:id/close`

**权限：** admin

**说明：** 强制结束后，所有参会者将收到会议结束通知，所有媒体资源将被回收。操作将记录到管理日志。

---

### 11. 获取会议统计

`GET /api/v1/admin/meetings/stats`

**权限：** admin

**成功响应：**
```json
{
    "code": 0,
    "message": "ok",
    "data": {
        "total_meetings": 1500,
        "ongoing_meetings": 5,
        "today_meetings": 23,
        "total_participants": 8500,
        "avg_duration": 1800
    }
}
```

---

## 系统管理

### 12. 获取仪表盘数据

`GET /api/v1/admin/dashboard`

**权限：** admin

**成功响应：**
```json
{
    "code": 0,
    "message": "ok",
    "data": {
        "total_users": 1200,
        "online_users": 85,
        "today_new_users": 12,
        "ongoing_meetings": 5,
        "today_meetings": 23,
        "today_messages": 5600
    }
}
```

---

### 13. 获取操作日志

`GET /api/v1/admin/logs`

**权限：** admin

**查询参数：**

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| module | string | 无 | 按模块筛选 |
| action | string | 无 | 按操作类型筛选 |
| admin_id | int | 无 | 按操作管理员筛选 |
| page | int | 1 | 页码 |
| page_size | int | 20 | 每页数量 |

---

### 14. 获取系统配置

`GET /api/v1/admin/system/config`

**权限：** super_admin

---

### 15. 更新系统配置

`PUT /api/v1/admin/system/config`

**权限：** super_admin
