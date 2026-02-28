# 即时通讯模块 API (IM)

> 通用规范（认证方式、响应格式、错误码）见 [README.md](README.md)
> 消息的实时收发通过 WebSocket 完成，见 [websocket.md](websocket.md)
> 本文档中的接口用于会话管理和消息历史查询等非实时操作。

---

## 接口列表

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | /api/v1/conversations | 需认证 | 获取会话列表 |
| POST | /api/v1/conversations | 需认证 | 创建群聊 |
| GET | /api/v1/conversations/:id | 需认证 | 获取会话详情 |
| GET | /api/v1/conversations/:id/messages | 需认证 | 获取消息历史 |
| POST | /api/v1/conversations/:id/members | 需认证 | 邀请成员加入群聊 |
| DELETE | /api/v1/conversations/:id/members/:uid | 需认证 | 移除群聊成员 |

---

## 1. 获取会话列表

`GET /api/v1/conversations`

**权限：** 需认证

**说明：** 返回当前用户的所有会话，按最后消息时间倒序排列。单聊会话的 `name`/`avatar` 为空，前端应使用 `target_user` 的信息展示。

**成功响应：**
```json
{
    "code": 0,
    "message": "ok",
    "data": [
        {
            "id": 1,
            "type": 1,
            "name": "",
            "avatar": "",
            "target_user": {
                "id": 2,
                "nickname": "李四",
                "avatar": "https://cdn.echochat.com/avatar/2.jpg",
                "online": true
            },
            "last_message": {
                "id": 100,
                "type": 1,
                "content": "你好",
                "sender_id": 2,
                "created_at": "2026-02-27 10:30:00"
            },
            "unread_count": 3,
            "is_pinned": false
        },
        {
            "id": 5,
            "type": 2,
            "name": "产品讨论组",
            "avatar": "https://cdn.echochat.com/group/5.jpg",
            "target_user": null,
            "last_message": {
                "id": 205,
                "type": 1,
                "content": "明天开会",
                "sender_id": 3,
                "created_at": "2026-02-27 11:00:00"
            },
            "unread_count": 0,
            "is_pinned": true,
            "member_count": 8
        }
    ]
}
```

---

## 2. 创建群聊

`POST /api/v1/conversations`

**权限：** 需认证

**请求参数：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 群聊名称 |
| member_ids | int[] | 是 | 初始成员用户 ID 列表（不含自己，至少 2 人） |

**说明：** 创建者自动成为群主（role=2），被邀请的成员为普通成员（role=0）。

---

## 3. 获取会话详情

`GET /api/v1/conversations/:id`

**权限：** 需认证，且为该会话成员

**成功响应（群聊示例）：**
```json
{
    "code": 0,
    "message": "ok",
    "data": {
        "id": 5,
        "type": 2,
        "name": "产品讨论组",
        "avatar": "https://cdn.echochat.com/group/5.jpg",
        "owner_id": 1,
        "max_members": 200,
        "member_count": 8,
        "members": [
            { "user_id": 1, "nickname": "张三", "role": 2, "online": true },
            { "user_id": 2, "nickname": "李四", "role": 0, "online": false }
        ],
        "created_at": "2026-02-20 09:00:00"
    }
}
```

---

## 4. 获取消息历史

`GET /api/v1/conversations/:id/messages`

**权限：** 需认证，且为该会话成员

**查询参数：**

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| before_id | int | 无 | 获取此消息 ID 之前的消息（用于向上翻页加载历史） |
| limit | int | 30 | 每次获取数量，最大 50 |

**成功响应：**
```json
{
    "code": 0,
    "message": "ok",
    "data": {
        "messages": [
            {
                "id": 98,
                "sender_id": 1,
                "sender_name": "张三",
                "sender_avatar": "https://...",
                "type": 1,
                "content": "明天几点开会？",
                "extra": {},
                "status": 1,
                "created_at": "2026-02-27 10:28:00"
            },
            {
                "id": 99,
                "sender_id": 2,
                "sender_name": "李四",
                "sender_avatar": "https://...",
                "type": 2,
                "content": "",
                "extra": {
                    "url": "https://cdn.echochat.com/img/xxx.jpg",
                    "width": 800,
                    "height": 600,
                    "thumbnail": "https://cdn.echochat.com/img/xxx_thumb.jpg"
                },
                "status": 1,
                "created_at": "2026-02-27 10:29:00"
            }
        ],
        "has_more": true
    }
}
```

---

## 5. 邀请成员加入群聊

`POST /api/v1/conversations/:id/members`

**权限：** 需认证，且为该群聊成员

**请求参数：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| user_ids | int[] | 是 | 要邀请的用户 ID 列表 |

**可能的错误码：** 3001（会话不存在），3002（非会话成员），4003（超出人数上限）

---

## 6. 移除群聊成员

`DELETE /api/v1/conversations/:id/members/:uid`

**权限：** 需认证，且为群主或管理员

**路径参数：**
- `id` — 会话 ID
- `uid` — 被移除的用户 ID

**可能的错误码：** 3001, 3002, 1003（非群主/管理员无权操作）
