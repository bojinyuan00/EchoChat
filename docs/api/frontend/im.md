# 即时通讯模块 REST API (IM)

> 通用规范（认证方式、响应格式、错误码）见 [README.md](../README.md)
> 消息的实时收发（发送/撤回/标记已读/正在输入）通过 WebSocket 完成，见 [websocket.md](../websocket.md)
> 本文档中的接口用于会话管理和消息历史查询等非实时操作。
> **最后更新：** 2026-03-03（代码审查修复后同步）

---

## 接口列表

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | /api/v1/im/conversations | 需认证 | 获取会话列表 |
| GET | /api/v1/im/messages | 需认证 | 获取历史消息（游标分页） |
| PUT | /api/v1/im/conversations/:id/pin | 需认证 | 置顶/取消置顶 |
| DELETE | /api/v1/im/conversations/:id | 需认证 | 删除会话（软删除） |
| DELETE | /api/v1/im/conversations/:id/messages | 需认证 | 清空聊天记录（个人视图） |
| GET | /api/v1/im/messages/search | 需认证 | 全局消息搜索 |
| GET | /api/v1/im/unread | 需认证 | 获取全局未读消息总数 |

---

## 1. 获取会话列表

`GET /api/v1/im/conversations`

**权限：** 需认证

**说明：** 返回当前用户的所有会话，排序：置顶优先 → 最后消息时间降序。已软删除的会话不返回。通过 LEFT JOIN 一次获取单聊对方用户 ID，避免 N+1 查询。

**成功响应：**

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "list": [
            {
                "id": 1,
                "type": 1,
                "peer_user_id": 2,
                "peer_nickname": "李四",
                "peer_avatar": "https://...",
                "last_msg_content": "你好",
                "last_msg_time": "2026-03-03 10:30:00",
                "last_msg_sender_id": 2,
                "is_pinned": false,
                "unread_count": 3
            }
        ]
    }
}
```

---

## 2. 获取历史消息

`GET /api/v1/im/messages`

**权限：** 需认证，且为该会话成员

**查询参数：**

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| conversation_id | int | 是 | - | 会话 ID |
| before_id | int | 否 | 0 | 游标：查询 ID 小于此值的消息，0=最新 |
| limit | int | 否 | 30 | 每次获取数量，最大 100 |

**说明：** 支持 `clear_before_msg_id` 个人视图过滤（清空聊天记录后，仅过滤当前用户视图，不影响对方）。

**成功响应：**

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "list": [
            {
                "id": 99,
                "conversation_id": 1,
                "sender_id": 2,
                "type": 1,
                "content": "你好",
                "status": 1,
                "client_msg_id": "",
                "created_at": "2026-03-03 10:29:00"
            }
        ],
        "has_more": true
    }
}
```

---

## 3. 置顶/取消置顶会话

`PUT /api/v1/im/conversations/:id/pin`

**权限：** 需认证，且为该会话成员

**请求参数：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| is_pinned | bool | 是 | true=置顶, false=取消 |

---

## 4. 删除会话

`DELETE /api/v1/im/conversations/:id`

**权限：** 需认证，且为该会话成员

**说明：** 软删除，仅影响当前用户视图，不影响对方。同时清零未读数并更新 Redis 全局未读。

---

## 5. 清空聊天记录

`DELETE /api/v1/im/conversations/:id/messages`

**权限：** 需认证，且为该会话成员

**说明：** 个人视图操作，不影响对方的消息。实现方式：记录清空时的最后消息 ID（`clear_before_msg_id`），后续查询历史消息时过滤。同时清零该会话未读数。

---

## 6. 全局消息搜索

`GET /api/v1/im/messages/search`

**权限：** 需认证

**查询参数：**

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| keyword | string | 是 | - | 搜索关键词 |
| limit | int | 否 | 50 | 返回条数上限 |

**说明：** 使用 PostgreSQL GIN 全文索引（`to_tsvector('simple', content) @@ plainto_tsquery('simple', ?)`），仅搜索用户所在会话的消息。

**成功响应：**

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "list": [
            {
                "id": 99,
                "conversation_id": 1,
                "sender_id": 2,
                "type": 1,
                "content": "你好世界",
                "status": 1,
                "created_at": "2026-03-03 10:29:00",
                "sender_nickname": "李四",
                "sender_avatar": "https://..."
            }
        ]
    }
}
```

---

## 7. 获取全局未读消息总数

`GET /api/v1/im/unread`

**权限：** 需认证

**说明：** 从 Redis STRING 读取全局未读总数，用于 TabBar badge 显示。

**成功响应：**

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "total_unread": 5
    }
}
```
