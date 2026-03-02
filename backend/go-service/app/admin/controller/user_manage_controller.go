// Package controller 提供 admin 模块的 HTTP 请求处理
package controller

import (
	"strconv"

	"github.com/echochat/backend/app/admin/service"
	"github.com/echochat/backend/app/dto"
	"github.com/echochat/backend/pkg/logs"
	"github.com/echochat/backend/pkg/middleware"
	"github.com/echochat/backend/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// UserManageController 管理端用户管理控制器
type UserManageController struct {
	userManageService *service.UserManageService
}

// NewUserManageController 创建用户管理控制器实例
func NewUserManageController(svc *service.UserManageService) *UserManageController {
	return &UserManageController{userManageService: svc}
}

// GetUserList 获取用户列表
// GET /api/v1/admin/users?page=1&page_size=10&keyword=xxx&status=1
func (ctl *UserManageController) GetUserList(c *gin.Context) {
	funcName := "controller.user_manage_controller.GetUserList"
	ctx := c.Request.Context()

	var req dto.UserListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		logs.Warn(ctx, funcName, "参数校验失败", zap.Error(err))
		utils.ResponseBadRequest(c, "参数错误: "+err.Error())
		return
	}

	logs.Info(ctx, funcName, "获取用户列表",
		zap.Int("page", req.Page),
		zap.Int("page_size", req.PageSize),
		zap.String("keyword", req.Keyword),
	)

	resp, err := ctl.userManageService.GetUserList(ctx, &req)
	if err != nil {
		logs.Error(ctx, funcName, "获取用户列表失败", zap.Error(err))
		utils.ResponseError(c, "获取用户列表失败")
		return
	}

	utils.ResponseOK(c, resp)
}

// GetUserDetail 获取用户详情
// GET /api/v1/admin/users/:id
func (ctl *UserManageController) GetUserDetail(c *gin.Context) {
	funcName := "controller.user_manage_controller.GetUserDetail"
	ctx := c.Request.Context()

	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ResponseBadRequest(c, "无效的用户 ID")
		return
	}

	logs.Info(ctx, funcName, "获取用户详情", zap.Int64("user_id", userID))

	detail, err := ctl.userManageService.GetUserDetail(ctx, userID)
	if err != nil {
		if err == service.ErrUserNotFound {
			utils.ResponseNotFound(c, "用户不存在")
			return
		}
		logs.Error(ctx, funcName, "获取用户详情失败", zap.Error(err))
		utils.ResponseError(c, "获取用户详情失败")
		return
	}

	utils.ResponseOK(c, detail)
}

// UpdateUserStatus 更新用户状态（启用/禁用）
// PUT /api/v1/admin/users/:id/status
func (ctl *UserManageController) UpdateUserStatus(c *gin.Context) {
	funcName := "controller.user_manage_controller.UpdateUserStatus"
	ctx := c.Request.Context()

	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ResponseBadRequest(c, "无效的用户 ID")
		return
	}

	var req dto.UpdateUserStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logs.Warn(ctx, funcName, "参数校验失败", zap.Error(err))
		utils.ResponseBadRequest(c, "参数错误: "+err.Error())
		return
	}

	adminUserID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	logs.Info(ctx, funcName, "更新用户状态",
		zap.Int64("target_user_id", userID),
		zap.Int("status", req.Status),
		zap.Int64("admin_user_id", adminUserID),
	)

	if err := ctl.userManageService.UpdateUserStatus(ctx, userID, req.Status, adminUserID); err != nil {
		switch err {
		case service.ErrUserNotFound:
			utils.ResponseNotFound(c, "用户不存在")
		case service.ErrCannotDisableSelf:
			utils.ResponseBadRequest(c, "不能禁用自己的账号")
		case service.ErrInvalidStatus:
			utils.ResponseBadRequest(c, "无效的用户状态")
		case service.ErrInsufficientPermission:
			utils.ResponseForbidden(c, "权限不足，无法操作更高等级的用户")
		default:
			logs.Error(ctx, funcName, "更新用户状态失败", zap.Error(err))
			utils.ResponseError(c, "更新用户状态失败")
		}
		return
	}

	utils.ResponseOK(c, nil)
}

// SetRoles 批量设置用户角色
// PUT /api/v1/admin/users/:id/roles
func (ctl *UserManageController) SetRoles(c *gin.Context) {
	funcName := "controller.user_manage_controller.SetRoles"
	ctx := c.Request.Context()

	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ResponseBadRequest(c, "无效的用户 ID")
		return
	}

	var req dto.SetRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logs.Warn(ctx, funcName, "参数校验失败", zap.Error(err))
		utils.ResponseBadRequest(c, "参数错误: "+err.Error())
		return
	}

	adminUserID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	logs.Info(ctx, funcName, "设置用户角色",
		zap.Int64("target_user_id", userID),
		zap.Strings("role_codes", req.RoleCodes),
		zap.Int64("admin_user_id", adminUserID),
	)

	if err := ctl.userManageService.SetUserRoles(ctx, userID, req.RoleCodes, adminUserID); err != nil {
		switch err {
		case service.ErrUserNotFound:
			utils.ResponseNotFound(c, "用户不存在")
		case service.ErrInvalidRole:
			utils.ResponseBadRequest(c, "包含无效的角色代码")
		case service.ErrInsufficientPermission:
			utils.ResponseForbidden(c, "权限不足，无法操作更高等级的用户")
		case service.ErrCannotAssignHigherRole:
			utils.ResponseForbidden(c, "不能分配高于自身等级的角色")
		default:
			logs.Error(ctx, funcName, "设置用户角色失败", zap.Error(err))
			utils.ResponseError(c, "设置用户角色失败")
		}
		return
	}

	utils.ResponseOK(c, nil)
}

// GetAllRoles 获取所有角色列表
// GET /api/v1/admin/roles
func (ctl *UserManageController) GetAllRoles(c *gin.Context) {
	funcName := "controller.user_manage_controller.GetAllRoles"
	ctx := c.Request.Context()

	logs.Info(ctx, funcName, "获取所有角色列表")

	roles, err := ctl.userManageService.GetAllRoles(ctx)
	if err != nil {
		logs.Error(ctx, funcName, "获取角色列表失败", zap.Error(err))
		utils.ResponseError(c, "获取角色列表失败")
		return
	}

	utils.ResponseOK(c, roles)
}

// CreateUser 管理员手动创建用户
// POST /api/v1/admin/users
func (ctl *UserManageController) CreateUser(c *gin.Context) {
	funcName := "controller.user_manage_controller.CreateUser"
	ctx := c.Request.Context()

	var req dto.AdminCreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logs.Warn(ctx, funcName, "参数校验失败", zap.Error(err))
		utils.ResponseBadRequest(c, "参数错误: "+err.Error())
		return
	}

	adminUserID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	logs.Info(ctx, funcName, "管理员创建用户",
		zap.String("username", req.Username),
		zap.String("email", logs.MaskEmail(req.Email)),
		zap.Int64("admin_user_id", adminUserID),
	)

	userInfo, err := ctl.userManageService.CreateUser(ctx, &req, adminUserID)
	if err != nil {
		switch err {
		case service.ErrUserExists:
			utils.ResponseBadRequest(c, "用户名或邮箱已被注册")
		case service.ErrInvalidRole:
			utils.ResponseBadRequest(c, "无效的角色代码")
		case service.ErrCannotAssignHigherRole:
			utils.ResponseForbidden(c, "不能分配高于自身等级的角色")
		default:
			logs.Error(ctx, funcName, "创建用户失败", zap.Error(err))
			utils.ResponseError(c, "创建用户失败")
		}
		return
	}

	utils.ResponseOK(c, userInfo)
}
