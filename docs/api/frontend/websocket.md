# 前端 WebSocket 事件协议

> 完整的 WebSocket 协议文档见 [websocket.md](../websocket.md)
> 本文档补充前端联系人模块使用的 WebSocket 事件及对接说明

---

## 连接管理

### 连接地址

| 环境 | 地址 |
|------|------|
| 开发环境 | `ws://localhost:8085/ws?token=<access_token>` |
| 生产环境 | `wss://api.echochat.com/ws?token=<access_token>` |

### 前端实现

- **连接服务：** `frontend/src/services/websocket.js`（WebSocketService 单例）
- **状态管理：** `frontend/src/store/websocket.js`（Pinia Store）
- **联系人监听：** `frontend/src/store/contact.js`（initWsListeners）

### 心跳与重连

- 心跳间隔：30 秒
- 重连策略：指数退避 1s → 2s → 4s → 8s → 16s → 最大 30s
- 连接时自动发送 heartbeat 事件

---

## 联系人相关事件

### heartbeat

**方向：** 客户端 → 服务端

**说明：** 心跳消息，服务端收到后续期在线状态 TTL

**发送格式：**
```json
{
    "event": "heartbeat",
    "data": {}
}
```

---

### notify.friend.request

**方向：** 服务端 → 客户端

**说明：** 收到新的好友申请推送

**data 内容：**
```json
{
    "friendship_id": 5,
    "from_user_id": 2,
    "from_nickname": "李四",
    "from_avatar": "",
    "message": "我是你的同事"
}
```

**前端处理：** `contactStore.initWsListeners` 监听此事件，自动刷新待处理申请列表

---

### contact.request.accepted

**方向：** 服务端 → 客户端

**说明：** 好友申请被对方接受的通知

**data 内容：**
```json
{
    "friendship_id": 5,
    "user_id": 1,
    "username": "zhangsan",
    "nickname": "张三"
}
```

**前端处理：** `contactStore.initWsListeners` 监听此事件，自动刷新好友列表

---

### user.status.online

**方向：** 服务端 → 客户端

**说明：** 好友上线通知

**data 内容：**
```json
{
    "user_id": 2,
    "nickname": "李四"
}
```

**前端处理：** 更新 `contactStore.onlineMap` 和好友列表中对应用户的 `is_online` 状态

---

### user.status.offline

**方向：** 服务端 → 客户端

**说明：** 好友离线通知

**data 内容：**
```json
{
    "user_id": 2
}
```

**前端处理：** 更新 `contactStore.onlineMap` 和好友列表中对应用户的 `is_online` 状态

---

## 事件监听代码示例

```javascript
import { useContactStore } from '@/store/contact'
import { useWebSocketStore } from '@/store/websocket'

const contactStore = useContactStore()
const wsStore = useWebSocketStore()

// 建立 WebSocket 连接
wsStore.connect()

// 初始化联系人事件监听
contactStore.initWsListeners()
```

以上代码会自动监听 `notify.friend.request`、`contact.request.accepted`、`user.status.online`、`user.status.offline` 四个事件。
