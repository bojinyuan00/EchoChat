// Package ws 提供 WebSocket 连接处理
// 负责 HTTP → WebSocket 升级、JWT 认证、消息路由分发
package ws

import (
	"net/http"

	"github.com/echochat/backend/config"
	"github.com/echochat/backend/pkg/logs"
	"github.com/echochat/backend/pkg/utils"
	"github.com/echochat/backend/pkg/ws"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 开发阶段允许所有来源，生产环境需限制
	},
}

// Handler WebSocket 连接处理器
type Handler struct {
	hub    *ws.Hub
	pubsub *ws.PubSub
	jwtCfg *config.JWTConfig
}

// NewHandler 创建 WebSocket Handler 实例
func NewHandler(hub *ws.Hub, pubsub *ws.PubSub, jwtCfg *config.JWTConfig) *Handler {
	return &Handler{
		hub:    hub,
		pubsub: pubsub,
		jwtCfg: jwtCfg,
	}
}

// Upgrade 处理 WebSocket 升级请求
// GET /ws?token=xxx → JWT 认证 → 升级连接 → 注册 Hub → 订阅 Redis 频道
func (h *Handler) Upgrade(c *gin.Context) {
	funcName := "ws.handler.Upgrade"

	token := c.Query("token")
	if token == "" {
		utils.ResponseUnauthorized(c, "缺少认证 Token")
		return
	}

	claims, err := utils.ParseToken(h.jwtCfg, token)
	if err != nil {
		logs.Warn(nil, funcName, "WebSocket Token 验证失败", zap.Error(err))
		utils.ResponseUnauthorized(c, "Token 无效或已过期")
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logs.Error(nil, funcName, "WebSocket 升级失败",
			zap.Int64("user_id", claims.UserID), zap.Error(err))
		return
	}

	client := ws.NewClient(h.hub, conn, claims.UserID)
	h.hub.Register(client)
	h.pubsub.Subscribe(claims.UserID)

	logs.Info(nil, funcName, "WebSocket 连接建立",
		zap.Int64("user_id", claims.UserID),
		zap.String("ip", c.ClientIP()))

	go client.WritePump()
	go client.ReadPump(h.onMessage)
}

// onMessage 处理客户端发来的 WebSocket 消息
// 根据 event 类型分发到不同的处理逻辑
func (h *Handler) onMessage(client *ws.Client, msg *ws.Message) {
	funcName := "ws.handler.onMessage"
	logs.Debug(nil, funcName, "收到 WebSocket 消息",
		zap.Int64("user_id", client.UserID),
		zap.String("event", msg.Event),
		zap.Int64("seq", msg.Seq))

	// Phase 2a 阶段暂无需要客户端主动发送的事件
	// Phase 2b 将在此处添加 im.message.send 等事件路由
	resp := ws.NewResponse(msg.Event, msg.Seq, 0, "ok", nil)
	data, err := ws.MarshalResponse(resp)
	if err != nil {
		logs.Error(nil, funcName, "序列化响应失败", zap.Error(err))
		return
	}
	client.Send(data)
}

// GetHub 返回 Hub 实例（供在线状态等模块访问）
func (h *Handler) GetHub() *ws.Hub {
	return h.hub
}

// GetPubSub 返回 PubSub 实例（供业务模块发送推送）
func (h *Handler) GetPubSub() *ws.PubSub {
	return h.pubsub
}
