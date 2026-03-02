package ws

import (
	"encoding/json"
	"time"

	"github.com/echochat/backend/pkg/logs"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	writeWait      = 10 * time.Second    // 写超时
	pongWait       = 60 * time.Second    // 等待 pong 的超时
	pingPeriod     = 30 * time.Second    // 心跳发送间隔（必须小于 pongWait）
	maxMessageSize = 4096                // 单条消息最大字节数
	sendBufSize    = 256                 // 发送缓冲区大小
)

// Client 封装单个 WebSocket 客户端连接
// 每个连接持有两个 goroutine：readPump（读取客户端消息）和 writePump（写入消息到客户端）
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte // 待发送消息缓冲队列
	UserID int64       // 关联的用户 ID
}

// NewClient 创建客户端实例
func NewClient(hub *Hub, conn *websocket.Conn, userID int64) *Client {
	return &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, sendBufSize),
		UserID: userID,
	}
}

// MessageHandler 消息处理回调函数类型
type MessageHandler func(client *Client, msg *Message)

// ReadPump 读取客户端消息的循环
// 当连接断开或出错时退出，并触发注销流程
func (c *Client) ReadPump(onMessage MessageHandler) {
	defer func() {
		c.hub.Unregister(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, rawMsg, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				logs.Warn(nil, "ws.client.ReadPump", "WebSocket 异常关闭",
					zap.Int64("user_id", c.UserID), zap.Error(err))
			}
			return
		}

		var msg Message
		if err := json.Unmarshal(rawMsg, &msg); err != nil {
			logs.Warn(nil, "ws.client.ReadPump", "消息格式解析失败",
				zap.Int64("user_id", c.UserID), zap.Error(err))
			continue
		}

		if onMessage != nil {
			onMessage(c, &msg)
		}
	}
}

// WritePump 向客户端写入消息的循环
// 从 send channel 读取消息并写入 WebSocket，同时负责心跳
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				logs.Warn(nil, "ws.client.WritePump", "写入消息失败",
					zap.Int64("user_id", c.UserID), zap.Error(err))
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Send 向客户端发送消息（非阻塞，缓冲区满时丢弃）
func (c *Client) Send(data []byte) bool {
	select {
	case c.send <- data:
		return true
	default:
		logs.Warn(nil, "ws.client.Send", "发送缓冲区已满，丢弃消息",
			zap.Int64("user_id", c.UserID))
		return false
	}
}
