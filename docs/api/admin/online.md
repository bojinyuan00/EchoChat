# 管理端在线监控 API

> 通用规范（认证方式、响应格式、错误码）见 [README.md](../README.md)
> 所有接口需要 JWT + admin/super_admin 角色权限

---

## 接口列表

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | /api/v1/admin/online/users | admin | 获取在线用户列表 |
| GET | /api/v1/admin/online/count | admin | 获取在线用户数量 |

---

## 1. 获取在线用户列表

`GET /api/v1/admin/online/users`

**说明：** 返回当前所有在线用户的基本信息。数据来源于 Redis SET `echo:user:online`。

**成功响应：**
```json
{
    "code": 0,
    "message": "ok",
    "data": [
        {
            "user_id": 1,
            "username": "super_admin"
        },
        {
            "user_id": 2,
            "username": "lisi"
        }
    ]
}
```

---

## 2. 获取在线用户数量

`GET /api/v1/admin/online/count`

**成功响应：**
```json
{
    "code": 0,
    "message": "ok",
    "data": {
        "count": 25
    }
}
```
