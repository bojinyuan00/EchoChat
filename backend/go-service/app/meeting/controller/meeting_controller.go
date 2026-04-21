// Package controller 提供 meeting 模块的 HTTP 接口
package controller

import (
	"errors"
	"strconv"

	"github.com/echochat/backend/app/constants"
	"github.com/echochat/backend/app/dto"
	"github.com/echochat/backend/app/meeting/model"
	"github.com/echochat/backend/app/meeting/service"
	"github.com/echochat/backend/pkg/logs"
	"github.com/echochat/backend/pkg/middleware"
	"github.com/echochat/backend/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// MeetingController 会议 REST 控制器
// Task 5 完成：填充 12 个接口的请求解析、业务调用、DTO 转换、错误码映射
type MeetingController struct {
	meetingService *service.MeetingService
}

// NewMeetingController 创建 MeetingController 实例
func NewMeetingController(meetingService *service.MeetingService) *MeetingController {
	return &MeetingController{meetingService: meetingService}
}

// requireUserID 统一的当前用户取值，失败直接写 401 并返回 false
func requireUserID(c *gin.Context) (int64, bool) {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return 0, false
	}
	return userID, true
}

// handleError 将 service 层领域错误统一映射为 HTTP 响应
// 映射原则：
//   - 资源不存在 → 404
//   - 权限不足（非 host）→ 403
//   - 业务规则违反（密码错误、已满、状态冲突等）→ 400
//   - 其余未知错误 → 500
func (ctl *MeetingController) handleError(c *gin.Context, err error, fallbackMsg string) {
	switch {
	case errors.Is(err, service.ErrMeetingNotFound):
		utils.ResponseNotFound(c, err.Error())
	case errors.Is(err, service.ErrNotMeetingHost):
		utils.ResponseForbidden(c, err.Error())
	case errors.Is(err, service.ErrMeetingEnded),
		errors.Is(err, service.ErrMeetingFull),
		errors.Is(err, service.ErrMeetingPasswordReq),
		errors.Is(err, service.ErrMeetingPasswordWrong),
		errors.Is(err, service.ErrMeetingPasswordLocked),
		errors.Is(err, service.ErrNotInMeeting),
		errors.Is(err, service.ErrAlreadyInMeeting),
		errors.Is(err, service.ErrAlreadyInOtherMeeting),
		errors.Is(err, service.ErrInviteTokenInvalid),
		errors.Is(err, service.ErrRoomCodeConflict),
		errors.Is(err, service.ErrKickSelfForbidden),
		errors.Is(err, service.ErrTransferToSelf),
		errors.Is(err, service.ErrTransferTargetInvalid):
		utils.ResponseBadRequest(c, err.Error())
	default:
		logs.Warn(c.Request.Context(), "controller.meeting_controller.handleError",
			fallbackMsg, zap.Error(err))
		utils.ResponseError(c, fallbackMsg)
	}
}

// ====== DTO 转换 ======

// roomToDTO 将 model.MeetingRoom 转 DTO，host_name/avatar 交由上层按需补全
func roomToDTO(r *model.MeetingRoom, onlineCount int) *dto.MeetingRoomDTO {
	if r == nil {
		return nil
	}
	out := &dto.MeetingRoomDTO{
		ID:          r.ID,
		RoomCode:    r.RoomCode,
		Title:       r.Title,
		HostID:      r.HostID,
		Type:        r.Type,
		HasPassword: r.PasswordHash != nil && *r.PasswordHash != "",
		MaxMembers:  r.MaxMembers,
		Status:      r.Status,
		StatusLabel: constants.MeetingStatusMap[r.Status],
		Settings:    r.Settings,
		CreatedAt:   r.CreatedAt.Format("2006-01-02 15:04:05"),
		OnlineCount: onlineCount,
		EndedReason: r.EndedReason,
	}
	if r.ScheduledAt != nil {
		out.ScheduledAt = r.ScheduledAt.Format("2006-01-02 15:04:05")
	}
	if r.StartedAt != nil {
		out.StartedAt = r.StartedAt.Format("2006-01-02 15:04:05")
	}
	if r.EndedAt != nil {
		out.EndedAt = r.EndedAt.Format("2006-01-02 15:04:05")
	}
	return out
}

// participantToDTO 将 model.MeetingParticipant 转 DTO
func participantToDTO(p *model.MeetingParticipant) *dto.MeetingParticipantDTO {
	if p == nil {
		return nil
	}
	out := &dto.MeetingParticipantDTO{
		ID:         p.ID,
		RoomID:     p.RoomID,
		UserID:     p.UserID,
		Role:       p.Role,
		RoleLabel:  constants.MeetingRoleMap[p.Role],
		JoinedAt:   p.JoinedAt.Format("2006-01-02 15:04:05"),
		LeftReason: p.LeftReason,
		Duration:   p.Duration,
		IsActive:   p.IsActive(),
	}
	if p.LeftAt != nil {
		out.LeftAt = p.LeftAt.Format("2006-01-02 15:04:05")
	}
	return out
}

// chatToDTO 将 model.MeetingChat 转 DTO
func chatToDTO(m *model.MeetingChat) *dto.MeetingChatDTO {
	if m == nil {
		return nil
	}
	return &dto.MeetingChatDTO{
		ID:        m.ID,
		RoomID:    m.RoomID,
		UserID:    m.UserID,
		Content:   m.Content,
		CreatedAt: m.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

// ====== REST API ======

// CreateRoom POST /api/v1/meeting/rooms
func (ctl *MeetingController) CreateRoom(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	var req dto.CreateMeetingRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseBadRequest(c, "参数校验失败: "+err.Error())
		return
	}
	room, _, routerID, err := ctl.meetingService.CreateRoom(c.Request.Context(), userID, &req)
	if err != nil {
		ctl.handleError(c, err, "创建会议失败")
		return
	}
	resp := dto.CreateMeetingRoomResponse{
		Room: *roomToDTO(room, 1),
	}
	_ = routerID // 当前 Noop 返回占位 RouterID，Task 7 后可拼入响应供前端订阅使用
	utils.ResponseCreated(c, resp)
}

// GetRoom GET /api/v1/meeting/rooms/:code
func (ctl *MeetingController) GetRoom(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	code := c.Param("code")
	if code == "" {
		utils.ResponseBadRequest(c, "会议号不能为空")
		return
	}
	room, participants, onlineCount, err := ctl.meetingService.GetRoomByCode(c.Request.Context(), userID, code)
	if err != nil {
		ctl.handleError(c, err, "获取会议详情失败")
		return
	}
	parts := make([]dto.MeetingParticipantDTO, 0, len(participants))
	for i := range participants {
		parts = append(parts, *participantToDTO(&participants[i]))
	}
	resp := dto.GetMeetingRoomResponse{
		Room:         *roomToDTO(room, int(onlineCount)),
		Participants: parts,
		OnlineCount:  int(onlineCount),
	}
	utils.ResponseOK(c, resp)
}

// JoinRoom POST /api/v1/meeting/rooms/:code/join
func (ctl *MeetingController) JoinRoom(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	code := c.Param("code")
	if code == "" {
		utils.ResponseBadRequest(c, "会议号不能为空")
		return
	}
	var req dto.JoinMeetingRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		utils.ResponseBadRequest(c, "参数校验失败: "+err.Error())
		return
	}
	room, participant, routerID, err := ctl.meetingService.JoinRoom(c.Request.Context(), userID, code, req.Password)
	if err != nil {
		ctl.handleError(c, err, "加入会议失败")
		return
	}
	resp := dto.JoinMeetingRoomResponse{
		Room:        *roomToDTO(room, 0),
		Participant: *participantToDTO(participant),
		RouterID:    routerID,
	}
	utils.ResponseOK(c, resp)
}

// LeaveRoom POST /api/v1/meeting/rooms/:code/leave
func (ctl *MeetingController) LeaveRoom(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	code := c.Param("code")
	if code == "" {
		utils.ResponseBadRequest(c, "会议号不能为空")
		return
	}
	duration, err := ctl.meetingService.LeaveRoom(c.Request.Context(), userID, code)
	if err != nil {
		ctl.handleError(c, err, "离开会议失败")
		return
	}
	utils.ResponseOK(c, dto.LeaveMeetingRoomResponse{Duration: duration})
}

// EndRoom POST /api/v1/meeting/rooms/:code/end
func (ctl *MeetingController) EndRoom(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	code := c.Param("code")
	if code == "" {
		utils.ResponseBadRequest(c, "会议号不能为空")
		return
	}
	if err := ctl.meetingService.EndRoom(c.Request.Context(), userID, code); err != nil {
		ctl.handleError(c, err, "结束会议失败")
		return
	}
	utils.ResponseOK(c, gin.H{})
}

// ListMyMeetings GET /api/v1/meeting/rooms/mine?status=&before_id=&limit=
func (ctl *MeetingController) ListMyMeetings(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	var req dto.ListMyMeetingsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.ResponseBadRequest(c, "参数校验失败: "+err.Error())
		return
	}
	rooms, hasMore, err := ctl.meetingService.ListMyMeetings(c.Request.Context(), userID, req.Status, req.BeforeID, req.Limit)
	if err != nil {
		ctl.handleError(c, err, "获取会议列表失败")
		return
	}
	list := make([]dto.MeetingRoomDTO, 0, len(rooms))
	for i := range rooms {
		list = append(list, *roomToDTO(&rooms[i], 0))
	}
	utils.ResponseOK(c, dto.ListMyMeetingsResponse{List: list, HasMore: hasMore})
}

// TransferHost POST /api/v1/meeting/rooms/:code/transfer-host
func (ctl *MeetingController) TransferHost(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	code := c.Param("code")
	if code == "" {
		utils.ResponseBadRequest(c, "会议号不能为空")
		return
	}
	var req dto.TransferHostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseBadRequest(c, "参数校验失败: "+err.Error())
		return
	}
	if err := ctl.meetingService.TransferHost(c.Request.Context(), userID, code, req.TargetUserID); err != nil {
		ctl.handleError(c, err, "转让主持人失败")
		return
	}
	utils.ResponseOK(c, gin.H{})
}

// KickMember POST /api/v1/meeting/rooms/:code/kick
func (ctl *MeetingController) KickMember(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	code := c.Param("code")
	if code == "" {
		utils.ResponseBadRequest(c, "会议号不能为空")
		return
	}
	var req dto.KickMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseBadRequest(c, "参数校验失败: "+err.Error())
		return
	}
	if err := ctl.meetingService.KickMember(c.Request.Context(), userID, code, req.UserID); err != nil {
		ctl.handleError(c, err, "踢出成员失败")
		return
	}
	utils.ResponseOK(c, gin.H{})
}

// InviteUsers POST /api/v1/meeting/rooms/:code/invite
func (ctl *MeetingController) InviteUsers(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	code := c.Param("code")
	if code == "" {
		utils.ResponseBadRequest(c, "会议号不能为空")
		return
	}
	var req dto.InviteUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseBadRequest(c, "参数校验失败: "+err.Error())
		return
	}
	pushed, skipped, err := ctl.meetingService.InviteUsers(c.Request.Context(), userID, code, req.InviteeIDs)
	if err != nil {
		ctl.handleError(c, err, "发送邀请失败")
		return
	}
	utils.ResponseOK(c, dto.InviteUsersResponse{Pushed: pushed, Skipped: skipped})
}

// RedeemInvite POST /api/v1/meeting/invite-tokens/:token/redeem
func (ctl *MeetingController) RedeemInvite(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	token := c.Param("token")
	if token == "" {
		utils.ResponseBadRequest(c, "邀请 Token 不能为空")
		return
	}
	resp, err := ctl.meetingService.RedeemInviteToken(c.Request.Context(), userID, token)
	if err != nil {
		ctl.handleError(c, err, "兑换邀请链接失败")
		return
	}
	utils.ResponseOK(c, resp)
}

// SendChat POST /api/v1/meeting/rooms/:code/chats
func (ctl *MeetingController) SendChat(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	code := c.Param("code")
	if code == "" {
		utils.ResponseBadRequest(c, "会议号不能为空")
		return
	}
	var req dto.SendMeetingChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseBadRequest(c, "参数校验失败: "+err.Error())
		return
	}
	msg, err := ctl.meetingService.SendChatMessage(c.Request.Context(), userID, code, req.Content)
	if err != nil {
		ctl.handleError(c, err, "发送会议聊天失败")
		return
	}
	utils.ResponseCreated(c, dto.SendMeetingChatResponse{Message: *chatToDTO(msg)})
}

// ListChats GET /api/v1/meeting/rooms/:code/chats?before_id=&limit=
func (ctl *MeetingController) ListChats(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	code := c.Param("code")
	if code == "" {
		utils.ResponseBadRequest(c, "会议号不能为空")
		return
	}
	beforeID, _ := strconv.ParseInt(c.Query("before_id"), 10, 64)
	limit, _ := strconv.Atoi(c.Query("limit"))
	msgs, hasMore, err := ctl.meetingService.ListChatMessages(c.Request.Context(), userID, code, beforeID, limit)
	if err != nil {
		ctl.handleError(c, err, "获取会议聊天失败")
		return
	}
	list := make([]dto.MeetingChatDTO, 0, len(msgs))
	for i := range msgs {
		list = append(list, *chatToDTO(&msgs[i]))
	}
	utils.ResponseOK(c, dto.ListMeetingChatsResponse{List: list, HasMore: hasMore})
}
