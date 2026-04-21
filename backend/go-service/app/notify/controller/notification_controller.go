// Package controller 提供 notify 模块的 HTTP 接口
package controller

import (
	"errors"
	"strconv"

	"github.com/echochat/backend/app/dto"
	"github.com/echochat/backend/app/notify/service"
	"github.com/echochat/backend/pkg/middleware"
	"github.com/echochat/backend/pkg/utils"
	"github.com/gin-gonic/gin"
)

// NotificationController 通知控制器（REST API）
type NotificationController struct {
	notifyService *service.NotifyService
}

// NewNotificationController 创建 NotificationController 实例
func NewNotificationController(notifyService *service.NotifyService) *NotificationController {
	return &NotificationController{notifyService: notifyService}
}

// GetList 获取通知列表
// GET /api/v1/notifications?category=&is_read=&before_id=&limit=
func (ctl *NotificationController) GetList(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	var req dto.GetNotificationsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.ResponseBadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	resp, err := ctl.notifyService.GetList(ctx, userID, &req)
	if err != nil {
		utils.ResponseError(c, "获取通知列表失败")
		return
	}
	utils.ResponseOK(c, resp)
}

// GetUnreadCount 获取未读数（总数 + 按分类）
// GET /api/v1/notifications/unread-count
func (ctl *NotificationController) GetUnreadCount(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	resp, err := ctl.notifyService.GetUnreadCount(ctx, userID)
	if err != nil {
		utils.ResponseError(c, "获取未读数失败")
		return
	}
	utils.ResponseOK(c, resp)
}

// MarkRead 标记单条通知为已读
// PUT /api/v1/notifications/:id/read
func (ctl *NotificationController) MarkRead(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ResponseBadRequest(c, "通知 ID 格式错误")
		return
	}

	if err := ctl.notifyService.MarkRead(ctx, userID, id); err != nil {
		if errors.Is(err, service.ErrNotificationNotFound) {
			utils.ResponseNotFound(c, err.Error())
			return
		}
		utils.ResponseError(c, "标记已读失败")
		return
	}
	utils.ResponseOK(c, nil)
}

// MarkAllRead 一键清零未读
// PUT /api/v1/notifications/read-all?category=
func (ctl *NotificationController) MarkAllRead(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	category := c.Query("category")
	affected, err := ctl.notifyService.MarkAllRead(ctx, userID, category)
	if err != nil {
		utils.ResponseError(c, "批量标记已读失败")
		return
	}
	utils.ResponseOK(c, dto.MarkReadResponse{Affected: int(affected)})
}

// Broadcast 管理员发起全员广播
// POST /api/v1/admin/notifications/broadcast
// 需要 admin/super_admin 角色，由路由层中间件保证
func (ctl *NotificationController) Broadcast(c *gin.Context) {
	ctx := c.Request.Context()
	operatorID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	var req dto.BroadcastNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseBadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	affected, err := ctl.notifyService.Broadcast(ctx, operatorID, &req)
	if err != nil {
		utils.ResponseError(c, "广播失败")
		return
	}
	utils.ResponseOK(c, dto.BroadcastNotificationResponse{Affected: affected})
}
