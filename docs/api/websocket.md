# WebSocket 事件协议

> 通用规范（认证方式、响应格式、错误码）见 [README.md](README.md)
> 本文档定义 EchoChat 系统中所有 WebSocket 实时通信事件。

---

## 连接说明

### 连接地址

| 环境 | 地址 |
|------|------|
| 开发环境 | `ws://localhost:8080/ws?token=<access_token>` |
| 生产环境 | `wss://api.echochat.com/ws?token=<access_token>` |

### 连接认证

通过 URL 查询参数 `token` 携带 JWT Access Token，服务端验证通过后建立连接。

### 心跳机制

- 客户端每 **30 秒** 发送一次 ping 帧
- 服务端响应 pong 帧
- 如果 **90 秒** 内未收到客户端心跳，服务端主动断开连接

### 断线重连

- 客户端检测到连接断开后自动重连
- 重连间隔采用指数退避：1s → 2s → 4s → 8s → 16s → 最大 30s
- 重连成功后拉取离线消息

---

## 消息格式

### 客户端发送格式

```json
{
    "event": "im.message.send",
    "seq": 1001,
    "data": { ... },
    "time": "2026-02-27 18:06:40"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| event | string | 事件名称，格式：`{模块}.{对象}.{动作}` |
| seq | int | 消息序列号，客户端自增，用于匹配请求和响应 |
| data | object | 事件数据 |
| time | string | 发送时间，格式：`yyyy-MM-dd HH:mm:ss`，时区 Asia/Shanghai |

### 服务端响应格式（ACK）

```json
{
    "event": "im.message.send.ack",
    "seq": 1001,
    "code": 0,
    "message": "ok",
    "data": { "msg_id": 10086 }
}
```

### 服务端推送格式

```json
{
    "event": "im.message.new",
    "data": { ... },
    "time": "2026-02-27 18:06:40"
}
```

推送类消息没有 seq 字段（不需要客户端确认）。

---

## 即时通讯事件

### im.message.send

**方向：** 客户端 → 服务端

**说明：** 发送消息到会话

**data 参数：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| conversation_id | int | 是 | 目标会话 ID |
| type | int | 是 | 消息类型：1=文本，2=图片，3=文件，4=语音 |
| content | string | 否 | 文本内容 |
| extra | object | 否 | 附加数据（图片/文件信息） |

**ACK 响应 data：** `{ "msg_id": 10086 }`

---

### im.message.new

**方向：** 服务端 → 客户端

**说明：** 收到新消息推送

**data 内容：**

```json
{
    "id": 10086,
    "conversation_id": 1,
    "sender_id": 2,
    "sender_name": "李四",
    "sender_avatar": "https://...",
    "type": 1,
    "content": "你好",
    "extra": {},
    "created_at": "2026-02-27 10:30:00"
}
```

---

### im.message.revoke

**方向：** 客户端 → 服务端

**说明：** 撤回消息（发送后 2 分钟内）

**data 参数：** `{ "message_id": 10086 }`

---

### im.message.read

**方向：** 客户端 → 服务端

**说明：** 消息已读回执

**data 参数：** `{ "conversation_id": 1, "message_id": 10086 }`

---

### im.typing.start

**方向：** 客户端 → 服务端

**说明：** 通知对方"正在输入"

**data 参数：** `{ "conversation_id": 1 }`

---

### im.typing.stop

**方向：** 客户端 → 服务端

**说明：** 停止输入

**data 参数：** `{ "conversation_id": 1 }`

---

## 会议信令事件

### meeting.room.join

**方向：** 客户端 → 服务端

**说明：** 加入会议房间

**data 参数：** `{ "room_code": "123-456-789" }`

**ACK 响应 data：** 房间信息、参与者列表、RTP Capabilities

---

### meeting.room.leave

**方向：** 客户端 → 服务端

**说明：** 离开会议房间

**data 参数：** `{ "room_code": "123-456-789" }`

---

### meeting.room.info

**方向：** 服务端 → 客户端

**说明：** 房间信息同步（成员变更、设置变更时推送）

---

### meeting.member.join

**方向：** 服务端 → 客户端（广播）

**说明：** 有新成员加入会议

**data 内容：**
```json
{
    "room_code": "123-456-789",
    "user_id": 3,
    "nickname": "王五",
    "avatar": "https://...",
    "role": 0
}
```

---

### meeting.member.leave

**方向：** 服务端 → 客户端（广播）

**说明：** 有成员离开会议

**data 内容：** `{ "room_code": "...", "user_id": 3 }`

---

### meeting.member.mute

**方向：** 双向

**说明：** 静音/解除静音

**data 内容：** `{ "room_code": "...", "user_id": 1, "muted": true }`

---

### meeting.member.video

**方向：** 双向

**说明：** 开关摄像头

**data 内容：** `{ "room_code": "...", "user_id": 1, "video_enabled": false }`

---

## mediasoup 信令事件

### meeting.transport.create

**方向：** 客户端 → 服务端

**说明：** 请求创建 WebRTC Transport（发送端或接收端）

**data 参数：** `{ "room_code": "...", "direction": "send" }` 或 `"recv"`

**ACK 响应 data：** Transport 参数（id, iceParameters, iceCandidates, dtlsParameters）

---

### meeting.transport.connect

**方向：** 客户端 → 服务端

**说明：** 完成 Transport DTLS 握手

**data 参数：** `{ "transport_id": "...", "dtls_parameters": { ... } }`

---

### meeting.produce.start

**方向：** 客户端 → 服务端

**说明：** 开始推流（音频或视频）

**data 参数：**
```json
{
    "transport_id": "...",
    "kind": "video",
    "rtp_parameters": { ... }
}
```

**ACK 响应 data：** `{ "producer_id": "..." }`

---

### meeting.produce.stop

**方向：** 客户端 → 服务端

**说明：** 停止推流

**data 参数：** `{ "producer_id": "..." }`

---

### meeting.consume.start

**方向：** 服务端 → 客户端

**说明：** 通知客户端可以开始接收某个参与者的流

**data 内容：**
```json
{
    "consumer_id": "...",
    "producer_id": "...",
    "kind": "video",
    "rtp_parameters": { ... },
    "user_id": 3,
    "nickname": "王五"
}
```

---

### meeting.consume.resume

**方向：** 客户端 → 服务端

**说明：** 恢复被暂停的 Consumer

**data 参数：** `{ "consumer_id": "..." }`

---

## 用户状态事件

### user.status.online

**方向：** 服务端 → 客户端

**说明：** 好友上线通知

**data 内容：** `{ "user_id": 2, "nickname": "李四" }`

---

### user.status.offline

**方向：** 服务端 → 客户端

**说明：** 好友离线通知

**data 内容：** `{ "user_id": 2 }`

---

## 通知事件

### notify.new

**方向：** 服务端 → 客户端

**说明：** 新通知推送（通用）

**data 内容：** 与 Notify API 获取通知列表中的单条通知格式一致

---

### notify.meeting.invite

**方向：** 服务端 → 客户端

**说明：** 会议邀请推送

**data 内容：**
```json
{
    "room_code": "123-456-789",
    "title": "产品需求讨论",
    "from_user_id": 1,
    "from_nickname": "张三"
}
```

---

### notify.friend.request

**方向：** 服务端 → 客户端

**说明：** 好友申请推送

**data 内容：**
```json
{
    "friendship_id": 5,
    "from_user_id": 2,
    "from_nickname": "李四",
    "from_avatar": "https://...",
    "message": "我是你的同事"
}
```
