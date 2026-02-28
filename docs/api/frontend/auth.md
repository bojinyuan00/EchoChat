# 认证模块 API (Auth)

> 通用规范（认证方式、响应格式、错误码）见 [README.md](../README.md)

---

## 接口列表

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| POST | /api/v1/auth/register | 公开 | 用户注册 |
| POST | /api/v1/auth/login | 公开 | 用户登录 |
| POST | /api/v1/auth/logout | 需认证 | 退出登录 |
| POST | /api/v1/auth/refresh-token | 公开 | 刷新 Token |
| GET | /api/v1/auth/profile | 需认证 | 获取个人信息 |
| PUT | /api/v1/auth/profile | 需认证 | 更新个人信息 |
| PUT | /api/v1/auth/password | 需认证 | 修改密码 |

---

## 1. 用户注册

`POST /api/v1/auth/register`

**权限：** 公开

**请求参数：**

| 字段 | 类型 | 必填 | 规则 | 说明 |
|------|------|------|------|------|
| username | string | 是 | 3-50 字符 | 用户名，全局唯一 |
| email | string | 是 | 合法邮箱格式 | 邮箱地址，全局唯一 |
| password | string | 是 | 6-50 字符 | 登录密码 |
| nickname | string | 否 | 最多 50 字符 | 昵称，默认与用户名相同 |

**请求示例：**
```json
{
    "username": "zhangsan",
    "email": "zhangsan@example.com",
    "password": "123456",
    "nickname": "张三"
}
```

**成功响应（200 OK）：**
```json
{
    "code": 0,
    "message": "success",
    "data": {
        "token": "eyJhbGciOiJIUzI1NiIs...",
        "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
        "expires_in": 7200,
        "user": {
            "id": 1,
            "username": "zhangsan",
            "email": "zhangsan@example.com",
            "nickname": "张三",
            "avatar": "",
            "gender": 0,
            "roles": ["user"]
        }
    },
    "trace_id": "6478824e-2926-4d35-aa5f-047c8cfbb36b",
    "time": "2026-02-28 16:42:03"
}
```

> `expires_in` 为 Access Token 有效期（秒），当前配置为 7200 秒（2 小时）。Refresh Token 有效期为 7 天。

**可能的错误：**

| HTTP 状态码 | 说明 |
|------------|------|
| 400 | 参数校验失败 / 用户名或邮箱已被注册 |

---

## 2. 用户登录

`POST /api/v1/auth/login`

**权限：** 公开

**请求参数：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| account | string | 是 | 用户名或邮箱（自动识别） |
| password | string | 是 | 登录密码 |

**请求示例：**
```json
{
    "account": "zhangsan",
    "password": "123456"
}
```

**成功响应（200 OK）：**
```json
{
    "code": 0,
    "message": "success",
    "data": {
        "token": "eyJhbGciOiJIUzI1NiIs...",
        "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
        "expires_in": 7200,
        "user": {
            "id": 1,
            "username": "zhangsan",
            "email": "zhangsan@example.com",
            "nickname": "张三",
            "avatar": "",
            "gender": 0,
            "roles": ["user"]
        }
    },
    "trace_id": "fe290a90-00af-4c6b-8af7-ba6ae1d57d20",
    "time": "2026-02-28 16:42:04"
}
```

**可能的错误：**

| HTTP 状态码 | 说明 |
|------------|------|
| 400 | 参数校验失败 |
| 401 | 账号或密码错误 |
| 403 | 账号已被禁用 / 账号已注销 |
| 404 | 用户不存在 |

---

## 3. 退出登录

`POST /api/v1/auth/logout`

**权限：** 需认证

**说明：** 采用有状态 JWT 方案，Token 存储在 Redis 中（`echo:auth:token:{user_id}` 和 `echo:auth:refresh:{user_id}`）。登出时服务端会从 Redis 中删除该用户的 Access Token 和 Refresh Token，使其立即失效。客户端也应同步清除本地存储的 Token。

**成功响应：**
```json
{
    "code": 0,
    "message": "success",
    "data": null,
    "trace_id": "...",
    "time": "2026-02-28 16:42:05"
}
```

---

## 4. 刷新 Token

`POST /api/v1/auth/refresh-token`

**权限：** 公开（携带 refresh_token）

**请求参数：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| refresh_token | string | 是 | 刷新令牌 |

**请求示例：**
```json
{
    "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**成功响应：**
```json
{
    "code": 0,
    "message": "success",
    "data": {
        "token": "eyJhbG...(新 Access Token)",
        "refresh_token": "eyJhbG...(新 Refresh Token)",
        "expires_in": 7200,
        "user": {
            "id": 1,
            "username": "zhangsan",
            "email": "zhangsan@example.com",
            "nickname": "张三",
            "avatar": "",
            "gender": 0,
            "roles": ["user"]
        }
    },
    "trace_id": "...",
    "time": "2026-02-28 16:42:06"
}
```

**可能的错误：**

| HTTP 状态码 | 说明 |
|------------|------|
| 400 | 参数校验失败 / 无效的 Refresh Token 类型 |
| 401 | Token 已过期或无效 |
| 403 | 账号已被禁用 / 账号已注销 |
| 404 | 用户不存在 |

---

## 5. 获取个人信息

`GET /api/v1/auth/profile`

**权限：** 需认证

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
        "avatar": "https://cdn.echochat.com/avatar/1.jpg",
        "gender": 1,
        "phone": "13800138000",
        "roles": ["user"],
        "created_at": "2026-02-27 10:00:00"
    },
    "trace_id": "...",
    "time": "2026-02-28 16:42:07"
}
```

> profile 接口返回的 `UserInfo` 包含 `phone` 和 `created_at` 字段（相比登录响应中的用户信息更完整）。

---

## 6. 更新个人信息

`PUT /api/v1/auth/profile`

**权限：** 需认证

**请求参数（均为可选，只传需要修改的字段）：**

| 字段 | 类型 | 必填 | 规则 | 说明 |
|------|------|------|------|------|
| nickname | string | 否 | 最多 50 字符 | 新昵称 |
| avatar | string | 否 | 最多 500 字符 | 新头像 URL |
| gender | int | 否 | 0/1/2 | 性别：0=未知，1=男，2=女 |
| phone | string | 否 | 最多 20 字符 | 手机号 |

**成功响应：** 返回更新后的完整用户信息（与获取个人信息接口格式一致）。

---

## 7. 修改密码

`PUT /api/v1/auth/password`

**权限：** 需认证

**请求参数：**

| 字段 | 类型 | 必填 | 规则 | 说明 |
|------|------|------|------|------|
| old_password | string | 是 | - | 原密码 |
| new_password | string | 是 | 6-50 字符 | 新密码 |

**成功响应：**
```json
{
    "code": 0,
    "message": "success",
    "data": null,
    "trace_id": "...",
    "time": "2026-02-28 16:42:08"
}
```

**可能的错误：**

| HTTP 状态码 | 说明 |
|------------|------|
| 400 | 参数校验失败 |
| 401 | 原密码错误 |
| 404 | 用户不存在 |
