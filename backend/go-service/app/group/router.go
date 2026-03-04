package group

import (
	"github.com/echochat/backend/app/group/controller"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册 group 模块的所有路由（需要 JWT 中间件）
func RegisterRoutes(r *gin.Engine, ctrl *controller.GroupController, jwtAuth gin.HandlerFunc) {
	authed := r.Group("/api/v1")
	authed.Use(jwtAuth)
	{
		// 群聊管理
		authed.POST("/groups", ctrl.CreateGroup)
		authed.GET("/groups/search", ctrl.SearchGroups)
		authed.GET("/groups/:id", ctrl.GetGroupDetail)
		authed.PUT("/groups/:id", ctrl.UpdateGroup)
		authed.DELETE("/groups/:id", ctrl.DissolveGroup)

		// 群成员管理
		authed.GET("/groups/:id/members", ctrl.GetMembers)
		authed.POST("/groups/:id/members", ctrl.InviteMembers)
		authed.DELETE("/groups/:id/members/me", ctrl.LeaveGroup)
		authed.DELETE("/groups/:id/members/:uid", ctrl.KickMember)
		authed.PUT("/groups/:id/members/me/nickname", ctrl.UpdateNickname)
		authed.PUT("/groups/:id/members/:uid/role", ctrl.SetMemberRole)
		authed.PUT("/groups/:id/members/:uid/mute", ctrl.MuteMember)

		// 群主转让 + 全体禁言
		authed.PUT("/groups/:id/transfer", ctrl.TransferOwner)
		authed.PUT("/groups/:id/all-mute", ctrl.SetAllMuted)

		// 入群申请
		authed.POST("/groups/:id/join-requests", ctrl.SubmitJoinRequest)
		authed.GET("/groups/:id/join-requests", ctrl.GetJoinRequests)
		authed.PUT("/groups/:id/join-requests/:rid", ctrl.ReviewJoinRequest)
	}
}
