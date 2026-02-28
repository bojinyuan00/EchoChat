# 联系人模块 API (Contact)

> 通用规范（认证方式、响应格式、错误码）见 [README.md](../README.md)

---

## 接口列表

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | /api/v1/contacts | 需认证 | 获取好友列表 |
| POST | /api/v1/contacts/request | 需认证 | 发送好友申请 |
| POST | /api/v1/contacts/accept | 需认证 | 接受好友申请 |
| POST | /api/v1/contacts/reject | 需认证 | 拒绝好友申请 |
| DELETE | /api/v1/contacts/:id | 需认证 | 删除好友 |
| PUT | /api/v1/contacts/:id/remark | 需认证 | 修改好友备注 |
| GET | /api/v1/contacts/groups | 需认证 | 获取好友分组列表 |
| POST | /api/v1/contacts/groups | 需认证 | 创建好友分组 |

---

## 1. 获取好友列表

`GET /api/v1/contacts`

**权限：** 需认证

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| group_id | int | 否 | 按分组筛选 |

**成功响应：**
```json
{
    "code": 0,
    "message": "ok",
    "data": [
        {
            "id": 1,
            "friend_id": 2,
            "username": "lisi",
            "nickname": "李四",
            "remark": "我的同事",
            "avatar": "https://cdn.echochat.com/avatar/2.jpg",
            "online": true,
            "group_id": 1,
            "group_name": "同事"
        }
    ]
}
```

---

## 2. 发送好友申请

`POST /api/v1/contacts/request`

**权限：** 需认证

**请求参数：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| target_id | int | 是 | 目标用户 ID |
| message | string | 否 | 申请附言，如"我是张三的同事" |

**可能的错误码：** 1004（用户不存在），1005（已是好友或已发送过申请）

---

## 3. 接受好友申请

`POST /api/v1/contacts/accept`

**权限：** 需认证

**请求参数：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| friendship_id | int | 是 | 好友关系记录 ID |

**说明：** 接受后系统自动创建双向好友关系，并发送通知给对方。

---

## 4. 拒绝好友申请

`POST /api/v1/contacts/reject`

**权限：** 需认证

**请求参数：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| friendship_id | int | 是 | 好友关系记录 ID |

---

## 5. 删除好友

`DELETE /api/v1/contacts/:id`

**权限：** 需认证

**路径参数：** `id` — 好友关系记录 ID

**说明：** 删除后双向关系均解除，关联的单聊会话不会删除（消息记录保留）。

---

## 6. 修改好友备注

`PUT /api/v1/contacts/:id/remark`

**权限：** 需认证

**路径参数：** `id` — 好友关系记录 ID

**请求参数：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| remark | string | 是 | 新备注名，最多 50 字符 |

---

## 7. 获取好友分组列表

`GET /api/v1/contacts/groups`

**权限：** 需认证

**成功响应：**
```json
{
    "code": 0,
    "message": "ok",
    "data": [
        { "id": 1, "name": "同事", "sort_order": 0, "count": 15 },
        { "id": 2, "name": "朋友", "sort_order": 1, "count": 8 }
    ]
}
```

---

## 8. 创建好友分组

`POST /api/v1/contacts/groups`

**权限：** 需认证

**请求参数：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 分组名称，最多 50 字符 |

**可能的错误码：** 1005（同名分组已存在）
