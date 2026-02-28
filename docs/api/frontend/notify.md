# 通知模块 API (Notify)

> 通用规范（认证方式、响应格式、错误码）见 [README.md](../README.md)
> 新通知的实时推送通过 WebSocket 完成，见 [websocket.md](../websocket.md)

---

## 接口列表

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | /api/v1/notifications | 需认证 | 获取通知列表 |
| PUT | /api/v1/notifications/:id/read | 需认证 | 标记通知已读 |
| PUT | /api/v1/notifications/read-all | 需认证 | 全部标记已读 |

---

## 1. 获取通知列表

`GET /api/v1/notifications`

**权限：** 需认证

**查询参数：**

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| is_read | bool | 无 | 筛选已读/未读，不传则返回全部 |
| type | string | 无 | 筛选通知类型（meeting_invite / friend_request / friend_accepted / meeting_reminder / system） |
| page | int | 1 | 页码 |
| page_size | int | 20 | 每页数量 |

**成功响应：**
```json
{
    "code": 0,
    "message": "ok",
    "data": {
        "list": [
            {
                "id": 1,
                "type": "meeting_invite",
                "title": "会议邀请",
                "content": "张三邀请你参加会议「产品需求讨论」",
                "extra": {
                    "room_code": "123-456-789",
                    "room_title": "产品需求讨论",
                    "from_user_id": 1,
                    "from_username": "zhangsan"
                },
                "is_read": false,
                "created_at": "2026-02-27 10:00:00"
            },
            {
                "id": 2,
                "type": "friend_request",
                "title": "好友申请",
                "content": "李四请求添加你为好友",
                "extra": {
                    "from_user_id": 2,
                    "from_username": "lisi",
                    "message": "我是你的同事"
                },
                "is_read": false,
                "created_at": "2026-02-27 09:30:00"
            }
        ],
        "total": 15,
        "page": 1,
        "page_size": 20
    }
}
```

---

## 2. 标记通知已读

`PUT /api/v1/notifications/:id/read`

**权限：** 需认证

**路径参数：** `id` — 通知 ID

---

## 3. 全部标记已读

`PUT /api/v1/notifications/read-all`

**权限：** 需认证

**说明：** 将当前用户的所有未读通知标记为已读。
