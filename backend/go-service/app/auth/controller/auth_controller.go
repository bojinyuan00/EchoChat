// Package controller 提供 auth 模块的 HTTP 接口处理
package controller

import (
	"github.com/echochat/backend/app/auth/service"
	"github.com/echochat/backend/app/constants"
	"github.com/echochat/backend/app/dto"
	"github.com/echochat/backend/pkg/logs"
	"github.com/echochat/backend/pkg/middleware"
	"github.com/echochat/backend/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AuthController 前台认证控制器
// 处理用户注册、登录、Token 刷新、个人信息管理等接口
type AuthController struct {
	authService *service.AuthService
}

// NewAuthController 创建前台认证控制器实例
func NewAuthController(authService *service.AuthService) *AuthController {
	return &AuthController{authService: authService}
}

// Register 用户注册
// POST /api/v1/auth/register
func (ctrl *AuthController) Register(c *gin.Context) {
	funcName := "controller.auth_controller.Register"
	ctx := c.Request.Context()

	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logs.Warn(ctx, funcName, "参数校验失败", zap.Error(err))
		utils.ResponseBadRequest(c, "参数校验失败: "+err.Error())
		return
	}

	logs.Info(ctx, funcName, "注册请求",
		zap.String("username", req.Username),
		zap.String("email", logs.MaskEmail(req.Email)),
	)

	resp, err := ctrl.authService.Register(ctx, &req)
	if err != nil {
		handleAuthError(c, err, "注册失败")
		return
	}

	utils.ResponseOK(c, resp)
}

// Login 用户登录
// POST /api/v1/auth/login
func (ctrl *AuthController) Login(c *gin.Context) {
	funcName := "controller.auth_controller.Login"
	ctx := c.Request.Context()

	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logs.Warn(ctx, funcName, "参数校验失败", zap.Error(err))
		utils.ResponseBadRequest(c, "参数校验失败: "+err.Error())
		return
	}

	logs.Info(ctx, funcName, "登录请求",
		zap.String("account", req.Account),
		zap.String("ip", c.ClientIP()),
	)

	resp, err := ctrl.authService.Login(ctx, &req, c.ClientIP(), constants.ClientTypeFrontend)
	if err != nil {
		handleAuthError(c, err, "登录失败")
		return
	}

	utils.ResponseOK(c, resp)
}

// Logout 用户登出
// POST /api/v1/auth/logout（需认证）
func (ctrl *AuthController) Logout(c *gin.Context) {
	funcName := "controller.auth_controller.Logout"
	ctx := c.Request.Context()

	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取用户信息")
		return
	}

	clientType := middleware.GetCurrentClientType(c)
	logs.Info(ctx, funcName, "用户登出",
		zap.Int64("user_id", userID),
		zap.String("client_type", clientType),
	)

	if err := ctrl.authService.Logout(ctx, userID, clientType); err != nil {
		utils.ResponseError(c, "登出失败")
		return
	}

	utils.ResponseOK(c, nil)
}

// RefreshToken 刷新 Access Token
// POST /api/v1/auth/refresh-token
func (ctrl *AuthController) RefreshToken(c *gin.Context) {
	funcName := "controller.auth_controller.RefreshToken"
	ctx := c.Request.Context()

	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logs.Warn(ctx, funcName, "参数校验失败", zap.Error(err))
		utils.ResponseBadRequest(c, "参数校验失败: "+err.Error())
		return
	}

	resp, err := ctrl.authService.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		handleAuthError(c, err, "刷新 Token 失败")
		return
	}

	utils.ResponseOK(c, resp)
}

// GetProfile 获取当前用户信息
// GET /api/v1/auth/profile（需认证）
func (ctrl *AuthController) GetProfile(c *gin.Context) {
	funcName := "controller.auth_controller.GetProfile"
	ctx := c.Request.Context()

	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取用户信息")
		return
	}

	logs.Debug(ctx, funcName, "获取个人信息", zap.Int64("user_id", userID))

	userInfo, err := ctrl.authService.GetProfile(ctx, userID)
	if err != nil {
		handleAuthError(c, err, "获取个人信息失败")
		return
	}

	utils.ResponseOK(c, userInfo)
}

// UpdateProfile 更新个人资料
// PUT /api/v1/auth/profile（需认证）
func (ctrl *AuthController) UpdateProfile(c *gin.Context) {
	funcName := "controller.auth_controller.UpdateProfile"
	ctx := c.Request.Context()

	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取用户信息")
		return
	}

	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logs.Warn(ctx, funcName, "参数校验失败", zap.Error(err))
		utils.ResponseBadRequest(c, "参数校验失败: "+err.Error())
		return
	}

	logs.Info(ctx, funcName, "更新个人资料", zap.Int64("user_id", userID))

	userInfo, err := ctrl.authService.UpdateProfile(ctx, userID, &req)
	if err != nil {
		handleAuthError(c, err, "更新个人资料失败")
		return
	}

	utils.ResponseOK(c, userInfo)
}

// ChangePassword 修改密码
// PUT /api/v1/auth/password（需认证）
func (ctrl *AuthController) ChangePassword(c *gin.Context) {
	funcName := "controller.auth_controller.ChangePassword"
	ctx := c.Request.Context()

	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取用户信息")
		return
	}

	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logs.Warn(ctx, funcName, "参数校验失败", zap.Error(err))
		utils.ResponseBadRequest(c, "参数校验失败: "+err.Error())
		return
	}

	logs.Info(ctx, funcName, "修改密码", zap.Int64("user_id", userID))

	if err := ctrl.authService.ChangePassword(ctx, userID, &req); err != nil {
		handleAuthError(c, err, "修改密码失败")
		return
	}

	utils.ResponseOK(c, nil)
}

// handleAuthError 统一认证业务错误映射
// 已知业务错误 → 返回 Service 层定义的具体提示（如"账号已被禁用"）
// 未知错误 → 返回 fallbackMsg（未传则默认"服务器内部错误"）
func handleAuthError(c *gin.Context, err error, fallbackMsg ...string) {
	switch err {
	case service.ErrUserAlreadyExists:
		utils.ResponseBadRequest(c, err.Error())
	case service.ErrUserNotFound:
		utils.ResponseNotFound(c, err.Error())
	case service.ErrPasswordWrong:
		utils.ResponseUnauthorized(c, "账号或密码错误")
	case service.ErrUserDisabled:
		utils.ResponseForbidden(c, err.Error())
	case service.ErrUserDeleted:
		utils.ResponseForbidden(c, err.Error())
	case service.ErrNotAdmin:
		utils.ResponseForbidden(c, err.Error())
	case service.ErrRefreshTokenType:
		utils.ResponseBadRequest(c, err.Error())
	default:
		msg := "服务器内部错误"
		if len(fallbackMsg) > 0 && fallbackMsg[0] != "" {
			msg = fallbackMsg[0]
		}
		utils.ResponseError(c, msg)
	}
}
