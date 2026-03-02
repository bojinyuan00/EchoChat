// Package controller 提供 admin 模块的 HTTP 接口处理
package controller

import (
	"github.com/echochat/backend/app/admin/service"
	"github.com/echochat/backend/pkg/logs"
	"github.com/echochat/backend/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// OnlineController 在线用户管理控制器（管理端）
type OnlineController struct {
	onlineManageService *service.OnlineManageService
}

// NewOnlineController 创建 OnlineController 实例
func NewOnlineController(onlineManageService *service.OnlineManageService) *OnlineController {
	return &OnlineController{onlineManageService: onlineManageService}
}

// GetOnlineUsers 获取在线用户列表
// GET /api/v1/admin/online/users
func (ctl *OnlineController) GetOnlineUsers(c *gin.Context) {
	funcName := "admin.online_controller.GetOnlineUsers"
	ctx := c.Request.Context()

	users, err := ctl.onlineManageService.GetOnlineUsers(ctx)
	if err != nil {
		logs.Error(ctx, funcName, "获取在线用户列表失败", zap.Error(err))
		utils.ResponseError(c, "获取在线用户列表失败")
		return
	}

	logs.Info(ctx, funcName, "获取在线用户列表成功", zap.Int("count", len(users)))
	utils.ResponseOK(c, users)
}

// GetOnlineCount 获取在线用户总数
// GET /api/v1/admin/online/count
func (ctl *OnlineController) GetOnlineCount(c *gin.Context) {
	funcName := "admin.online_controller.GetOnlineCount"
	ctx := c.Request.Context()

	count, err := ctl.onlineManageService.GetOnlineCount(ctx)
	if err != nil {
		logs.Error(ctx, funcName, "获取在线用户数失败", zap.Error(err))
		utils.ResponseError(c, "获取在线用户数失败")
		return
	}

	logs.Info(ctx, funcName, "获取在线用户数成功", zap.Int64("count", count))
	utils.ResponseOK(c, gin.H{"count": count})
}
