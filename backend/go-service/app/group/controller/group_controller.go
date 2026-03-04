// Package controller 提供 group 模块的 HTTP 接口处理
package controller

import (
	"strconv"

	"github.com/echochat/backend/app/dto"
	"github.com/echochat/backend/app/group/service"
	"github.com/echochat/backend/pkg/middleware"
	"github.com/echochat/backend/pkg/utils"
	"github.com/gin-gonic/gin"
)

// GroupController 群聊管理控制器（REST API）
type GroupController struct {
	groupService *service.GroupService
}

// NewGroupController 创建 GroupController 实例
func NewGroupController(groupService *service.GroupService) *GroupController {
	return &GroupController{groupService: groupService}
}

// CreateGroup 创建群聊
// POST /api/v1/groups
func (ctl *GroupController) CreateGroup(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	var req dto.CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseBadRequest(c, "参数校验失败: "+err.Error())
		return
	}

	result, err := ctl.groupService.CreateGroup(ctx, userID, &req)
	if err != nil {
		ctl.handleError(c, err, "创建群聊失败")
		return
	}
	utils.ResponseOK(c, result)
}

// GetGroupDetail 获取群详情
// GET /api/v1/groups/:id
func (ctl *GroupController) GetGroupDetail(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ResponseBadRequest(c, "群 ID 格式错误")
		return
	}

	result, err := ctl.groupService.GetGroupDetail(ctx, userID, groupID)
	if err != nil {
		ctl.handleError(c, err, "获取群详情失败")
		return
	}
	utils.ResponseOK(c, result)
}

// UpdateGroup 更新群信息
// PUT /api/v1/groups/:id
func (ctl *GroupController) UpdateGroup(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ResponseBadRequest(c, "群 ID 格式错误")
		return
	}

	var req dto.UpdateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseBadRequest(c, "参数校验失败: "+err.Error())
		return
	}

	if err := ctl.groupService.UpdateGroup(ctx, userID, groupID, &req); err != nil {
		ctl.handleError(c, err, "更新群信息失败")
		return
	}
	utils.ResponseOK(c, nil)
}

// DissolveGroup 解散群聊
// DELETE /api/v1/groups/:id
func (ctl *GroupController) DissolveGroup(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ResponseBadRequest(c, "群 ID 格式错误")
		return
	}

	if err := ctl.groupService.DissolveGroup(ctx, userID, groupID); err != nil {
		ctl.handleError(c, err, "解散群聊失败")
		return
	}
	utils.ResponseOK(c, nil)
}

// GetMembers 获取群成员列表
// GET /api/v1/groups/:id/members
func (ctl *GroupController) GetMembers(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ResponseBadRequest(c, "群 ID 格式错误")
		return
	}

	result, err := ctl.groupService.GetMembers(ctx, userID, groupID)
	if err != nil {
		ctl.handleError(c, err, "获取成员列表失败")
		return
	}
	utils.ResponseOK(c, result)
}

// InviteMembers 邀请入群
// POST /api/v1/groups/:id/members
func (ctl *GroupController) InviteMembers(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ResponseBadRequest(c, "群 ID 格式错误")
		return
	}

	var req dto.InviteMembersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseBadRequest(c, "参数校验失败: "+err.Error())
		return
	}

	if err := ctl.groupService.InviteMembers(ctx, userID, groupID, req.UserIDs); err != nil {
		ctl.handleError(c, err, "邀请入群失败")
		return
	}
	utils.ResponseOK(c, nil)
}

// KickMember 踢出成员
// DELETE /api/v1/groups/:id/members/:uid
func (ctl *GroupController) KickMember(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ResponseBadRequest(c, "群 ID 格式错误")
		return
	}

	targetID, err := strconv.ParseInt(c.Param("uid"), 10, 64)
	if err != nil {
		utils.ResponseBadRequest(c, "用户 ID 格式错误")
		return
	}

	if err := ctl.groupService.KickMember(ctx, userID, groupID, targetID); err != nil {
		ctl.handleError(c, err, "踢出成员失败")
		return
	}
	utils.ResponseOK(c, nil)
}

// SetMemberRole 设置/取消管理员
// PUT /api/v1/groups/:id/members/:uid/role
func (ctl *GroupController) SetMemberRole(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ResponseBadRequest(c, "群 ID 格式错误")
		return
	}

	targetID, err := strconv.ParseInt(c.Param("uid"), 10, 64)
	if err != nil {
		utils.ResponseBadRequest(c, "用户 ID 格式错误")
		return
	}

	var req dto.SetRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseBadRequest(c, "参数校验失败: "+err.Error())
		return
	}

	if err := ctl.groupService.SetMemberRole(ctx, userID, groupID, targetID, req.Role); err != nil {
		ctl.handleError(c, err, "设置角色失败")
		return
	}
	utils.ResponseOK(c, nil)
}

// MuteMember 禁言/解除禁言
// PUT /api/v1/groups/:id/members/:uid/mute
func (ctl *GroupController) MuteMember(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ResponseBadRequest(c, "群 ID 格式错误")
		return
	}

	targetID, err := strconv.ParseInt(c.Param("uid"), 10, 64)
	if err != nil {
		utils.ResponseBadRequest(c, "用户 ID 格式错误")
		return
	}

	var req dto.MuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseBadRequest(c, "参数校验失败: "+err.Error())
		return
	}

	if err := ctl.groupService.MuteMember(ctx, userID, groupID, targetID, req.IsMuted); err != nil {
		ctl.handleError(c, err, "操作失败")
		return
	}
	utils.ResponseOK(c, nil)
}

// TransferOwner 转让群主
// PUT /api/v1/groups/:id/transfer
func (ctl *GroupController) TransferOwner(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ResponseBadRequest(c, "群 ID 格式错误")
		return
	}

	var req dto.TransferOwnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseBadRequest(c, "参数校验失败: "+err.Error())
		return
	}

	if err := ctl.groupService.TransferOwner(ctx, userID, groupID, req.NewOwnerID); err != nil {
		ctl.handleError(c, err, "转让群主失败")
		return
	}
	utils.ResponseOK(c, nil)
}

// LeaveGroup 退出群聊
// DELETE /api/v1/groups/:id/members/me
func (ctl *GroupController) LeaveGroup(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ResponseBadRequest(c, "群 ID 格式错误")
		return
	}

	if err := ctl.groupService.LeaveGroup(ctx, userID, groupID); err != nil {
		ctl.handleError(c, err, "退出群聊失败")
		return
	}
	utils.ResponseOK(c, nil)
}

// UpdateNickname 修改群内昵称
// PUT /api/v1/groups/:id/members/me/nickname
func (ctl *GroupController) UpdateNickname(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ResponseBadRequest(c, "群 ID 格式错误")
		return
	}

	var req dto.UpdateNicknameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseBadRequest(c, "参数校验失败: "+err.Error())
		return
	}

	if err := ctl.groupService.UpdateNickname(ctx, userID, groupID, req.Nickname); err != nil {
		ctl.handleError(c, err, "修改昵称失败")
		return
	}
	utils.ResponseOK(c, nil)
}

// SubmitJoinRequest 申请入群
// POST /api/v1/groups/:id/join-requests
func (ctl *GroupController) SubmitJoinRequest(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ResponseBadRequest(c, "群 ID 格式错误")
		return
	}

	var req dto.JoinGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseBadRequest(c, "参数校验失败: "+err.Error())
		return
	}

	if err := ctl.groupService.SubmitJoinRequest(ctx, userID, groupID, req.Message); err != nil {
		ctl.handleError(c, err, "申请入群失败")
		return
	}
	utils.ResponseOK(c, nil)
}

// GetJoinRequests 获取入群申请列表
// GET /api/v1/groups/:id/join-requests
func (ctl *GroupController) GetJoinRequests(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ResponseBadRequest(c, "群 ID 格式错误")
		return
	}

	result, err := ctl.groupService.GetJoinRequests(ctx, userID, groupID)
	if err != nil {
		ctl.handleError(c, err, "获取入群申请失败")
		return
	}
	utils.ResponseOK(c, result)
}

// ReviewJoinRequest 审批入群申请
// PUT /api/v1/groups/:id/join-requests/:rid
func (ctl *GroupController) ReviewJoinRequest(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ResponseBadRequest(c, "群 ID 格式错误")
		return
	}

	requestID, err := strconv.ParseInt(c.Param("rid"), 10, 64)
	if err != nil {
		utils.ResponseBadRequest(c, "申请 ID 格式错误")
		return
	}

	var req dto.ReviewJoinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseBadRequest(c, "参数校验失败: "+err.Error())
		return
	}

	if err := ctl.groupService.ReviewJoinRequest(ctx, userID, groupID, requestID, req.Action); err != nil {
		ctl.handleError(c, err, "审批失败")
		return
	}
	utils.ResponseOK(c, nil)
}

// SearchGroups 搜索群聊
// GET /api/v1/groups/search?keyword=xx&page=1&page_size=20
func (ctl *GroupController) SearchGroups(c *gin.Context) {
	ctx := c.Request.Context()
	_, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	var req dto.SearchGroupRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.ResponseBadRequest(c, "参数校验失败: "+err.Error())
		return
	}

	result, err := ctl.groupService.SearchGroups(ctx, &req)
	if err != nil {
		ctl.handleError(c, err, "搜索群聊失败")
		return
	}
	utils.ResponseOK(c, result)
}

// SetAllMuted 设置全体禁言
// PUT /api/v1/groups/:id/all-mute
func (ctl *GroupController) SetAllMuted(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ResponseBadRequest(c, "群 ID 格式错误")
		return
	}

	var body struct {
		IsAllMuted bool `json:"is_all_muted"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ResponseBadRequest(c, "参数校验失败: "+err.Error())
		return
	}

	if err := ctl.groupService.SetAllMuted(ctx, userID, groupID, body.IsAllMuted); err != nil {
		ctl.handleError(c, err, "设置全体禁言失败")
		return
	}
	utils.ResponseOK(c, nil)
}

// handleError 统一业务错误映射
// 已知业务错误 → 返回 Service 层定义的具体提示
// 未知错误 → 返回 fallbackMsg（未传则默认"服务器内部错误"）
func (ctl *GroupController) handleError(c *gin.Context, err error, fallbackMsg ...string) {
	switch err {
	case service.ErrGroupNotFound:
		utils.ResponseNotFound(c, err.Error())
	case service.ErrGroupDissolved:
		utils.ResponseBadRequest(c, err.Error())
	case service.ErrNotGroupMember:
		utils.ResponseForbidden(c, err.Error())
	case service.ErrNotGroupOwner:
		utils.ResponseForbidden(c, err.Error())
	case service.ErrNotGroupAdmin:
		utils.ResponseForbidden(c, err.Error())
	case service.ErrGroupFull:
		utils.ResponseBadRequest(c, err.Error())
	case service.ErrAlreadyMember:
		utils.ResponseBadRequest(c, err.Error())
	case service.ErrCannotKickHigherRole:
		utils.ResponseForbidden(c, err.Error())
	case service.ErrOwnerCannotLeave:
		utils.ResponseBadRequest(c, err.Error())
	case service.ErrCannotMuteSelf:
		utils.ResponseBadRequest(c, err.Error())
	case service.ErrAlreadyMuted:
		utils.ResponseBadRequest(c, err.Error())
	case service.ErrUserMuted:
		utils.ResponseForbidden(c, err.Error())
	case service.ErrGroupAllMuted:
		utils.ResponseForbidden(c, err.Error())
	case service.ErrPendingRequestExists:
		utils.ResponseBadRequest(c, err.Error())
	case service.ErrJoinRequestNotFound:
		utils.ResponseNotFound(c, err.Error())
	default:
		msg := "服务器内部错误"
		if len(fallbackMsg) > 0 && fallbackMsg[0] != "" {
			msg = fallbackMsg[0]
		}
		utils.ResponseError(c, msg)
	}
}
