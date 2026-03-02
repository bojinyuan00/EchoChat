// Package contact 提供联系人/好友管理模块
package contact

import (
	"github.com/echochat/backend/app/contact/controller"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册 contact 模块的所有路由（需要 JWT 中间件）
func RegisterRoutes(r *gin.Engine, ctrl *controller.ContactController, jwtAuth gin.HandlerFunc) {
	authed := r.Group("/api/v1")
	authed.Use(jwtAuth)
	{
		// 好友关系
		authed.GET("/contacts", ctrl.GetFriendList)
		authed.POST("/contacts/request", ctrl.SendFriendRequest)
		authed.POST("/contacts/accept", ctrl.AcceptFriendRequest)
		authed.POST("/contacts/reject", ctrl.RejectFriendRequest)
		authed.DELETE("/contacts/:id", ctrl.DeleteFriend)
		authed.PUT("/contacts/:id/remark", ctrl.UpdateRemark)
		authed.GET("/contacts/requests", ctrl.GetPendingRequests)

		// 好友分组
		authed.GET("/contacts/groups", ctrl.GetGroups)
		authed.POST("/contacts/groups", ctrl.CreateGroup)
		authed.PUT("/contacts/groups/:id", ctrl.UpdateGroup)
		authed.DELETE("/contacts/groups/:id", ctrl.DeleteGroup)
		authed.PUT("/contacts/:id/group", ctrl.MoveToGroup)

		// 黑名单
		authed.POST("/contacts/block", ctrl.BlockUser)
		authed.DELETE("/contacts/block/:user_id", ctrl.UnblockUser)
		authed.GET("/contacts/block", ctrl.GetBlockList)

		// 搜索与推荐
		authed.GET("/users/search", ctrl.SearchUsers)
		authed.GET("/contacts/recommend", ctrl.GetRecommendFriends)
	}
}
