// Package admin 提供管理后台功能模块
package admin

import (
	"github.com/echochat/backend/app/admin/controller"
	"github.com/echochat/backend/app/constants"
	"github.com/echochat/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册 admin 模块的所有路由
// 所有管理端 API 需要 JWT + admin 角色双重中间件
func RegisterRoutes(
	r *gin.Engine,
	userCtrl *controller.UserManageController,
	onlineCtrl *controller.OnlineController,
	contactManageCtrl *controller.ContactManageController,
	groupManageCtrl *controller.GroupManageController,
	jwtAuth gin.HandlerFunc,
) {
	// 管理端路由组：JWT 认证 + admin/super_admin 角色检查
	adminGroup := r.Group("/api/v1/admin")
	adminGroup.Use(jwtAuth, middleware.RequireRole(constants.RoleAdmin, constants.RoleSuperAdmin))
	{
		// 用户管理
		adminGroup.GET("/users", userCtrl.GetUserList)
		adminGroup.GET("/users/:id", userCtrl.GetUserDetail)
		adminGroup.PUT("/users/:id/status", userCtrl.UpdateUserStatus)
		adminGroup.PUT("/users/:id/roles", userCtrl.SetRoles)
		adminGroup.POST("/users", userCtrl.CreateUser)

		// 角色管理
		adminGroup.GET("/roles", userCtrl.GetAllRoles)

		// 在线监控
		adminGroup.GET("/online/users", onlineCtrl.GetOnlineUsers)
		adminGroup.GET("/online/count", onlineCtrl.GetOnlineCount)

		// 好友关系管理
		adminGroup.GET("/contacts", contactManageCtrl.GetAllContacts)
		adminGroup.DELETE("/contacts/:id", contactManageCtrl.DeleteContact)

		// 群组管理
		adminGroup.GET("/groups", groupManageCtrl.GetGroupList)
		adminGroup.GET("/groups/:id", groupManageCtrl.GetGroupDetail)
		adminGroup.DELETE("/groups/:id", groupManageCtrl.DissolveGroup)
	}
}
