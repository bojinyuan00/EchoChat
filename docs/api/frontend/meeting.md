# 会议模块 API (Meeting)

> 通用规范（认证方式、响应格式、错误码）见 [README.md](../README.md)
> 会议中的实时信令（Transport/Producer/Consumer）通过 WebSocket 完成，见 [websocket.md](../websocket.md)

---

## ⚠️ 文档状态说明（2026-04-21 更新）

**本文档为系统总设计阶段的占位版本，下方所列接口清单尚未落地实现。**

Phase 2e-2 已进入设计阶段，专用设计文档 [`docs/plans/2026-04-21-phase2e-2-design.md`](../../plans/2026-04-21-phase2e-2-design.md) §6.2 锁定了 **MVP 最终 12 个接口**的路径与行为。与本文旧版接口清单的关键差异：

| 维度 | 本文档旧版（总设计） | Phase 2e-2 MVP（即将实施） |
|---|---|---|
| 路径前缀 | `/api/v1/meetings/*` | `/api/v1/meeting/rooms/*`（更符合 REST 语义） |
| 范围 | 含预约会议 + 即将开始/历史会议 | MVP 仅即时会议；预约推迟到 Phase 2e-3 |
| 密码存储 | 明文 `password` | bcrypt `password_hash` |
| 主持人控制 | 仅 join/leave | 新增 `kick` / `transfer-host` / 结束会议 / 邀请 |
| 会议内聊天 | 无 | 新增 `POST /:code/chats` + `GET /:code/chats` |
| 邀请链接 | 无 | 新增 `POST /invite-tokens/:token/redeem` |

**实施指引**：
- **Phase 2e-2 开发时**：请以 Phase 2e-2 专用设计文档 §6.2 为唯一实现口径；Task 5（会议 REST 接口）完成后**全量重写**本文档
- **其他模块集成时**：请勿照搬本文旧接口路径；若需调用 meeting 模块 API 请先阅读 Phase 2e-2 设计文档

以下旧版占位内容保留供历史对照，直到 Phase 2e-2 实施完成后整体替换。

---

## 接口列表

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| POST | /api/v1/meetings | 需认证 | 创建即时会议 |
| POST | /api/v1/meetings/schedule | 需认证 | 预约会议 |
| GET | /api/v1/meetings/:code | 需认证 | 获取会议信息 |
| POST | /api/v1/meetings/:code/join | 需认证 | 加入会议 |
| POST | /api/v1/meetings/:code/leave | 需认证 | 离开会议 |
| GET | /api/v1/meetings/upcoming | 需认证 | 获取即将开始的会议 |
| GET | /api/v1/meetings/ongoing | 需认证 | 获取进行中的会议 |
| GET | /api/v1/meetings/history | 需认证 | 获取历史会议 |

---

## 1. 创建即时会议

`POST /api/v1/meetings`

**权限：** 需认证

**请求参数：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| title | string | 是 | 会议标题 |
| password | string | 否 | 会议密码，不设则任何人可加入 |
| max_members | int | 否 | 最大人数，默认 50 |
| settings | object | 否 | 会议设置 |

**settings 可选字段：**

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| mute_on_join | bool | false | 入会时自动静音 |
| allow_recording | bool | false | 是否允许录制（预留） |

**成功响应：**
```json
{
    "code": 0,
    "message": "ok",
    "data": {
        "id": 1,
        "room_code": "123-456-789",
        "title": "产品需求讨论",
        "type": 1,
        "status": 1,
        "host_id": 1,
        "max_members": 50,
        "created_at": "2026-02-27 10:00:00"
    }
}
```

---

## 2. 预约会议

`POST /api/v1/meetings/schedule`

**权限：** 需认证

**请求参数：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| title | string | 是 | 会议标题 |
| scheduled_at | string | 是 | 预约时间，格式：`yyyy-MM-dd HH:mm:ss`，如 `"2026-03-01 14:00:00"` |
| password | string | 否 | 会议密码 |
| max_members | int | 否 | 最大人数 |
| invite_user_ids | int[] | 否 | 预先邀请的用户 ID 列表 |
| settings | object | 否 | 会议设置 |

**说明：** 预约会议创建后 status=0（未开始），被邀请的用户会收到通知。系统在预约时间前 15 分钟和 5 分钟各推送一次提醒。

---

## 3. 获取会议信息

`GET /api/v1/meetings/:code`

**权限：** 需认证

**路径参数：** `code` — 会议号

**成功响应：**
```json
{
    "code": 0,
    "message": "ok",
    "data": {
        "id": 1,
        "room_code": "123-456-789",
        "title": "产品需求讨论",
        "type": 1,
        "status": 1,
        "host": {
            "id": 1,
            "nickname": "张三",
            "avatar": "https://..."
        },
        "has_password": true,
        "max_members": 50,
        "current_members": 5,
        "started_at": "2026-02-27 10:00:00",
        "participants": [
            { "user_id": 1, "nickname": "张三", "role": 1, "joined_at": "2026-02-27 10:00:00" },
            { "user_id": 2, "nickname": "李四", "role": 0, "joined_at": "2026-02-27 10:01:00" }
        ]
    }
}
```

---

## 4. 加入会议

`POST /api/v1/meetings/:code/join`

**权限：** 需认证

**请求参数：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| password | string | 否 | 会议密码（如果会议设有密码） |

**成功响应包含加入会议所需的信令参数。**

**可能的错误码：** 4001, 4002, 4003, 4004

---

## 5. 离开会议

`POST /api/v1/meetings/:code/leave`

**权限：** 需认证

**说明：** 离开后系统自动计算参会时长。如果主持人离开且没有联合主持人，会议将自动结束。

---

## 6. 获取即将开始的会议

`GET /api/v1/meetings/upcoming`

**权限：** 需认证

**说明：** 返回当前用户被邀请的、尚未开始的预约会议列表，按预约时间升序排列。

---

## 7. 获取进行中的会议

`GET /api/v1/meetings/ongoing`

**权限：** 需认证

**说明：** 返回当前用户正在参与的或被邀请的进行中会议。

---

## 8. 获取历史会议

`GET /api/v1/meetings/history`

**权限：** 需认证

**查询参数：** 支持分页（page, page_size）

**说明：** 返回当前用户参与过的已结束会议，按结束时间倒序排列。
