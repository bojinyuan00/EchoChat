// Package ws 提供 WebSocket 连接处理
// 负责 HTTP → WebSocket 升级、JWT 认证、消息路由分发
package ws

import (
	"context"
	"net/http"

	"github.com/echochat/backend/config"
	"github.com/echochat/backend/pkg/logs"
	"github.com/echochat/backend/pkg/utils"
	"github.com/echochat/backend/pkg/ws"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// TokenValidator 有状态 JWT 验证接口（检查 Token 是否在 Redis 中有效）
// 由 auth.AuthService 实现，用于防止已登出用户建立 WebSocket 连接
type TokenValidator interface {
	ValidateAccessToken(ctx context.Context, userID int64, clientType, token string) bool
}

// OfflineMessagePusher 离线消息推送接口
// 由 im.handler.OfflinePusher 实现，WebSocket 连接建立后触发推送
type OfflineMessagePusher interface {
	PushOfflineMessages(ctx context.Context, userID int64)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // TODO: 生产环境通过配置限制 allowed origins
	},
}

// Handler WebSocket 连接处理器
type Handler struct {
	hub            *ws.Hub
	pubsub         *ws.PubSub
	jwtCfg         *config.JWTConfig
	onlineService  *OnlineService
	tokenValidator TokenValidator
	offlinePusher  OfflineMessagePusher
}

// NewHandler 创建 WebSocket Handler 实例
func NewHandler(hub *ws.Hub, pubsub *ws.PubSub, jwtCfg *config.JWTConfig, onlineService *OnlineService, tokenValidator TokenValidator) *Handler {
	return &Handler{
		hub:            hub,
		pubsub:         pubsub,
		jwtCfg:         jwtCfg,
		onlineService:  onlineService,
		tokenValidator: tokenValidator,
	}
}

// SetOfflinePusher 设置离线消息推送器（由 IM 模块在初始化时注入）
func (h *Handler) SetOfflinePusher(pusher OfflineMessagePusher) {
	h.offlinePusher = pusher
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

	clientType := claims.ClientType
	if clientType == "" {
		clientType = "frontend"
	}
	if h.tokenValidator != nil && !h.tokenValidator.ValidateAccessToken(c.Request.Context(), claims.UserID, clientType, token) {
		logs.Warn(nil, funcName, "WebSocket Token 已失效（Redis 校验）",
			zap.Int64("user_id", claims.UserID))
		utils.ResponseUnauthorized(c, "认证已失效，请重新登录")
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logs.Error(nil, funcName, "WebSocket 升级失败",
			zap.Int64("user_id", claims.UserID), zap.Error(err))
		return
	}

	client := ws.NewClient(h.hub, conn, claims.UserID)
	client.SetOnDisconnect(func(userID int64) {
		if client.IsClosedByHub() && h.hub.IsOnline(userID) {
			logs.Info(nil, "ws.handler.onDisconnect", "连接被新连接替换，跳过下线清理",
				zap.Int64("user_id", userID))
			return
		}
		h.pubsub.Unsubscribe(userID)
		h.onlineService.UserOffline(context.Background(), userID)
	})
	h.hub.Register(client)
	h.pubsub.Subscribe(claims.UserID)
	h.onlineService.UserOnline(c.Request.Context(), claims.UserID, c.ClientIP())

	logs.Info(nil, funcName, "WebSocket 连接建立",
		zap.Int64("user_id", claims.UserID),
		zap.String("ip", c.ClientIP()))

	go client.WritePump()
	go client.ReadPump(h.createReadHandler(claims.UserID))

	if h.offlinePusher != nil {
		go h.offlinePusher.PushOfflineMessages(context.Background(), claims.UserID)
	}
}

// createReadHandler 创建带生命周期管理的消息处理函数
// 优先查 Hub 事件路由表（业务模块注册的处理器），未命中再走内置 fallback
func (h *Handler) createReadHandler(userID int64) ws.MessageHandler {
	return func(client *ws.Client, msg *ws.Message) {
		funcName := "ws.handler.onMessage"
		logs.Debug(nil, funcName, "收到 WebSocket 消息",
			zap.Int64("user_id", client.UserID),
			zap.String("event", msg.Event),
			zap.Int64("seq", msg.Seq))

		// 优先查事件路由表（IM、Meeting 等模块注册的处理器）
		if h.hub.DispatchEvent(client, msg) {
			return
		}

		// 内置事件 fallback
		switch msg.Event {
		case "heartbeat":
			h.onlineService.HeartbeatRenew(context.Background(), userID)
			resp := ws.NewResponse(msg.Event, msg.Seq, 0, "pong", nil)
			data, err := ws.MarshalResponse(resp)
			if err != nil {
				logs.Error(nil, funcName, "序列化心跳响应失败", zap.Error(err))
				return
			}
			client.Send(data)
		default:
			logs.Warn(nil, funcName, "未知事件类型",
				zap.String("event", msg.Event),
				zap.Int64("user_id", client.UserID))
			resp := ws.NewResponse(msg.Event, msg.Seq, -1, "未知事件", nil)
			data, err := ws.MarshalResponse(resp)
			if err != nil {
				logs.Error(nil, funcName, "序列化响应失败", zap.Error(err))
				return
			}
			client.Send(data)
		}
	}
}

// GetHub 返回 Hub 实例（供在线状态等模块访问）
func (h *Handler) GetHub() *ws.Hub {
	return h.hub
}

// GetPubSub 返回 PubSub 实例（供业务模块发送推送）
func (h *Handler) GetPubSub() *ws.PubSub {
	return h.pubsub
}
