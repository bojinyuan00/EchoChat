// Package controller 提供 contact 模块的 HTTP 接口处理
package controller

import (
	"strconv"

	"github.com/echochat/backend/app/contact/service"
	"github.com/echochat/backend/app/dto"
	"github.com/echochat/backend/pkg/middleware"
	"github.com/echochat/backend/pkg/utils"
	"github.com/gin-gonic/gin"
)

// ContactController 联系人控制器
type ContactController struct {
	contactService *service.ContactService
}

// NewContactController 创建 ContactController 实例
func NewContactController(contactService *service.ContactService) *ContactController {
	return &ContactController{contactService: contactService}
}

// GetFriendList 获取好友列表
// GET /api/v1/contacts?group_id=xx
func (ctl *ContactController) GetFriendList(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	var groupID *int64
	if gid := c.Query("group_id"); gid != "" {
		id, err := strconv.ParseInt(gid, 10, 64)
		if err == nil {
			groupID = &id
		}
	}

	friends, err := ctl.contactService.GetFriendList(ctx, userID, groupID)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.ResponseOK(c, friends)
}

// SendFriendRequest 发送好友申请
// POST /api/v1/contacts/request
func (ctl *ContactController) SendFriendRequest(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	var req dto.SendFriendRequestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseBadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	err := ctl.contactService.SendFriendRequest(ctx, userID, req.TargetID, req.Message)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.ResponseOK(c, nil)
}

// AcceptFriendRequest 接受好友申请
// POST /api/v1/contacts/accept
func (ctl *ContactController) AcceptFriendRequest(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	var req dto.HandleFriendRequestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseBadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	err := ctl.contactService.AcceptFriendRequest(ctx, req.RequestID, userID)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.ResponseOK(c, nil)
}

// RejectFriendRequest 拒绝好友申请
// POST /api/v1/contacts/reject
func (ctl *ContactController) RejectFriendRequest(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	var req dto.HandleFriendRequestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseBadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	err := ctl.contactService.RejectFriendRequest(ctx, req.RequestID, userID)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.ResponseOK(c, nil)
}

// GetPendingRequests 获取待处理的好友申请列表
// GET /api/v1/contacts/requests
func (ctl *ContactController) GetPendingRequests(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	requests, err := ctl.contactService.GetPendingRequests(ctx, userID)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.ResponseOK(c, requests)
}

// DeleteFriend 删除好友
// DELETE /api/v1/contacts/:id
func (ctl *ContactController) DeleteFriend(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	friendID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ResponseBadRequest(c, "好友 ID 格式错误")
		return
	}

	if err := ctl.contactService.DeleteFriend(ctx, userID, friendID); err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.ResponseOK(c, nil)
}

// UpdateRemark 设置好友备注
// PUT /api/v1/contacts/:id/remark
func (ctl *ContactController) UpdateRemark(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	friendID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ResponseBadRequest(c, "好友 ID 格式错误")
		return
	}

	var req dto.UpdateRemarkReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseBadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	if err := ctl.contactService.UpdateRemark(ctx, userID, friendID, req.Remark); err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.ResponseOK(c, nil)
}

// BlockUser 拉黑用户
// POST /api/v1/contacts/block
func (ctl *ContactController) BlockUser(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	var req dto.BlockUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseBadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	err := ctl.contactService.BlockUser(ctx, userID, req.TargetID)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.ResponseOK(c, nil)
}

// UnblockUser 取消拉黑
// DELETE /api/v1/contacts/block/:user_id
func (ctl *ContactController) UnblockUser(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	targetID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		utils.ResponseBadRequest(c, "用户 ID 格式错误")
		return
	}

	if err := ctl.contactService.UnblockUser(ctx, userID, targetID); err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.ResponseOK(c, nil)
}

// GetBlockList 获取黑名单列表
// GET /api/v1/contacts/block
func (ctl *ContactController) GetBlockList(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	blocked, err := ctl.contactService.GetBlockList(ctx, userID)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.ResponseOK(c, blocked)
}

// SearchUsers 搜索用户
// GET /api/v1/users/search?keyword=xx&page=1&page_size=20
func (ctl *ContactController) SearchUsers(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	keyword := c.Query("keyword")
	if keyword == "" {
		utils.ResponseBadRequest(c, "搜索关键词不能为空")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	users, total, err := ctl.contactService.SearchUsers(ctx, userID, keyword, page, pageSize)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.ResponseOK(c, gin.H{
		"list":  users,
		"total": total,
	})
}

// GetRecommendFriends 好友推荐
// GET /api/v1/contacts/recommend
func (ctl *ContactController) GetRecommendFriends(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	friends, err := ctl.contactService.GetRecommendFriends(ctx, userID)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.ResponseOK(c, friends)
}

// GetGroups 获取好友分组列表
// GET /api/v1/contacts/groups
func (ctl *ContactController) GetGroups(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	groups, err := ctl.contactService.GetGroups(ctx, userID)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.ResponseOK(c, groups)
}

// CreateGroup 创建好友分组
// POST /api/v1/contacts/groups
func (ctl *ContactController) CreateGroup(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	var req dto.CreateGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseBadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	group, err := ctl.contactService.CreateGroup(ctx, userID, req.Name)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.ResponseCreated(c, group)
}

// UpdateGroup 修改好友分组
// PUT /api/v1/contacts/groups/:id
func (ctl *ContactController) UpdateGroup(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ResponseBadRequest(c, "分组 ID 格式错误")
		return
	}

	var req dto.UpdateGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseBadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	if err := ctl.contactService.UpdateGroup(ctx, userID, groupID, req.Name, req.SortOrder); err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.ResponseOK(c, nil)
}

// DeleteGroup 删除好友分组
// DELETE /api/v1/contacts/groups/:id
func (ctl *ContactController) DeleteGroup(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ResponseBadRequest(c, "分组 ID 格式错误")
		return
	}

	if err := ctl.contactService.DeleteGroup(ctx, userID, groupID); err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.ResponseOK(c, nil)
}

// MoveToGroup 移动好友到分组
// PUT /api/v1/contacts/:id/group
func (ctl *ContactController) MoveToGroup(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	friendID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ResponseBadRequest(c, "好友 ID 格式错误")
		return
	}

	var req dto.MoveToGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseBadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	if err := ctl.contactService.MoveToGroup(ctx, userID, friendID, req.GroupID); err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.ResponseOK(c, nil)
}

// handleError 统一错误处理
func (ctl *ContactController) handleError(c *gin.Context, err error) {
	switch err {
	case service.ErrSelfRequest:
		utils.ResponseBadRequest(c, err.Error())
	case service.ErrAlreadyFriend:
		utils.ResponseBadRequest(c, err.Error())
	case service.ErrPendingExists:
		utils.ResponseBadRequest(c, err.Error())
	case service.ErrBlocked:
		utils.ResponseForbidden(c, err.Error())
	case service.ErrRequestNotFound:
		utils.ResponseNotFound(c, err.Error())
	case service.ErrFriendNotFound:
		utils.ResponseNotFound(c, err.Error())
	case service.ErrGroupNotFound:
		utils.ResponseNotFound(c, err.Error())
	default:
		utils.ResponseError(c, "服务内部错误")
	}
}
