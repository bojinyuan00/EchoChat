package ws

import "github.com/gin-gonic/gin"

// RegisterRoutes 注册 WebSocket 相关路由
func RegisterRoutes(engine *gin.Engine, handler *Handler) {
	engine.GET("/ws", handler.Upgrade)
}
