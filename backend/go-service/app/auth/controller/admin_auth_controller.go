package controller

import (
	"github.com/echochat/backend/app/auth/service"
	"github.com/echochat/backend/app/dto"
	"github.com/echochat/backend/pkg/logs"
	"github.com/echochat/backend/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AdminAuthController 后台管理认证控制器
// 与前台 AuthController 的区别：登录后额外检查管理员角色权限
type AdminAuthController struct {
	authService *service.AuthService
}

// NewAdminAuthController 创建后台认证控制器实例
func NewAdminAuthController(authService *service.AuthService) *AdminAuthController {
	return &AdminAuthController{authService: authService}
}

// AdminLogin 管理员登录
// POST /api/v1/admin/auth/login
func (ctrl *AdminAuthController) AdminLogin(c *gin.Context) {
	funcName := "controller.admin_auth_controller.AdminLogin"
	ctx := c.Request.Context()

	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logs.Warn(ctx, funcName, "参数校验失败", zap.Error(err))
		utils.ResponseBadRequest(c, "参数校验失败: "+err.Error())
		return
	}

	logs.Info(ctx, funcName, "管理员登录请求",
		zap.String("account", req.Account),
		zap.String("ip", c.ClientIP()),
	)

	resp, err := ctrl.authService.AdminLogin(ctx, &req, c.ClientIP())
	if err != nil {
		handleAuthError(c, err)
		return
	}

	utils.ResponseOK(c, resp)
}
