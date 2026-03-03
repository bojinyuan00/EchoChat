// Package controller 提供 IM 模块的 HTTP 接口处理
package controller

import (
	"strconv"

	"github.com/echochat/backend/app/dto"
	"github.com/echochat/backend/app/im/service"
	"github.com/echochat/backend/pkg/middleware"
	"github.com/echochat/backend/pkg/utils"
	"github.com/gin-gonic/gin"
)

// IMController 即时通讯控制器（REST API）
type IMController struct {
	imService *service.IMService
}

// NewIMController 创建 IMController 实例
func NewIMController(imService *service.IMService) *IMController {
	return &IMController{imService: imService}
}

// GetConversations 获取会话列表
// GET /api/v1/im/conversations
func (ctl *IMController) GetConversations(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	result, err := ctl.imService.GetConversationList(ctx, userID)
	if err != nil {
		ctl.handleError(c, err, "获取会话列表失败")
		return
	}
	utils.ResponseOK(c, result)
}

// GetHistoryMessages 获取历史消息
// GET /api/v1/im/messages?conversation_id=xx&before_id=xx&limit=30
func (ctl *IMController) GetHistoryMessages(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	var req dto.HistoryMessageRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.ResponseBadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	result, err := ctl.imService.GetHistoryMessages(ctx, userID, &req)
	if err != nil {
		ctl.handleError(c, err, "获取历史消息失败")
		return
	}
	utils.ResponseOK(c, result)
}

// PinConversation 置顶/取消置顶会话
// PUT /api/v1/im/conversations/:id/pin
func (ctl *IMController) PinConversation(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	convID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ResponseBadRequest(c, "会话 ID 格式错误")
		return
	}

	var req dto.PinConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseBadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	if err := ctl.imService.PinConversation(ctx, userID, convID, req.IsPinned); err != nil {
		ctl.handleError(c, err, "更新置顶状态失败")
		return
	}
	utils.ResponseOK(c, nil)
}

// DeleteConversation 删除会话
// DELETE /api/v1/im/conversations/:id
func (ctl *IMController) DeleteConversation(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	convID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ResponseBadRequest(c, "会话 ID 格式错误")
		return
	}

	if err := ctl.imService.DeleteConversation(ctx, userID, convID); err != nil {
		ctl.handleError(c, err, "删除会话失败")
		return
	}
	utils.ResponseOK(c, nil)
}

// ClearHistory 清空聊天记录
// DELETE /api/v1/im/conversations/:id/messages
func (ctl *IMController) ClearHistory(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	convID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ResponseBadRequest(c, "会话 ID 格式错误")
		return
	}

	if err := ctl.imService.ClearHistory(ctx, userID, convID); err != nil {
		ctl.handleError(c, err, "清空聊天记录失败")
		return
	}
	utils.ResponseOK(c, nil)
}

// SearchMessages 全局消息搜索
// GET /api/v1/im/messages/search?keyword=xx&limit=50
func (ctl *IMController) SearchMessages(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	var req dto.SearchMessageRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.ResponseBadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	result, err := ctl.imService.SearchMessages(ctx, userID, &req)
	if err != nil {
		ctl.handleError(c, err, "消息搜索失败")
		return
	}
	utils.ResponseOK(c, result)
}

// GetTotalUnread 获取全局未读消息总数
// GET /api/v1/im/unread
func (ctl *IMController) GetTotalUnread(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	count, err := ctl.imService.GetTotalUnread(ctx, userID)
	if err != nil {
		ctl.handleError(c, err, "获取未读消息数失败")
		return
	}
	utils.ResponseOK(c, gin.H{"total_unread": count})
}

// handleError 统一业务错误映射
// 已知业务错误 → 返回 Service 层定义的具体提示
// 未知错误 → 返回 fallbackMsg（未传则默认"服务器内部错误"）
func (ctl *IMController) handleError(c *gin.Context, err error, fallbackMsg ...string) {
	switch err {
	case service.ErrNotFriend:
		utils.ResponseForbidden(c, err.Error())
	case service.ErrEmptyContent:
		utils.ResponseBadRequest(c, err.Error())
	case service.ErrInvalidMsgType:
		utils.ResponseBadRequest(c, err.Error())
	case service.ErrConvNotFound:
		utils.ResponseNotFound(c, err.Error())
	case service.ErrMsgNotFound:
		utils.ResponseNotFound(c, err.Error())
	case service.ErrNotSender:
		utils.ResponseForbidden(c, err.Error())
	case service.ErrRecallTimeout:
		utils.ResponseBadRequest(c, err.Error())
	case service.ErrNotMember:
		utils.ResponseForbidden(c, err.Error())
	case service.ErrDuplicateMsg:
		utils.ResponseBadRequest(c, err.Error())
	default:
		msg := "服务器内部错误"
		if len(fallbackMsg) > 0 && fallbackMsg[0] != "" {
			msg = fallbackMsg[0]
		}
		utils.ResponseError(c, msg)
	}
}
