package notify

import (
	"github.com/echochat/backend/app/constants"
	"github.com/echochat/backend/app/notify/controller"
	"github.com/echochat/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册 notify 模块的所有路由
// 前台用户接口需 JWT，后台广播接口额外要求 admin/super_admin 角色
func RegisterRoutes(r *gin.Engine, ctrl *controller.NotificationController, jwtAuth gin.HandlerFunc) {
	// 前台用户接口
	authed := r.Group("/api/v1")
	authed.Use(jwtAuth)
	{
		authed.GET("/notifications", ctrl.GetList)
		authed.GET("/notifications/unread-count", ctrl.GetUnreadCount)
		authed.PUT("/notifications/:id/read", ctrl.MarkRead)
		authed.PUT("/notifications/read-all", ctrl.MarkAllRead)
	}

	// 管理端广播接口：JWT + admin/super_admin 角色
	adminGroup := r.Group("/api/v1/admin")
	adminGroup.Use(jwtAuth, middleware.RequireRole(constants.RoleAdmin, constants.RoleSuperAdmin))
	{
		adminGroup.POST("/notifications/broadcast", ctrl.Broadcast)
	}
}
