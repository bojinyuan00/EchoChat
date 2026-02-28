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
| username | string | 是 | 3-50 字符，字母数字下划线 | 用户名 |
| email | string | 是 | 合法邮箱格式 | 邮箱地址 |
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

**成功响应：**
```json
{
    "code": 0,
    "message": "ok",
    "data": {
        "token": "eyJhbGciOiJIUzI1NiIs...",
        "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
        "expires_in": 604800,
        "user": {
            "id": 1,
            "username": "zhangsan",
            "email": "zhangsan@example.com",
            "nickname": "张三",
            "avatar": "",
            "roles": ["user"]
        }
    }
}
```

**可能的错误码：** 1001, 2001, 2002

---

## 2. 用户登录

`POST /api/v1/auth/login`

**权限：** 公开

**请求参数：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| account | string | 是 | 用户名或邮箱（自动识别格式） |
| password | string | 是 | 登录密码 |

**请求示例：**
```json
{
    "account": "zhangsan",
    "password": "123456"
}
```

**成功响应：** 与注册接口返回格式一致

**可能的错误码：** 1001, 2003, 2004

---

## 3. 退出登录

`POST /api/v1/auth/logout`

**权限：** 需认证

**说明：** 服务端清除 Redis 中的 Token 记录，客户端需同步清除本地存储的 Token。

**成功响应：**
```json
{
    "code": 0,
    "message": "ok",
    "data": null
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

**成功响应：**
```json
{
    "code": 0,
    "message": "ok",
    "data": {
        "token": "eyJhbG...(新 Access Token)",
        "refresh_token": "eyJhbG...(新 Refresh Token)",
        "expires_in": 604800
    }
}
```

**可能的错误码：** 1002

---

## 5. 获取个人信息

`GET /api/v1/auth/profile`

**权限：** 需认证

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
        "avatar": "https://cdn.echochat.com/avatar/1.jpg",
        "gender": 1,
        "phone": "13800138000",
        "roles": ["user"],
        "created_at": "2026-02-27 10:00:00"
    }
}
```

---

## 6. 更新个人信息

`PUT /api/v1/auth/profile`

**权限：** 需认证

**请求参数（均为可选，只传需要修改的字段）：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| nickname | string | 否 | 新昵称 |
| avatar | string | 否 | 新头像 URL |
| gender | int | 否 | 性别：0=未知，1=男，2=女 |
| phone | string | 否 | 手机号 |

---

## 7. 修改密码

`PUT /api/v1/auth/password`

**权限：** 需认证

**请求参数：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| old_password | string | 是 | 原密码 |
| new_password | string | 是 | 新密码（6-50 字符） |

**可能的错误码：** 1001, 2003（原密码错误）
