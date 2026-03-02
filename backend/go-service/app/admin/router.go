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
	ctrl *controller.UserManageController,
	jwtAuth gin.HandlerFunc,
) {
	// 管理端路由组：JWT 认证 + admin/super_admin 角色检查
	adminGroup := r.Group("/api/v1/admin")
	adminGroup.Use(jwtAuth, middleware.RequireRole(constants.RoleAdmin, constants.RoleSuperAdmin))
	{
		// 用户管理
		adminGroup.GET("/users", ctrl.GetUserList)
		adminGroup.GET("/users/:id", ctrl.GetUserDetail)
		adminGroup.PUT("/users/:id/status", ctrl.UpdateUserStatus)
		adminGroup.PUT("/users/:id/roles", ctrl.SetRoles)
		adminGroup.POST("/users", ctrl.CreateUser)

		// 角色管理
		adminGroup.GET("/roles", ctrl.GetAllRoles)
	}
}
