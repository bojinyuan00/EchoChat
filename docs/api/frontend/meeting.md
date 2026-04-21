# 会议模块 API (Meeting) — Phase 2e-2 MVP

> 通用规范（认证、响应包络、通用错误码）见 [README.md](../README.md)
> 会议内实时信令（Transport / Producer / Consumer / 控制事件）通过 WebSocket 完成，见 [websocket.md](../websocket.md)

**实施状态**：本文档对应 Phase 2e-2 Task 5 已落地的 12 个 REST 接口，统一前缀 `/api/v1/meeting`，全部需要 JWT 认证。Task 5 完成时间：2026-04-21。

**设计口径**：以 [`docs/plans/2026-04-21-phase2e-2-design.md`](../../plans/2026-04-21-phase2e-2-design.md) §6.2 为单一事实来源（SSOT）。

---

## 路径总览

| # | 方法 | 路径 | 业务 | 权限 |
|---|------|------|------|------|
| 1 | POST | `/api/v1/meeting/rooms` | 创建即时会议 | 已登录且当前不在其他活跃会议 |
| 2 | GET  | `/api/v1/meeting/rooms/mine` | 我发起/参与过的最近会议 | 已登录 |
| 3 | GET  | `/api/v1/meeting/rooms/:code` | 会议详情 + 成员列表 | 当前活跃参会者 |
| 4 | POST | `/api/v1/meeting/rooms/:code/join` | 加入会议 | 已登录 |
| 5 | POST | `/api/v1/meeting/rooms/:code/leave` | 离开会议 | 当前活跃参会者 |
| 6 | POST | `/api/v1/meeting/rooms/:code/end` | 结束会议 | host |
| 7 | POST | `/api/v1/meeting/rooms/:code/transfer-host` | 转让主持人 | host |
| 8 | POST | `/api/v1/meeting/rooms/:code/kick` | 移除成员 | host |
| 9 | POST | `/api/v1/meeting/rooms/:code/invite` | 邀请用户（走通知中心） | 当前活跃参会者 |
| 10 | POST | `/api/v1/meeting/invite-tokens/:token/redeem` | 兑换邀请链接 | 已登录 |
| 11 | POST | `/api/v1/meeting/rooms/:code/chats` | 发送会议内文字消息 | 当前活跃参会者 |
| 12 | GET  | `/api/v1/meeting/rooms/:code/chats` | 拉取会议历史消息（游标分页） | 当前活跃参会者 |

**路径参数约定**：`:code` 为用户可见的 9 位会议号 `XXX-XXX-XXX`；`:token` 为 32 位十六进制邀请令牌。

---

## 通用约定

### 响应包络

成功：

```json
{ "code": 0, "message": "success", "data": { ... }, "trace_id": "...", "time": "2026-04-21 16:30:00" }
```

失败：

```json
{ "code": 400, "message": "会议号或密码错误", "trace_id": "...", "time": "2026-04-21 16:30:00" }
```

其中 HTTP 状态码与 `code` 一致：`200 OK` / `201 Created` / `400 Bad Request` / `403 Forbidden` / `404 Not Found` / `500 Internal Server Error`。

### 领域错误码映射

所有领域错误以 `message` 中文字面量为准，由 controller 层 `handleError` 统一映射：

| HTTP | 领域错误 | 触发场景 |
|------|----------|---------|
| 404 | `会议不存在` | `:code` 无匹配记录 |
| 403 | `仅主持人可操作` | 非 host 调用 end / transfer-host / kick |
| 400 | `会议已结束` | 会议 status=2 时再次操作 |
| 400 | `会议已满员` | 活跃参会人数 ≥ `max_members` |
| 400 | `会议需要密码` | 房间设密但请求未携带 |
| 400 | `会议密码错误` | bcrypt 校验失败 |
| 400 | `密码尝试过多，请稍后再试` | 同 `(user, code)` 5 次内错，Redis 锁 10 分钟 |
| 400 | `你当前未在会议中` | 操作要求活跃参会者但调用方 left_at 非空 |
| 400 | `你已在会议中` | 已活跃时重复 join |
| 400 | `你当前已在其他会议中` | 用户已在其它活跃会议中，违反单点参会 |
| 400 | `邀请链接已失效` | redis key 过期或被兑换 |
| 400 | `会议号冲突，请重试` | 生成房间号 5 次重试仍冲突（理论无发生） |
| 400 | `不能踢自己` | kick 目标为自己 |
| 400 | `不能将主持人转让给自己` | transfer-host 目标为自己 |
| 400 | `目标用户不是当前活跃参会者` | transfer-host / kick 对象 left_at 非空 |
| 500 | `{fallbackMsg}` + 原 error | 未识别异常（DB 故障等） |

---

## 1. 创建即时会议

`POST /api/v1/meeting/rooms`

**请求体**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| title | string | 是 | 会议标题，1~200 字符 |
| password | string | 否 | 入会密码明文，4~20 字符；服务端 bcrypt 哈希存储 |
| max_members | int | 否 | 容量上限，2~8（MVP 硬上限 8，超过将被截断为 8） |

**响应 `201 Created`**

```json
{
  "code": 0,
  "data": {
    "room": {
      "id": 12,
      "room_code": "835-000-036",
      "title": "T5 验证会议",
      "host_id": 16,
      "type": 1,
      "has_password": false,
      "max_members": 4,
      "status": 1,
      "status_label": "进行中",
      "started_at": "2026-04-21 16:30:10",
      "settings": "{}",
      "created_at": "2026-04-21 16:30:10",
      "online_count": 1
    }
  }
}
```

**错误**：`你当前已在其他会议中` (400) / `会议号冲突，请重试` (400)。

---

## 2. 我的会议列表

`GET /api/v1/meeting/rooms/mine`

**查询参数**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| status | int | 否 | 过滤状态：0=未开始 / 1=进行中 / 2=已结束；不传=全部 |
| before_id | int64 | 否 | 游标：仅返回 id < before_id 的记录 |
| limit | int | 否 | 页大小，默认 20，最大 50 |

**响应 `200 OK`**

```json
{
  "code": 0,
  "data": {
    "list": [
      { "id": 12, "room_code": "835-000-036", "title": "T5 验证会议", "status": 1, ... }
    ],
    "has_more": false
  }
}
```

**说明**：返回 host + 参会者两类记录合并后的最近会议，按 `id DESC` 排序。

---

## 3. 会议详情

`GET /api/v1/meeting/rooms/:code`

**权限**：调用方必须是该会议的当前活跃参会者（host 或 participant，left_at IS NULL）。

**响应 `200 OK`**

```json
{
  "code": 0,
  "data": {
    "room": { "id": 12, "room_code": "835-000-036", ... },
    "participants": [
      { "id": 30, "room_id": 12, "user_id": 16, "role": 1, "role_label": "主持人", "is_active": true, "joined_at": "...", "duration": 0 },
      { "id": 31, "room_id": 12, "user_id": 17, "role": 0, "role_label": "参会者", "is_active": true, "joined_at": "...", "duration": 0 }
    ],
    "online_count": 2
  }
}
```

**错误**：`会议不存在` (404) / `你当前未在会议中` (400)。

---

## 4. 加入会议

`POST /api/v1/meeting/rooms/:code/join`

**请求体**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| password | string | 条件 | 房间 `has_password=true` 时必传 |

**校验顺序**：房间存在 → 未结束 → 单点参会（若已在其他活跃会议则 400）→ 密码锁定（5 次错误封禁 10 分钟）→ 密码校验 → 容量 → 写 participant → 广播 `meeting.member.joined`。

**首次加入**：创建 `meeting_participants` 新行。
**复入**（之前 left）：复用原行重置 `left_at=NULL`、`joined_at=now`、`duration=0`，避免审计表膨胀。

**响应 `200 OK`**

```json
{
  "code": 0,
  "data": {
    "room": { ... },
    "participant": { ... },
    "router_id": "stub-router-835-000-036"
  }
}
```

`router_id` 当前为 Noop 占位，Task 7 接入 Node media-server 后改为真实 mediasoup Router ID，前端据此建立 WebSocket 订阅。

---

## 5. 离开会议

`POST /api/v1/meeting/rooms/:code/leave`

**行为**：
- `participant.left_at = now`、`duration = EXTRACT(EPOCH FROM now - joined_at)`；
- 若离开者是 host 且仍有其他活跃成员 → 自动将 host 转让给**最早加入的活跃成员**，广播 `meeting.host.changed`；
- 若房间无剩余活跃成员 → 标记 `status=2 ended_reason=empty_ttl` 并触发 mediaOrchestrator.CloseRouter；
- 广播 `meeting.member.left`。

**响应 `200 OK`**

```json
{ "code": 0, "data": { "duration": 185 } }
```

---

## 6. 结束会议

`POST /api/v1/meeting/rooms/:code/end` — host 专用。

**行为**：所有活跃成员强制离会（`left_reason=host_end`），房间 `status=2 ended_reason=host_ended`，广播 `meeting.room.ended`，调用 `mediaOrchestrator.CloseRouter`。

**错误**：`仅主持人可操作` (403)。

---

## 7. 转让主持人

`POST /api/v1/meeting/rooms/:code/transfer-host`

**请求体**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| target_user_id | int64 | 是 | 新 host 的用户 ID，必须是当前活跃参会者且不是自己 |

**行为**：`meeting_rooms.host_id` 更新 + `meeting_participants.role` 对调（事务），广播 `meeting.host.changed`。

**错误**：`仅主持人可操作` (403) / `不能将主持人转让给自己` (400) / `目标用户不是当前活跃参会者` (400)。

---

## 8. 踢出成员

`POST /api/v1/meeting/rooms/:code/kick`

**请求体**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| user_id | int64 | 是 | 被踢用户 ID |

**行为**：标记 `left_at=now`、`left_reason=kicked`；对被踢者定向推送 `meeting.member.kicked`（前端收到后跳首页），同时房间广播 `meeting.member.left`。

**错误**：`仅主持人可操作` (403) / `不能踢自己` (400) / `目标用户不是当前活跃参会者` (400)。

---

## 9. 邀请用户

`POST /api/v1/meeting/rooms/:code/invite`

**请求体**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| invitee_ids | int64[] | 是 | 被邀请用户 ID 数组，1~50 个，自动去重 |

**行为**：对每个 invitee：
1. 若该用户已在本会议活跃 → 跳过；
2. 生成 32 位十六进制 Token 写 Redis key `echo:meeting:invite:{token}`，TTL 600 秒（`MeetingInviteTokenTTL`），value = `{"room_code","inviter_id","invitee_id","has_password"}`；
3. 通过 Phase 2e-1 `notify.Pusher.PushBatch` 推送 `type=meeting_invite` 通知，`Extra` 含 `room_code / room_title / has_password / invite_token`；
4. 离线被邀请者走通知入库，上线后由 WS 或未读轮询获得。

**响应 `200 OK`**

```json
{ "code": 0, "data": { "pushed": 1, "skipped": 0 } }
```

出于安全考虑，响应体**不包含** token（token 仅通过通知 Extra 定向下发给被邀请者）。

---

## 10. 兑换邀请链接

`POST /api/v1/meeting/invite-tokens/:token/redeem`

**行为**：查询 Redis key，若存在则返回 `room_code + inviter_id + has_password`，前端据此决定弹密码框再调 `POST /rooms/:code/join`。Token 兑换后**保留 60 秒冗余**（不立即删除，允许用户刷新页面二次兑换），随后由 Redis TTL 自然过期。

**响应 `200 OK`**

```json
{
  "code": 0,
  "data": {
    "room_code": "835-000-036",
    "inviter_id": 16,
    "has_password": false
  }
}
```

**错误**：`邀请链接已失效` (400)。

---

## 11. 发送会议内聊天

`POST /api/v1/meeting/rooms/:code/chats`

**请求体**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| content | string | 是 | 消息文本，1~500 字符 |

**行为**：写 `meeting_chats` 后向房间内活跃成员广播 WS 事件 `meeting.chat`，载荷为响应中的 `message` 对象。

**响应 `201 Created`**

```json
{
  "code": 0,
  "data": {
    "message": {
      "id": 18,
      "room_id": 12,
      "user_id": 17,
      "content": "hello",
      "created_at": "2026-04-21 16:32:05"
    }
  }
}
```

---

## 12. 拉取会议历史聊天

`GET /api/v1/meeting/rooms/:code/chats`

**查询参数**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| before_id | int64 | 否 | 游标：仅返回 id < before_id 的消息 |
| limit | int | 否 | 页大小，默认 30，最大 100 |

**响应 `200 OK`**

```json
{
  "code": 0,
  "data": {
    "list": [
      { "id": 18, "room_id": 12, "user_id": 17, "content": "hello", "created_at": "..." }
    ],
    "has_more": false
  }
}
```

**保留策略**：定时任务每日清理 `status=2 AND ended_at < NOW() - 24h` 的房间聊天记录（不进入 IM 消息流）。

---

## WebSocket 事件关联

参考 [websocket.md](../websocket.md) 的 `meeting.*` 事件族。Task 5 内部当前使用 `ws.PubSub.PublishToUser` 对活跃参会者逐个推送（Task 6 将封装为 `BroadcastToMeeting`，接口无感替换）：

| 事件 | 触发 REST | 载荷关键字段 |
|------|-----------|-------------|
| `meeting.member.joined` | JoinRoom | room_code / participant |
| `meeting.member.left`   | LeaveRoom / KickMember | room_code / user_id / reason |
| `meeting.member.kicked` | KickMember（仅发给被踢者） | room_code |
| `meeting.host.changed`  | TransferHost / 隐式（host 离会后自动转让） | room_code / old_host_id / new_host_id |
| `meeting.room.ended`    | EndRoom / 空房 TTL | room_code / reason |
| `meeting.chat`          | SendChat | message 对象 |

---

## 验证记录

Task 5 端到端验证脚本（`/tmp/meeting_t5_test.sh`）结果：**19/19 PASS**，覆盖 12 接口的 happy path 与 5 类错误路径（密码错误 / 房间不存在 / 单点参会冲突 / 非 host 越权 / 邀请链接失效）。

---

## 后续任务关联

- **Task 6**：WebSocket 信令协议（`meeting.*` 事件、mediasoup transport/producer/consumer 流转）
- **Task 7**：Go → Node HTTP 客户端，将 `NoopMediaOrchestrator` 替换为 `HTTPMediaOrchestrator`，接入真实 mediasoup Router
- **Task 13**：通知卡片 UI 补齐 `meeting_invite` 内联按钮
