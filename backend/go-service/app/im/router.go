// Package im 提供即时通讯模块
package im

import (
	"github.com/echochat/backend/app/im/controller"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册 IM 模块的所有路由（需要 JWT 中间件）
func RegisterRoutes(r *gin.Engine, ctrl *controller.IMController, jwtAuth gin.HandlerFunc) {
	authed := r.Group("/api/v1/im")
	authed.Use(jwtAuth)
	{
		// 会话管理
		authed.GET("/conversations", ctrl.GetConversations)
		authed.PUT("/conversations/:id/pin", ctrl.PinConversation)
		authed.DELETE("/conversations/:id", ctrl.DeleteConversation)
		authed.DELETE("/conversations/:id/messages", ctrl.ClearHistory)

		// 消息
		authed.GET("/messages", ctrl.GetHistoryMessages)
		authed.GET("/messages/search", ctrl.SearchMessages)

		// 未读数
		authed.GET("/unread", ctrl.GetTotalUnread)
	}
}
