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
| GET | /api/v1/contacts/requests | 需认证 | 获取待处理的好友申请列表 |
| DELETE | /api/v1/contacts/:id | 需认证 | 删除好友 |
| PUT | /api/v1/contacts/:id/remark | 需认证 | 修改好友备注 |
| PUT | /api/v1/contacts/:id/group | 需认证 | 移动好友到分组 |
| POST | /api/v1/contacts/block | 需认证 | 拉黑用户 |
| DELETE | /api/v1/contacts/block/:id | 需认证 | 取消拉黑 |
| GET | /api/v1/contacts/block | 需认证 | 获取黑名单 |
| GET | /api/v1/contacts/groups | 需认证 | 获取好友分组列表 |
| POST | /api/v1/contacts/groups | 需认证 | 创建好友分组 |
| PUT | /api/v1/contacts/groups/:id | 需认证 | 修改好友分组 |
| DELETE | /api/v1/contacts/groups/:id | 需认证 | 删除好友分组 |
| GET | /api/v1/contacts/recommend | 需认证 | 好友推荐 |
| GET | /api/v1/users/search | 需认证 | 搜索用户 |

---

## 1. 获取好友列表

`GET /api/v1/contacts`

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| group_id | int | 否 | 按分组筛选，不传则返回全部 |

**成功响应：**
```json
{
    "code": 0,
    "message": "ok",
    "data": [
        {
            "user_id": 2,
            "username": "lisi",
            "nickname": "李四",
            "avatar": "",
            "remark": "我的同事",
            "group_id": 1,
            "is_online": true
        }
    ]
}
```

---

## 2. 发送好友申请

`POST /api/v1/contacts/request`

**请求参数：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| target_id | int | 是 | 目标用户 ID |
| message | string | 否 | 申请附言 |

**错误场景：**
- 400: 不能添加自己为好友
- 400: 已是好友
- 400: 已有待处理的申请
- 403: 对方已将你拉黑

---

## 3. 接受好友申请

`POST /api/v1/contacts/accept`

**请求参数：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| request_id | int | 是 | 好友申请记录 ID |

**说明：** 接受后系统自动创建双向好友关系，并通过 WebSocket 推送 `contact.request.accepted` 事件给对方。

---

## 4. 拒绝好友申请

`POST /api/v1/contacts/reject`

**请求参数：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| request_id | int | 是 | 好友申请记录 ID |

---

## 5. 获取待处理的好友申请

`GET /api/v1/contacts/requests`

**成功响应：**
```json
{
    "code": 0,
    "message": "ok",
    "data": [
        {
            "id": 5,
            "user_id": 3,
            "username": "wangwu",
            "nickname": "王五",
            "avatar": "",
            "message": "我是你的同学",
            "created_at": "2026-03-01T10:30:00Z"
        }
    ]
}
```

---

## 6. 删除好友

`DELETE /api/v1/contacts/:id`

**路径参数：** `id` — 好友的用户 ID

**说明：** 删除后双向关系均解除。

---

## 7. 修改好友备注

`PUT /api/v1/contacts/:id/remark`

**路径参数：** `id` — 好友的用户 ID

**请求参数：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| remark | string | 是 | 新备注名，最多 50 字符 |

---

## 8. 移动好友到分组

`PUT /api/v1/contacts/:id/group`

**路径参数：** `id` — 好友的用户 ID

**请求参数：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| group_id | int | 是 | 目标分组 ID，0 为默认分组 |

---

## 9. 拉黑用户

`POST /api/v1/contacts/block`

**请求参数：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| target_id | int | 是 | 目标用户 ID |

**说明：** 拉黑后自动解除好友关系（如果存在），对方无法向你发送好友申请和消息。

---

## 10. 取消拉黑

`DELETE /api/v1/contacts/block/:id`

**路径参数：** `id` — 被拉黑用户的 ID

---

## 11. 获取黑名单

`GET /api/v1/contacts/block`

**成功响应：**
```json
{
    "code": 0,
    "message": "ok",
    "data": [
        {
            "user_id": 5,
            "username": "blocked_user",
            "nickname": "某用户",
            "avatar": ""
        }
    ]
}
```

---

## 12. 获取好友分组列表

`GET /api/v1/contacts/groups`

**成功响应：**
```json
{
    "code": 0,
    "message": "ok",
    "data": [
        { "id": 1, "name": "同事", "sort_order": 0, "friend_count": 15 },
        { "id": 2, "name": "朋友", "sort_order": 1, "friend_count": 8 }
    ]
}
```

---

## 13. 创建好友分组

`POST /api/v1/contacts/groups`

**请求参数：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 分组名称，最多 50 字符 |

---

## 14. 修改好友分组

`PUT /api/v1/contacts/groups/:id`

**路径参数：** `id` — 分组 ID

**请求参数：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 新分组名称 |
| sort_order | int | 否 | 排序序号 |

---

## 15. 删除好友分组

`DELETE /api/v1/contacts/groups/:id`

**路径参数：** `id` — 分组 ID

**说明：** 删除分组后，该分组内的好友自动移至默认分组（group_id = 0）。

---

## 16. 好友推荐

`GET /api/v1/contacts/recommend`

**说明：** 基于共同好友算法推荐可能认识的人。

**成功响应：**
```json
{
    "code": 0,
    "message": "ok",
    "data": [
        {
            "user_id": 8,
            "username": "zhaoliu",
            "nickname": "赵六",
            "avatar": "",
            "common_count": 3
        }
    ]
}
```

---

## 17. 搜索用户

`GET /api/v1/users/search`

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| keyword | string | 是 | 搜索关键词（用户名/昵称模糊匹配） |
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页数量，默认 20 |

**成功响应：**
```json
{
    "code": 0,
    "message": "ok",
    "data": [
        {
            "user_id": 10,
            "username": "newuser",
            "nickname": "新用户",
            "avatar": "",
            "is_friend": false
        }
    ]
}
```
