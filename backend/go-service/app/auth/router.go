package auth

import (
	"github.com/echochat/backend/app/auth/controller"
	"github.com/echochat/backend/app/constants"
	"github.com/echochat/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册 auth 模块的所有路由
// 包含前台公开路由、前台需认证路由、后台管理认证路由
// authMiddleware 为 JWT 认证中间件，由外部传入以解耦
func RegisterRoutes(
	r *gin.Engine,
	ctrl *controller.AuthController,
	adminCtrl *controller.AdminAuthController,
	authMiddleware gin.HandlerFunc,
) {
	// ======================== 前台用户端 ========================

	// 前台公开路由（无需认证）
	public := r.Group("/api/v1/auth")
	{
		public.POST("/register", ctrl.Register)
		public.POST("/login", ctrl.Login)
		public.POST("/refresh-token", ctrl.RefreshToken)
	}

	// 前台需认证路由
	authed := r.Group("/api/v1/auth")
	authed.Use(authMiddleware)
	{
		authed.POST("/logout", ctrl.Logout)
		authed.GET("/profile", ctrl.GetProfile)
		authed.PUT("/profile", ctrl.UpdateProfile)
		authed.PUT("/password", ctrl.ChangePassword)
	}

	// ======================== 后台管理端 ========================

	// 管理端公开路由（无需认证，登录后在 Service 层检查管理员角色）
	adminPublic := r.Group("/api/v1/admin/auth")
	{
		adminPublic.POST("/login", adminCtrl.AdminLogin)
	}

	// 管理端需认证路由（JWT + admin 角色双重检查）
	// 复用 AuthController handler，JWT 中的 client_type=admin 保证 Redis 正确隔离
	adminAuthed := r.Group("/api/v1/admin/auth")
	adminAuthed.Use(authMiddleware, middleware.RequireRole(constants.RoleAdmin, constants.RoleSuperAdmin))
	{
		adminAuthed.POST("/logout", ctrl.Logout)
		adminAuthed.GET("/profile", ctrl.GetProfile)
		adminAuthed.PUT("/profile", ctrl.UpdateProfile)
		adminAuthed.PUT("/password", ctrl.ChangePassword)
	}
}
