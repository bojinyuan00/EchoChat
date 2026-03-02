package ws

import (
	"sync"

	"github.com/echochat/backend/pkg/logs"
	"go.uber.org/zap"
)

// Hub 管理所有活跃的 WebSocket 客户端连接
// 提供按 userID 注册/注销/查找连接的能力
type Hub struct {
	clients    map[int64]*Client // userID -> Client 映射
	register   chan *Client      // 注册通道
	unregister chan *Client      // 注销通道
	mu         sync.RWMutex     // 保护 clients map 的读写锁
}

// NewHub 创建 Hub 实例
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[int64]*Client),
		register:   make(chan *Client, 256),
		unregister: make(chan *Client, 256),
	}
}

// Run 启动 Hub 主循环，处理连接注册和注销
// 应在单独的 goroutine 中运行
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if old, ok := h.clients[client.UserID]; ok {
				logs.Info(nil, "ws.hub.Run", "用户重复连接，关闭旧连接",
					zap.Int64("user_id", client.UserID))
				close(old.send)
			}
			h.clients[client.UserID] = client
			h.mu.Unlock()

			logs.Info(nil, "ws.hub.Run", "客户端已注册",
				zap.Int64("user_id", client.UserID),
				zap.Int("online_count", h.OnlineCount()))

		case client := <-h.unregister:
			h.mu.Lock()
			if existing, ok := h.clients[client.UserID]; ok && existing == client {
				delete(h.clients, client.UserID)
				close(client.send)
			}
			h.mu.Unlock()

			logs.Info(nil, "ws.hub.Run", "客户端已注销",
				zap.Int64("user_id", client.UserID),
				zap.Int("online_count", h.OnlineCount()))
		}
	}
}

// Register 注册客户端连接
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister 注销客户端连接
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// GetClient 根据 userID 获取客户端连接（线程安全）
func (h *Hub) GetClient(userID int64) (*Client, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	client, ok := h.clients[userID]
	return client, ok
}

// SendToUser 向指定用户发送消息（仅本地 Hub）
// 如果用户不在本实例，返回 false
func (h *Hub) SendToUser(userID int64, data []byte) bool {
	h.mu.RLock()
	client, ok := h.clients[userID]
	h.mu.RUnlock()

	if !ok {
		return false
	}

	select {
	case client.send <- data:
		return true
	default:
		return false
	}
}

// OnlineCount 返回当前在线连接数
func (h *Hub) OnlineCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// OnlineUserIDs 返回所有在线用户 ID 列表
func (h *Hub) OnlineUserIDs() []int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ids := make([]int64, 0, len(h.clients))
	for id := range h.clients {
		ids = append(ids, id)
	}
	return ids
}

// IsOnline 检查指定用户是否在线（本地 Hub）
func (h *Hub) IsOnline(userID int64) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.clients[userID]
	return ok
}
