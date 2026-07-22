# 群聊管理 REST API

> 当前实现基线：2026-07-22，根据 `backend/go-service/app/group/router.go` 与 `app/dto/group_dto.go` 复核。
> 通用认证、响应包络和错误码见 [API README](../README.md)。全部接口均需 JWT。

## 路径总览（19 个）

### 群聊管理（5 个）

| 方法 | 路径 | 说明 | 请求参数 |
|---|---|---|---|
| POST | `/api/v1/groups` | 创建群聊 | JSON：`name`（必填，最长 100）、`member_ids`（必填）、`avatar` |
| GET | `/api/v1/groups/search` | 搜索可发现群聊 | Query：`keyword`（必填）、`page`、`page_size` |
| GET | `/api/v1/groups/:id` | 获取群详情 | Path：`id` |
| PUT | `/api/v1/groups/:id` | 更新群资料 | JSON：可选 `name`、`avatar`、`notice`、`is_searchable` |
| DELETE | `/api/v1/groups/:id` | 解散群聊 | 群主权限 |

### 成员管理（7 个）

| 方法 | 路径 | 说明 | 请求参数/权限 |
|---|---|---|---|
| GET | `/api/v1/groups/:id/members` | 获取成员列表 | 群成员权限 |
| POST | `/api/v1/groups/:id/members` | 发出成员邀请 | JSON：`user_ids`；群主或管理员权限 |
| DELETE | `/api/v1/groups/:id/members/me` | 主动退群 | 群主需先转让群主 |
| DELETE | `/api/v1/groups/:id/members/:uid` | 移除成员 | 群主或管理员权限 |
| PUT | `/api/v1/groups/:id/members/me/nickname` | 修改本人群昵称 | JSON：`nickname`（最长 50） |
| PUT | `/api/v1/groups/:id/members/:uid/role` | 设置/取消管理员 | JSON：`role`，`0` 普通成员、`1` 管理员；群主权限 |
| PUT | `/api/v1/groups/:id/members/:uid/mute` | 禁言/解除成员禁言 | JSON：`is_muted`；群主或管理员权限 |

### 群主与全体禁言（2 个）

| 方法 | 路径 | 说明 | 请求参数/权限 |
|---|---|---|---|
| PUT | `/api/v1/groups/:id/transfer` | 转让群主 | JSON：`new_owner_id`；群主权限 |
| PUT | `/api/v1/groups/:id/all-mute` | 开启/关闭全体禁言 | JSON：`is_all_muted`；群主或管理员权限 |

### 入群申请（3 个）

| 方法 | 路径 | 说明 | 请求参数/权限 |
|---|---|---|---|
| POST | `/api/v1/groups/:id/join-requests` | 提交入群申请 | JSON：`message` |
| GET | `/api/v1/groups/:id/join-requests` | 获取待审批申请 | 群主或管理员权限 |
| PUT | `/api/v1/groups/:id/join-requests/:rid` | 审批申请 | JSON：`action`，仅 `approve` 或 `reject` |

### 邀请处理（2 个）

| 方法 | 路径 | 说明 | 权限 |
|---|---|---|---|
| POST | `/api/v1/groups/invitations/:rid/accept` | 接受待处理邀请并正式入群 | 仅被邀请者本人 |
| POST | `/api/v1/groups/invitations/:rid/reject` | 拒绝待处理邀请 | 仅被邀请者本人 |

## 主要响应对象

群详情使用以下字段：

```json
{
  "id": 12,
  "conversation_id": 30,
  "name": "EchoChat 讨论组",
  "avatar": "https://example.invalid/group.png",
  "owner_id": 1,
  "notice": "群公告",
  "max_members": 500,
  "member_count": 8,
  "is_searchable": true,
  "is_all_muted": false,
  "status": 1,
  "created_at": "2026-07-22 12:00:00"
}
```

成员对象字段为 `user_id`、`nickname`、`user_nickname`、`avatar`、`role`、`is_muted`、`joined_at`。入群申请对象字段为 `id`、`group_id`、`user_id`、`user_nickname`、`user_avatar`、`message`、`status`、`created_at`。

## 状态与语义

- 成员角色：`0` 普通成员、`1` 管理员、`2` 群主。
- 群状态：`1` 正常、`2` 已解散。
- 入群申请状态：`0` 待处理、`1` 已通过、`2` 已拒绝。
- 邀请成员不会立即入群；被邀请者调用 accept 后才写入成员关系。
