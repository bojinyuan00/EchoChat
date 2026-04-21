// Package meeting 会议模块（Phase 2e-2）
package meeting

import (
	"github.com/echochat/backend/app/meeting/controller"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册 meeting 模块的全部前台路由
// 所有接口统一挂载在 /api/v1/meeting/* 前缀下，均需 JWT 认证
// 设计文档 §6.2 的 12 个接口 Task 4 骨架阶段仅搭路由层，业务实现留待 Task 5/6/7
func RegisterRoutes(r *gin.Engine, ctrl *controller.MeetingController, jwtAuth gin.HandlerFunc) {
	authed := r.Group("/api/v1/meeting")
	authed.Use(jwtAuth)
	{
		// 会议房间生命周期
		authed.POST("/rooms", ctrl.CreateRoom)
		authed.GET("/rooms", ctrl.ListMyMeetings)
		authed.GET("/rooms/:code", ctrl.GetRoom)
		authed.POST("/rooms/:code/join", ctrl.JoinRoom)
		authed.POST("/rooms/:code/leave", ctrl.LeaveRoom)
		authed.POST("/rooms/:code/end", ctrl.EndRoom)

		// 主持人四件套（Task 7）
		authed.POST("/rooms/:code/transfer-host", ctrl.TransferHost)
		authed.POST("/rooms/:code/kick", ctrl.KickMember)

		// 邀请（Task 5）
		authed.POST("/rooms/:code/invite", ctrl.InviteUsers)
		authed.POST("/invites/:token/redeem", ctrl.RedeemInvite)

		// 会议内聊天（Task 6）
		authed.POST("/rooms/:code/chats", ctrl.SendChat)
		authed.GET("/rooms/:code/chats", ctrl.ListChats)
	}
}
