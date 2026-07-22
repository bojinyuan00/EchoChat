# 管理端群聊管理 REST API

> 当前实现基线：2026-07-22，根据 `backend/go-service/app/admin/router.go` 与 `group_manage_controller.go` 复核。
> 通用认证、响应包络和错误码见 [API README](../README.md)。全部接口要求管理员 JWT。

## 路径总览（3 个）

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/v1/admin/groups?page=1&page_size=20&keyword=` | 分页查询群聊，可按名称模糊搜索 |
| GET | `/api/v1/admin/groups/:id` | 获取群详情及成员列表 |
| DELETE | `/api/v1/admin/groups/:id` | 管理员解散群聊 |

## 1. 群聊列表

`GET /api/v1/admin/groups`

查询参数：`page` 默认 1，`page_size` 默认 20 且范围 1–100，`keyword` 可选。响应 `data` 包含 `list` 与 `total`。列表项字段为：

```json
{
  "id": 12,
  "conversation_id": 30,
  "name": "EchoChat 讨论组",
  "avatar": "",
  "owner_id": 1,
  "owner_name": "Alice",
  "member_count": 8,
  "max_members": 500,
  "status": 1,
  "is_all_muted": false,
  "created_at": "2026-07-22 12:00:00"
}
```

## 2. 群聊详情

`GET /api/v1/admin/groups/:id`

在列表项字段之外增加 `notice`、`is_searchable` 和 `members`。成员字段为 `user_id`、`username`、`nickname`、`role`、`is_muted`、`joined_at`。

## 3. 解散群聊

`DELETE /api/v1/admin/groups/:id`

成功时 `data` 为 `null`。`:id` 不是整数时返回参数错误；服务层失败使用统一错误响应。
