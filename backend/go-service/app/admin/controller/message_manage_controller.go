// Package controller 提供管理端消息管理的 HTTP 接口处理
package controller

import (
	"strconv"

	"github.com/echochat/backend/app/admin/service"
	"github.com/echochat/backend/app/dto"
	"github.com/echochat/backend/pkg/utils"
	"github.com/gin-gonic/gin"
)

// MessageManageController 管理端消息管理控制器
type MessageManageController struct {
	msgService *service.MessageManageService
}

// NewMessageManageController 创建消息管理控制器
func NewMessageManageController(msgService *service.MessageManageService) *MessageManageController {
	return &MessageManageController{msgService: msgService}
}

// GetMessageList 获取消息列表（分页+多条件筛选）
// GET /api/v1/admin/messages
func (ctl *MessageManageController) GetMessageList(c *gin.Context) {
	ctx := c.Request.Context()

	var req dto.AdminMessageListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.ResponseBadRequest(c, "参数格式错误")
		return
	}

	result, err := ctl.msgService.GetMessageList(ctx, &req)
	if err != nil {
		ctl.handleError(c, err, "获取消息列表失败")
		return
	}

	utils.ResponseOK(c, result)
}

// GetMessageDetail 获取消息详情
// GET /api/v1/admin/messages/:id
func (ctl *MessageManageController) GetMessageDetail(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		utils.ResponseBadRequest(c, "无效的消息 ID")
		return
	}

	result, err := ctl.msgService.GetMessageDetail(ctx, id)
	if err != nil {
		ctl.handleError(c, err, "获取消息详情失败")
		return
	}

	utils.ResponseOK(c, result)
}

// DeleteMessage 删除消息（软删除）
// DELETE /api/v1/admin/messages/:id
func (ctl *MessageManageController) DeleteMessage(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		utils.ResponseBadRequest(c, "无效的消息 ID")
		return
	}

	if err := ctl.msgService.DeleteMessage(ctx, id); err != nil {
		ctl.handleError(c, err, "删除消息失败")
		return
	}

	utils.ResponseOK(c, nil)
}

// RecallMessage 管理员撤回消息
// PUT /api/v1/admin/messages/:id/recall
func (ctl *MessageManageController) RecallMessage(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		utils.ResponseBadRequest(c, "无效的消息 ID")
		return
	}

	if err := ctl.msgService.RecallMessage(ctx, id); err != nil {
		ctl.handleError(c, err, "撤回消息失败")
		return
	}

	utils.ResponseOK(c, nil)
}

// GetMessageStats 获取消息统计数据
// GET /api/v1/admin/messages/stats
func (ctl *MessageManageController) GetMessageStats(c *gin.Context) {
	ctx := c.Request.Context()

	var req dto.AdminMessageStatsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.ResponseBadRequest(c, "参数格式错误")
		return
	}

	result, err := ctl.msgService.GetStats(ctx, &req)
	if err != nil {
		ctl.handleError(c, err, "获取消息统计失败")
		return
	}

	utils.ResponseOK(c, result)
}

// handleError 统一业务错误映射
func (ctl *MessageManageController) handleError(c *gin.Context, err error, fallbackMsg ...string) {
	switch err {
	case service.ErrMessageNotFound:
		utils.ResponseNotFound(c, err.Error())
	case service.ErrMessageRecalled:
		utils.ResponseBadRequest(c, err.Error())
	case service.ErrMessageDeleted:
		utils.ResponseBadRequest(c, err.Error())
	default:
		msg := "服务器内部错误"
		if len(fallbackMsg) > 0 && fallbackMsg[0] != "" {
			msg = fallbackMsg[0]
		}
		utils.ResponseError(c, msg)
	}
}
