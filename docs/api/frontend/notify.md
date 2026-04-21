# 通知模块 API (Notify)

> 通用规范（认证方式、响应格式、错误码）见 [README.md](../README.md)  
> 新通知的实时推送通过 WebSocket 完成，见 [websocket.md](../websocket.md)  
> 所属阶段：Phase 2e-1（已完成）

---

## 概览

统一通知中心，覆盖四大业务类型：

| 分类 | 类型常量 | 触发模块 |
|---|---|---|
| 好友 | `friend_request` / `friend_accepted` / `friend_rejected` | contact |
| 群聊 | `group_invite` / `group_join_request` / `group_join_approved` / `group_join_rejected` / `group_kicked` / `group_role_changed` | group |
| 系统 | `system_broadcast` | admin 广播 |
| 会议 | `meeting_invite` / `meeting_reminder` | meeting（Phase 2e-2/2e-3 接入） |

> 通知持久化在 `notify_notifications` 表，每条通知对应单个接收者；30 天前的已读通知由后台定时任务自动清理，未读通知永久保留。

---

## 接口列表

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | /api/v1/notifications | 需认证 | 获取通知列表（游标分页） |
| GET | /api/v1/notifications/unread-count | 需认证 | 获取未读数统计 |
| PUT | /api/v1/notifications/:id/read | 需认证 | 标记单条已读 |
| PUT | /api/v1/notifications/read-all | 需认证 | 全部/按分类标记已读 |
| POST | /api/v1/admin/notifications/broadcast | 管理员 | 全员广播系统通知 |

---

## 1. 获取通知列表

`GET /api/v1/notifications`

**权限：** 需认证

**查询参数：**

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| category | string | `all` | 分类过滤：`all` / `friend` / `group` / `meeting` / `system` |
| is_read | bool | 无 | 是否已读：`true`/`false`，不传代表全部 |
| before_id | int64 | 无 | 游标：返回 id 小于此值的记录（按 id 降序） |
| limit | int | 20 | 页大小（最大 100） |

**成功响应：**
```json
{
    "code": 0,
    "message": "success",
    "data": {
        "list": [
            {
                "id": 128,
                "type": "group_invite",
                "category": "group",
                "title": "",
                "content": "张三 邀请你加入「产品交流群」",
                "extra": "{\"group_id\":12,\"group_name\":\"产品交流群\",\"conversation_id\":45,\"inviter_id\":3,\"inviter_name\":\"张三\"}",
                "actor_id": 3,
                "actor_name": "张三",
                "actor_avatar": "",
                "target_type": "group",
                "target_id": 12,
                "is_read": false,
                "created_at": "2026-04-20 10:15:00"
            }
        ],
        "has_more": true
    }
}
```

> `extra` 是 JSON 字符串，前端按需 `JSON.parse` 解析。游标翻页使用最后一条 `id` 作为下一次请求的 `before_id`。

---

## 2. 获取未读数统计

`GET /api/v1/notifications/unread-count`

**权限：** 需认证

**成功响应：**
```json
{
    "code": 0,
    "message": "success",
    "data": {
        "total": 5,
        "by_category": {
            "friend": 1,
            "group": 3,
            "meeting": 0,
            "system": 1
        }
    }
}
```

---

## 3. 标记单条已读

`PUT /api/v1/notifications/:id/read`

**权限：** 需认证（只能标记属于自己的通知）

**响应：**
```json
{ "code": 0, "message": "success", "data": { "affected": 1 } }
```

> 已读幂等：重复调用返回 `affected=0`。

---

## 4. 批量标记已读

`PUT /api/v1/notifications/read-all?category={category}`

**权限：** 需认证

**Query 参数：**

| 参数 | 类型 | 必填 | 说明 |
|---|---|---|---|
| category | string | 否 | 分类：`friend`/`group`/`meeting`/`system`，不传代表全部 |

> ⚠️ 注意：`category` 通过 **Query String** 传递（非 Request Body）。`PUT` 方法无需请求体。

**示例：**
- 全部标已读：`PUT /api/v1/notifications/read-all`
- 仅好友类标已读：`PUT /api/v1/notifications/read-all?category=friend`

**响应：**
```json
{ "code": 0, "message": "success", "data": { "affected": 3 } }
```

---

## 5. 管理员广播系统通知

`POST /api/v1/admin/notifications/broadcast`

**权限：** `RoleAdmin` 或 `RoleSuperAdmin`（JWT + 角色校验）

**请求体：**
```json
{
    "title": "系统维护通知",
    "content": "今晚 02:00 - 04:00 将进行系统升级，请提前保存工作",
    "target_type": "system",
    "target_id": null,
    "extra": null
}
```

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| title | string | 是 | 标题，最长 100 |
| content | string | 是 | 正文，最长 500 |
| target_type | string | 否 | 业务对象类型，默认 `system` |
| target_id | int64 | 否 | 业务对象 ID |
| extra | string | 否 | 扩展 JSON 字符串 |

**响应：**
```json
{ "code": 0, "message": "success", "data": { "affected": 128 } }
```

> 后端将遍历所有活跃用户批量写入通知，在线用户实时收到 `notify.new` WS 推送；离线用户重连后通过 `notify.unread.total` 触发未读数补偿。

---

## 6. WebSocket 事件

### `notify.new` — 新通知到达

| 方向 | 触发 |
|---|---|
| Server → Client | 每次通知写入成功后 |

**载荷**：与 REST 返回的单条通知对象字段一致。

```json
{
  "event": "notify.new",
  "data": {
    "id": 129,
    "type": "friend_request",
    "category": "friend",
    "title": "好友申请",
    "content": "Bob 请求添加你为好友",
    "actor_id": 5,
    "actor_name": "Bob",
    "target_type": "user",
    "target_id": 5,
    "is_read": false,
    "created_at": "2026-04-20 10:17:21"
  }
}
```

### `notify.unread.total` — 未读数补偿

| 方向 | 触发 |
|---|---|
| Server → Client | WebSocket 连接建立成功时 |

**载荷**：

```json
{
  "event": "notify.unread.total",
  "data": {
    "total": 5,
    "by_category": { "friend": 1, "group": 3, "meeting": 0, "system": 1 }
  }
}
```

> 前端以该值覆盖本地缓存的未读数，保证断线重连后铃铛徽标准确。

---

## 已知限制（单端连接架构）

- 当前 `ws.Hub` 仅支持同一用户单连接（新连接关闭旧连接），**不提供多端已读同步**；此限制已在 `docs/plans/2026-04-20-phase2e-design.md` §3.1/§3.5 修订。
- 管理端广播发布 UI 推迟到 Phase 2f（后端接口已落地，管理端页面未提供）。
- `group_invite` 的"接受/拒绝"内联操作当前语义：接受仅跳转会话；拒绝调用 `leaveGroup`（由于当前 InviteMembers 为直接加入成员）。
