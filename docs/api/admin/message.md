# 管理端消息管理 API

> **最后更新**：2026-03-03
> **阶段**：Phase 2d

---

## 接口列表

| 方法 | 路径 | 描述 | 认证 |
|------|------|------|------|
| GET | /api/v1/admin/messages | 消息列表（分页+多条件筛选） | JWT + admin |
| GET | /api/v1/admin/messages/:id | 消息详情 | JWT + admin |
| DELETE | /api/v1/admin/messages/:id | 删除消息（软删除） | JWT + admin |
| PUT | /api/v1/admin/messages/:id/recall | 撤回消息 | JWT + admin |
| GET | /api/v1/admin/messages/stats | 消息统计 | JWT + admin |

---

## GET /api/v1/admin/messages

获取消息列表（分页+多条件筛选）。

### 请求参数（Query）

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码（默认 1） |
| page_size | int | 否 | 每页条数（默认 20，最大 100） |
| keyword | string | 否 | 搜索关键词（模糊匹配消息内容） |
| type | int | 否 | 消息类型：1=文本 / 2=图片 / 3=语音 / 5=文件 / 10=系统 |
| sender_id | int64 | 否 | 发送者 ID |
| conversation_id | int64 | 否 | 会话 ID |
| status | int | 否 | 消息状态：1=正常 / 2=已撤回 / 3=已删除 |
| start_time | string | 否 | 开始日期（YYYY-MM-DD） |
| end_time | string | 否 | 结束日期（YYYY-MM-DD） |

### 响应

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total": 156,
    "list": [
      {
        "id": 1,
        "conversation_id": 5,
        "sender_id": 10,
        "sender_nickname": "张三",
        "sender_avatar": "https://...",
        "type": 1,
        "type_label": "文本",
        "content": "你好",
        "extra": null,
        "status": 1,
        "status_label": "正常",
        "created_at": "2026-03-03 14:30:00"
      }
    ],
    "page": 1,
    "page_size": 20
  }
}
```

---

## GET /api/v1/admin/messages/:id

获取消息详情。

### 响应

同列表中的单条消息对象。

---

## DELETE /api/v1/admin/messages/:id

删除消息（软删除，状态改为 3=已删除）。

### 响应

```json
{
  "code": 0,
  "message": "success",
  "data": null
}
```

---

## PUT /api/v1/admin/messages/:id/recall

管理员撤回消息。撤回后会通过 WebSocket 推送 `im.message.recalled` 事件给该会话的所有在线成员。

### 响应

```json
{
  "code": 0,
  "message": "success",
  "data": null
}
```

### WS 推送事件

```json
{
  "event": "im.message.recalled",
  "data": {
    "message_id": 123,
    "conversation_id": 5,
    "operator_id": 0,
    "sender_id": 10,
    "recall_text": "管理员撤回了一条消息"
  }
}
```

---

## GET /api/v1/admin/messages/stats

获取消息统计数据。

### 请求参数（Query）

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| days | int | 否 | 统计天数（默认 7，最大 90） |

### 响应

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total_count": 5680,
    "today_count": 120,
    "type_distribution": [
      { "type": 1, "label": "文本", "count": 4500 },
      { "type": 2, "label": "图片", "count": 800 }
    ],
    "daily_trend": [
      { "date": "2026-02-25", "count": 150 },
      { "date": "2026-02-26", "count": 200 }
    ],
    "active_users": [
      { "user_id": 10, "nickname": "张三", "count": 500 }
    ],
    "active_groups": [
      { "group_id": 1, "name": "技术交流群", "count": 300 }
    ]
  }
}
```
