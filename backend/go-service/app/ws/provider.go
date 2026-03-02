package ws

import (
	"github.com/echochat/backend/config"
	"github.com/echochat/backend/pkg/ws"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
)

// ProvideHub 创建并启动 Hub 实例
func ProvideHub() *ws.Hub {
	hub := ws.NewHub()
	go hub.Run()
	return hub
}

// ProvidePubSub 创建 PubSub 实例
func ProvidePubSub(rdb *redis.Client, hub *ws.Hub) *ws.PubSub {
	return ws.NewPubSub(rdb, hub)
}

// ProvideWSHandler 创建 WebSocket Handler
func ProvideWSHandler(hub *ws.Hub, pubsub *ws.PubSub, cfg *config.JWTConfig) *Handler {
	return NewHandler(hub, pubsub, cfg)
}

// WSSet WebSocket 模块 Wire Provider Set
var WSSet = wire.NewSet(
	ProvideHub,
	ProvidePubSub,
	ProvideWSHandler,
)
