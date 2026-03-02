# 管理端好友关系管理 API

> 通用规范（认证方式、响应格式、错误码）见 [README.md](../README.md)
> 所有接口需要 JWT + admin/super_admin 角色权限

---

## 接口列表

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | /api/v1/admin/contacts | admin | 获取所有好友关系列表（分页） |
| DELETE | /api/v1/admin/contacts/:id | admin | 删除好友关系（双向解除） |

---

## 1. 获取所有好友关系列表

`GET /api/v1/admin/contacts`

**说明：** 分页查询系统中所有好友关系记录，包含双方用户名信息。

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页数量，默认 20 |

**成功响应：**
```json
{
    "code": 0,
    "message": "ok",
    "data": {
        "total": 150,
        "list": [
            {
                "id": 1,
                "user_id": 1,
                "username": "zhangsan",
                "friend_id": 2,
                "friend_username": "lisi",
                "remark": "同事",
                "status": 1,
                "created_at": "2026-03-01 10:00:00"
            }
        ]
    }
}
```

**status 状态码：**

| 值 | 说明 |
|-----|------|
| 0 | 待确认（pending） |
| 1 | 已通过（accepted） |
| 2 | 已拒绝（rejected） |
| 3 | 已拉黑（blocked） |

---

## 2. 删除好友关系

`DELETE /api/v1/admin/contacts/:id`

**路径参数：** `id` — 好友关系记录 ID

**说明：** 管理员可以强制删除任意好友关系。此操作会双向解除（同时删除 A→B 和 B→A 的关系记录）。

**成功响应：**
```json
{
    "code": 0,
    "message": "ok"
}
```
