// Package service 提供 group 模块的业务逻辑
package service

import (
	"context"
	"errors"
	"fmt"

	authModel "github.com/echochat/backend/app/auth/model"
	"github.com/echochat/backend/app/constants"
	"github.com/echochat/backend/app/dto"
	"github.com/echochat/backend/app/group/dao"
	"github.com/echochat/backend/app/group/model"
	imModel "github.com/echochat/backend/app/im/model"
	"github.com/echochat/backend/pkg/logs"
	"github.com/echochat/backend/pkg/ws"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrGroupNotFound        = errors.New("群聊不存在")
	ErrGroupDissolved       = errors.New("群聊已解散")
	ErrNotGroupMember       = errors.New("你不是该群成员")
	ErrNotGroupOwner        = errors.New("仅群主可执行此操作")
	ErrNotGroupAdmin        = errors.New("仅群主或管理员可执行此操作")
	ErrGroupFull            = errors.New("群成员已满")
	ErrAlreadyMember        = errors.New("该用户已是群成员")
	ErrCannotKickHigherRole = errors.New("不能操作同级或更高权限的成员")
	ErrOwnerCannotLeave     = errors.New("群主不能退出群聊，请先转让群主")
	ErrCannotMuteSelf       = errors.New("不能禁言自己")
	ErrAlreadyMuted         = errors.New("该成员已被禁言")
	ErrUserMuted            = errors.New("你已被禁言，无法发送消息")
	ErrGroupAllMuted        = errors.New("当前群已开启全体禁言")
	ErrPendingRequestExists = errors.New("已有待处理的入群申请")
	ErrJoinRequestNotFound  = errors.New("入群申请不存在")
)

// UserInfoProvider 获取用户信息的接口（通过接口注入，由 contact.FriendshipDAO 隐式实现）
type UserInfoProvider interface {
	GetUsersByIDs(ctx context.Context, userIDs []int64) ([]authModel.User, error)
}

// MessageWriter 写入系统消息的接口（由 im.MessageDAO 隐式实现）
type MessageWriter interface {
	Create(ctx context.Context, msg *imModel.Message) error
}

// GroupService 群聊业务服务
type GroupService struct {
	groupDAO       *dao.GroupDAO
	joinRequestDAO *dao.JoinRequestDAO
	userInfo       UserInfoProvider
	pubsub         *ws.PubSub
	msgWriter      MessageWriter
}

// NewGroupService 创建 GroupService 实例
func NewGroupService(
	groupDAO *dao.GroupDAO,
	joinRequestDAO *dao.JoinRequestDAO,
	userInfo UserInfoProvider,
	pubsub *ws.PubSub,
	msgWriter MessageWriter,
) *GroupService {
	return &GroupService{
		groupDAO:       groupDAO,
		joinRequestDAO: joinRequestDAO,
		userInfo:       userInfo,
		pubsub:         pubsub,
		msgWriter:      msgWriter,
	}
}

// CreateGroup 创建群聊
func (s *GroupService) CreateGroup(ctx context.Context, ownerID int64, req *dto.CreateGroupRequest) (*dto.GroupDTO, error) {
	funcName := "service.group_service.CreateGroup"
	logs.Info(ctx, funcName, "创建群聊",
		zap.Int64("owner_id", ownerID), zap.String("name", req.Name), zap.Int("member_count", len(req.MemberIDs)))

	group, err := s.groupDAO.CreateGroupWithMembers(ctx, ownerID, req.Name, req.Avatar, req.MemberIDs)
	if err != nil {
		logs.Error(ctx, funcName, "创建群聊失败", zap.Error(err))
		return nil, err
	}

	s.writeSystemMessage(ctx, group.ConversationID, fmt.Sprintf("%s 创建了群聊", s.getUserNickname(ctx, ownerID)))

	allMemberIDs := append([]int64{ownerID}, req.MemberIDs...)
	s.pushToMembers(ctx, allMemberIDs, 0, "group.created", map[string]interface{}{
		"group_id":        group.ID,
		"conversation_id": group.ConversationID,
		"name":            group.Name,
		"owner_id":        ownerID,
	})

	return s.toGroupDTO(group, int(len(req.MemberIDs)+1)), nil
}

// GetGroupDetail 获取群详情（需要是群成员）
func (s *GroupService) GetGroupDetail(ctx context.Context, userID, groupID int64) (*dto.GroupDTO, error) {
	funcName := "service.group_service.GetGroupDetail"
	logs.Debug(ctx, funcName, "获取群详情",
		zap.Int64("user_id", userID), zap.Int64("group_id", groupID))

	group, err := s.groupDAO.GetByID(ctx, groupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGroupNotFound
		}
		return nil, err
	}

	_, err = s.groupDAO.GetMember(ctx, group.ConversationID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotGroupMember
		}
		return nil, err
	}

	count, _ := s.groupDAO.GetMemberCount(ctx, group.ConversationID)
	return s.toGroupDTO(group, int(count)), nil
}

// UpdateGroup 更新群信息（群主或管理员）
func (s *GroupService) UpdateGroup(ctx context.Context, userID, groupID int64, req *dto.UpdateGroupRequest) error {
	funcName := "service.group_service.UpdateGroup"
	logs.Info(ctx, funcName, "更新群信息",
		zap.Int64("user_id", userID), zap.Int64("group_id", groupID))

	group, _, err := s.checkGroupAdmin(ctx, groupID, userID)
	if err != nil {
		return err
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Avatar != nil {
		updates["avatar"] = *req.Avatar
	}
	if req.Notice != nil {
		updates["notice"] = *req.Notice
	}
	if req.IsSearchable != nil {
		updates["is_searchable"] = *req.IsSearchable
	}

	if len(updates) == 0 {
		return nil
	}

	if err := s.groupDAO.UpdateGroupInfo(ctx, groupID, updates); err != nil {
		return err
	}

	if req.Notice != nil {
		s.writeSystemMessage(ctx, group.ConversationID,
			fmt.Sprintf("%s 更新了群公告", s.getUserNickname(ctx, userID)))
	}

	s.pushToGroupMembers(ctx, group.ConversationID, 0, "group.info.update", map[string]interface{}{
		"group_id":        groupID,
		"conversation_id": group.ConversationID,
		"operator_id":     userID,
		"updates":         updates,
	})

	return nil
}

// DissolveGroup 解散群聊（仅群主）
func (s *GroupService) DissolveGroup(ctx context.Context, userID, groupID int64) error {
	funcName := "service.group_service.DissolveGroup"
	logs.Info(ctx, funcName, "解散群聊",
		zap.Int64("user_id", userID), zap.Int64("group_id", groupID))

	group, err := s.getActiveGroup(ctx, groupID)
	if err != nil {
		return err
	}
	if group.OwnerID != userID {
		return ErrNotGroupOwner
	}

	memberIDs, _ := s.groupDAO.GetMemberIDs(ctx, group.ConversationID)

	if err := s.groupDAO.DissolveGroup(ctx, groupID); err != nil {
		return err
	}

	s.writeSystemMessage(ctx, group.ConversationID, "群聊已解散")

	s.pushToMembers(ctx, memberIDs, 0, "group.dissolved", map[string]interface{}{
		"group_id":        groupID,
		"conversation_id": group.ConversationID,
		"operator_id":     userID,
	})

	return nil
}

// GetMembers 获取群成员列表（需要是群成员）
func (s *GroupService) GetMembers(ctx context.Context, userID, groupID int64) ([]dto.GroupMemberDTO, error) {
	funcName := "service.group_service.GetMembers"
	logs.Debug(ctx, funcName, "获取群成员列表",
		zap.Int64("user_id", userID), zap.Int64("group_id", groupID))

	group, err := s.getActiveGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}

	_, err = s.groupDAO.GetMember(ctx, group.ConversationID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotGroupMember
		}
		return nil, err
	}

	members, err := s.groupDAO.GetMembers(ctx, group.ConversationID)
	if err != nil {
		return nil, err
	}

	userIDs := make([]int64, 0, len(members))
	for _, m := range members {
		userIDs = append(userIDs, m.UserID)
	}

	userMap := make(map[int64]*authModel.User)
	if len(userIDs) > 0 && s.userInfo != nil {
		users, uErr := s.userInfo.GetUsersByIDs(ctx, userIDs)
		if uErr != nil {
			logs.Error(ctx, funcName, "批量查询用户信息失败", zap.Error(uErr))
		} else {
			for i := range users {
				userMap[users[i].ID] = &users[i]
			}
		}
	}

	list := make([]dto.GroupMemberDTO, 0, len(members))
	for _, m := range members {
		item := dto.GroupMemberDTO{
			UserID:   m.UserID,
			Nickname: m.Nickname,
			Role:     m.Role,
			IsMuted:  m.IsMuted,
		}
		if m.JoinedAt != nil {
			item.JoinedAt = m.JoinedAt.Format("2006-01-02 15:04:05")
		}
		if user, ok := userMap[m.UserID]; ok {
			item.UserNickname = user.Nickname
			item.Avatar = user.Avatar
		}
		list = append(list, item)
	}

	return list, nil
}

// InviteMembers 邀请用户入群（群主/管理员）
func (s *GroupService) InviteMembers(ctx context.Context, userID, groupID int64, targetIDs []int64) error {
	funcName := "service.group_service.InviteMembers"
	logs.Info(ctx, funcName, "邀请入群",
		zap.Int64("user_id", userID), zap.Int64("group_id", groupID), zap.Int("count", len(targetIDs)))

	group, _, err := s.checkGroupAdmin(ctx, groupID, userID)
	if err != nil {
		return err
	}

	count, err := s.groupDAO.GetMemberCount(ctx, group.ConversationID)
	if err != nil {
		return err
	}
	if int(count)+len(targetIDs) > group.MaxMembers {
		return ErrGroupFull
	}

	addedIDs := make([]int64, 0, len(targetIDs))
	for _, uid := range targetIDs {
		existing, _ := s.groupDAO.GetMember(ctx, group.ConversationID, uid)
		if existing != nil {
			continue
		}
		if err := s.groupDAO.AddMember(ctx, group.ConversationID, uid, constants.GroupRoleNormal); err != nil {
			logs.Error(ctx, funcName, "添加成员失败", zap.Int64("target_id", uid), zap.Error(err))
			return err
		}
		addedIDs = append(addedIDs, uid)
	}

	if len(addedIDs) > 0 {
		inviterName := s.getUserNickname(ctx, userID)
		addedNames := s.getUserNicknames(ctx, addedIDs)
		s.writeSystemMessage(ctx, group.ConversationID,
			fmt.Sprintf("%s 邀请 %s 加入了群聊", inviterName, joinNames(addedNames)))

		s.pushToGroupMembers(ctx, group.ConversationID, 0, "group.member.join", map[string]interface{}{
			"group_id":        groupID,
			"conversation_id": group.ConversationID,
			"user_ids":        addedIDs,
			"operator_id":     userID,
		})
	}

	return nil
}

// KickMember 踢出群成员（群主/管理员，不能操作同级或更高权限的成员）
func (s *GroupService) KickMember(ctx context.Context, userID, groupID, targetID int64) error {
	funcName := "service.group_service.KickMember"
	logs.Info(ctx, funcName, "踢出成员",
		zap.Int64("user_id", userID), zap.Int64("group_id", groupID), zap.Int64("target_id", targetID))

	group, operatorMember, err := s.checkGroupAdmin(ctx, groupID, userID)
	if err != nil {
		return err
	}

	targetMember, err := s.groupDAO.GetMember(ctx, group.ConversationID, targetID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotGroupMember
		}
		return err
	}

	if targetMember.Role >= operatorMember.Role {
		return ErrCannotKickHigherRole
	}

	if err := s.groupDAO.RemoveMember(ctx, group.ConversationID, targetID); err != nil {
		return err
	}

	targetName := s.getUserNickname(ctx, targetID)
	s.writeSystemMessage(ctx, group.ConversationID,
		fmt.Sprintf("%s 被移出了群聊", targetName))

	s.pushToGroupMembers(ctx, group.ConversationID, 0, "group.member.kicked", map[string]interface{}{
		"group_id":        groupID,
		"conversation_id": group.ConversationID,
		"user_id":         targetID,
		"operator_id":     userID,
	})
	s.pushToUser(ctx, targetID, "group.member.kicked", map[string]interface{}{
		"group_id":        groupID,
		"conversation_id": group.ConversationID,
		"user_id":         targetID,
		"operator_id":     userID,
	})

	return nil
}

// SetMemberRole 设置/取消管理员（仅群主）
func (s *GroupService) SetMemberRole(ctx context.Context, userID, groupID, targetID int64, role int) error {
	funcName := "service.group_service.SetMemberRole"
	logs.Info(ctx, funcName, "设置成员角色",
		zap.Int64("user_id", userID), zap.Int64("group_id", groupID),
		zap.Int64("target_id", targetID), zap.Int("role", role))

	group, err := s.getActiveGroup(ctx, groupID)
	if err != nil {
		return err
	}
	if group.OwnerID != userID {
		return ErrNotGroupOwner
	}

	_, err = s.groupDAO.GetMember(ctx, group.ConversationID, targetID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotGroupMember
		}
		return err
	}

	if err := s.groupDAO.UpdateMemberRole(ctx, group.ConversationID, targetID, role); err != nil {
		return err
	}

	targetName := s.getUserNickname(ctx, targetID)
	if role == constants.GroupRoleAdmin {
		s.writeSystemMessage(ctx, group.ConversationID, fmt.Sprintf("%s 被设为管理员", targetName))
	} else {
		s.writeSystemMessage(ctx, group.ConversationID, fmt.Sprintf("%s 被取消管理员", targetName))
	}

	s.pushToGroupMembers(ctx, group.ConversationID, 0, "group.role.update", map[string]interface{}{
		"group_id":        groupID,
		"conversation_id": group.ConversationID,
		"user_id":         targetID,
		"role":            role,
		"operator_id":     userID,
	})

	return nil
}

// MuteMember 禁言/解除禁言成员（群主/管理员）
func (s *GroupService) MuteMember(ctx context.Context, userID, groupID, targetID int64, isMuted bool) error {
	funcName := "service.group_service.MuteMember"
	logs.Info(ctx, funcName, "更新禁言状态",
		zap.Int64("user_id", userID), zap.Int64("group_id", groupID),
		zap.Int64("target_id", targetID), zap.Bool("is_muted", isMuted))

	if userID == targetID {
		return ErrCannotMuteSelf
	}

	group, operatorMember, err := s.checkGroupAdmin(ctx, groupID, userID)
	if err != nil {
		return err
	}

	targetMember, err := s.groupDAO.GetMember(ctx, group.ConversationID, targetID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotGroupMember
		}
		return err
	}

	if targetMember.Role >= operatorMember.Role {
		return ErrCannotKickHigherRole
	}

	if err := s.groupDAO.UpdateMemberMuted(ctx, group.ConversationID, targetID, isMuted); err != nil {
		return err
	}

	s.pushToGroupMembers(ctx, group.ConversationID, 0, "group.mute.update", map[string]interface{}{
		"group_id":        groupID,
		"conversation_id": group.ConversationID,
		"user_id":         targetID,
		"is_muted":        isMuted,
		"operator_id":     userID,
	})

	return nil
}

// LeaveGroup 退出群聊（群主不能退出，需先转让）
func (s *GroupService) LeaveGroup(ctx context.Context, userID, groupID int64) error {
	funcName := "service.group_service.LeaveGroup"
	logs.Info(ctx, funcName, "退出群聊",
		zap.Int64("user_id", userID), zap.Int64("group_id", groupID))

	group, err := s.getActiveGroup(ctx, groupID)
	if err != nil {
		return err
	}
	if group.OwnerID == userID {
		return ErrOwnerCannotLeave
	}

	_, err = s.groupDAO.GetMember(ctx, group.ConversationID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotGroupMember
		}
		return err
	}

	if err := s.groupDAO.RemoveMember(ctx, group.ConversationID, userID); err != nil {
		return err
	}

	userName := s.getUserNickname(ctx, userID)
	s.writeSystemMessage(ctx, group.ConversationID, fmt.Sprintf("%s 退出了群聊", userName))

	s.pushToGroupMembers(ctx, group.ConversationID, 0, "group.member.leave", map[string]interface{}{
		"group_id":        groupID,
		"conversation_id": group.ConversationID,
		"user_id":         userID,
	})

	return nil
}

// TransferOwner 转让群主（仅群主）
func (s *GroupService) TransferOwner(ctx context.Context, userID, groupID, newOwnerID int64) error {
	funcName := "service.group_service.TransferOwner"
	logs.Info(ctx, funcName, "转让群主",
		zap.Int64("user_id", userID), zap.Int64("group_id", groupID), zap.Int64("new_owner", newOwnerID))

	group, err := s.getActiveGroup(ctx, groupID)
	if err != nil {
		return err
	}
	if group.OwnerID != userID {
		return ErrNotGroupOwner
	}

	_, err = s.groupDAO.GetMember(ctx, group.ConversationID, newOwnerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotGroupMember
		}
		return err
	}

	if err := s.groupDAO.TransferOwner(ctx, groupID, userID, newOwnerID, group.ConversationID); err != nil {
		return err
	}

	oldOwnerName := s.getUserNickname(ctx, userID)
	newOwnerName := s.getUserNickname(ctx, newOwnerID)
	s.writeSystemMessage(ctx, group.ConversationID,
		fmt.Sprintf("%s 将群主转让给了 %s", oldOwnerName, newOwnerName))

	s.pushToGroupMembers(ctx, group.ConversationID, 0, "group.owner.transfer", map[string]interface{}{
		"group_id":        groupID,
		"conversation_id": group.ConversationID,
		"old_owner_id":    userID,
		"new_owner_id":    newOwnerID,
	})

	return nil
}

// UpdateNickname 修改群内昵称
func (s *GroupService) UpdateNickname(ctx context.Context, userID, groupID int64, nickname string) error {
	funcName := "service.group_service.UpdateNickname"
	logs.Info(ctx, funcName, "修改群昵称",
		zap.Int64("user_id", userID), zap.Int64("group_id", groupID))

	group, err := s.getActiveGroup(ctx, groupID)
	if err != nil {
		return err
	}

	_, err = s.groupDAO.GetMember(ctx, group.ConversationID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotGroupMember
		}
		return err
	}

	return s.groupDAO.UpdateMemberNickname(ctx, group.ConversationID, userID, nickname)
}

// SubmitJoinRequest 提交入群申请
func (s *GroupService) SubmitJoinRequest(ctx context.Context, userID, groupID int64, message string) error {
	funcName := "service.group_service.SubmitJoinRequest"
	logs.Info(ctx, funcName, "提交入群申请",
		zap.Int64("user_id", userID), zap.Int64("group_id", groupID))

	group, err := s.getActiveGroup(ctx, groupID)
	if err != nil {
		return err
	}

	existing, _ := s.groupDAO.GetMember(ctx, group.ConversationID, userID)
	if existing != nil {
		return ErrAlreadyMember
	}

	pending, _ := s.joinRequestDAO.GetPendingByGroupAndUser(ctx, groupID, userID)
	if pending != nil {
		return ErrPendingRequestExists
	}

	req, err := s.joinRequestDAO.Create(ctx, groupID, userID, message)
	if err != nil {
		return err
	}

	adminIDs, _ := s.groupDAO.GetAdminIDs(ctx, group.ConversationID)
	userName := s.getUserNickname(ctx, userID)
	s.pushToMembers(ctx, adminIDs, 0, "group.join.request", map[string]interface{}{
		"group_id":      groupID,
		"request_id":    req.ID,
		"user_id":       userID,
		"user_nickname": userName,
		"message":       message,
	})

	return nil
}

// GetJoinRequests 获取入群申请列表（群主/管理员）
func (s *GroupService) GetJoinRequests(ctx context.Context, userID, groupID int64) ([]dto.JoinRequestDTO, error) {
	funcName := "service.group_service.GetJoinRequests"
	logs.Debug(ctx, funcName, "获取入群申请列表",
		zap.Int64("user_id", userID), zap.Int64("group_id", groupID))

	_, _, err := s.checkGroupAdmin(ctx, groupID, userID)
	if err != nil {
		return nil, err
	}

	requests, err := s.joinRequestDAO.GetListByGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}

	userIDs := make([]int64, 0, len(requests))
	for _, r := range requests {
		userIDs = append(userIDs, r.UserID)
	}

	userMap := make(map[int64]*authModel.User)
	if len(userIDs) > 0 && s.userInfo != nil {
		users, uErr := s.userInfo.GetUsersByIDs(ctx, userIDs)
		if uErr != nil {
			logs.Error(ctx, funcName, "批量查询用户信息失败", zap.Error(uErr))
		} else {
			for i := range users {
				userMap[users[i].ID] = &users[i]
			}
		}
	}

	list := make([]dto.JoinRequestDTO, 0, len(requests))
	for _, r := range requests {
		item := dto.JoinRequestDTO{
			ID:        r.ID,
			GroupID:   r.GroupID,
			UserID:    r.UserID,
			Message:   r.Message,
			Status:    r.Status,
			CreatedAt: r.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if user, ok := userMap[r.UserID]; ok {
			item.UserNickname = user.Nickname
			item.UserAvatar = user.Avatar
		}
		list = append(list, item)
	}

	return list, nil
}

// ReviewJoinRequest 审批入群申请（群主/管理员）
func (s *GroupService) ReviewJoinRequest(ctx context.Context, userID, groupID, requestID int64, action string) error {
	funcName := "service.group_service.ReviewJoinRequest"
	logs.Info(ctx, funcName, "审批入群申请",
		zap.Int64("user_id", userID), zap.Int64("request_id", requestID), zap.String("action", action))

	group, _, err := s.checkGroupAdmin(ctx, groupID, userID)
	if err != nil {
		return err
	}

	req, err := s.joinRequestDAO.GetByID(ctx, requestID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrJoinRequestNotFound
		}
		return err
	}

	if req.GroupID != groupID || req.Status != constants.JoinRequestStatusPending {
		return ErrJoinRequestNotFound
	}

	if action == "approve" {
		count, _ := s.groupDAO.GetMemberCount(ctx, group.ConversationID)
		if int(count) >= group.MaxMembers {
			return ErrGroupFull
		}

		if err := s.joinRequestDAO.Approve(ctx, requestID, userID); err != nil {
			return err
		}
		if err := s.groupDAO.AddMember(ctx, group.ConversationID, req.UserID, constants.GroupRoleNormal); err != nil {
			return err
		}

		newMemberName := s.getUserNickname(ctx, req.UserID)
		s.writeSystemMessage(ctx, group.ConversationID, fmt.Sprintf("%s 加入了群聊", newMemberName))

		s.pushToUser(ctx, req.UserID, "group.join.approved", map[string]interface{}{
			"group_id":        groupID,
			"conversation_id": group.ConversationID,
			"request_id":      requestID,
		})

		s.pushToGroupMembers(ctx, group.ConversationID, 0, "group.member.join", map[string]interface{}{
			"group_id":        groupID,
			"conversation_id": group.ConversationID,
			"user_ids":        []int64{req.UserID},
		})

		return nil
	}

	return s.joinRequestDAO.Reject(ctx, requestID, userID)
}

// SearchGroups 搜索公开群
func (s *GroupService) SearchGroups(ctx context.Context, req *dto.SearchGroupRequest) (*dto.SearchGroupResponse, error) {
	funcName := "service.group_service.SearchGroups"
	logs.Debug(ctx, funcName, "搜索群聊", zap.String("keyword", req.Keyword))

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	groups, total, err := s.groupDAO.SearchGroups(ctx, req.Keyword, offset, pageSize)
	if err != nil {
		return nil, err
	}

	list := make([]dto.GroupDTO, 0, len(groups))
	for _, g := range groups {
		count, _ := s.groupDAO.GetMemberCount(ctx, g.ConversationID)
		list = append(list, *s.toGroupDTO(&g, int(count)))
	}

	return &dto.SearchGroupResponse{List: list, Total: total}, nil
}

// SetAllMuted 设置/取消全体禁言（群主/管理员）
func (s *GroupService) SetAllMuted(ctx context.Context, userID, groupID int64, isMuted bool) error {
	funcName := "service.group_service.SetAllMuted"
	logs.Info(ctx, funcName, "设置全体禁言",
		zap.Int64("user_id", userID), zap.Int64("group_id", groupID), zap.Bool("is_muted", isMuted))

	group, _, err := s.checkGroupAdmin(ctx, groupID, userID)
	if err != nil {
		return err
	}

	if err := s.groupDAO.SetAllMuted(ctx, groupID, isMuted); err != nil {
		return err
	}

	operatorName := s.getUserNickname(ctx, userID)
	if isMuted {
		s.writeSystemMessage(ctx, group.ConversationID, fmt.Sprintf("%s 开启了全体禁言", operatorName))
	} else {
		s.writeSystemMessage(ctx, group.ConversationID, fmt.Sprintf("%s 关闭了全体禁言", operatorName))
	}

	s.pushToGroupMembers(ctx, group.ConversationID, 0, "group.mute.update", map[string]interface{}{
		"group_id":        groupID,
		"conversation_id": group.ConversationID,
		"is_all_muted":    isMuted,
		"operator_id":     userID,
	})

	return nil
}

// ====== 内部辅助方法 ======

// getActiveGroup 获取群聊并校验是否存在且未解散
func (s *GroupService) getActiveGroup(ctx context.Context, groupID int64) (*model.Group, error) {
	group, err := s.groupDAO.GetByID(ctx, groupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGroupNotFound
		}
		return nil, err
	}
	if group.Status == constants.GroupStatusDissolved {
		return nil, ErrGroupDissolved
	}
	return group, nil
}

// checkGroupAdmin 校验用户是群主或管理员
func (s *GroupService) checkGroupAdmin(ctx context.Context, groupID, userID int64) (*model.Group, *imMember, error) {
	group, err := s.getActiveGroup(ctx, groupID)
	if err != nil {
		return nil, nil, err
	}

	member, err := s.groupDAO.GetMember(ctx, group.ConversationID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrNotGroupMember
		}
		return nil, nil, err
	}

	if member.Role < constants.GroupRoleAdmin {
		return nil, nil, ErrNotGroupAdmin
	}

	return group, &imMember{
		UserID: member.UserID,
		Role:   member.Role,
	}, nil
}

// imMember 内部使用的成员简要信息
type imMember struct {
	UserID int64
	Role   int
}

// ====== 推送和系统消息辅助方法 ======

// pushToUser 向单个用户推送通知
func (s *GroupService) pushToUser(ctx context.Context, userID int64, event string, data interface{}) {
	if s.pubsub == nil {
		return
	}
	push := ws.NewPushMessage(event, data)
	bytes, err := ws.MarshalPush(push)
	if err != nil {
		logs.Error(ctx, "service.group_service.pushToUser", "序列化推送消息失败", zap.Error(err))
		return
	}
	if err := s.pubsub.Publish(ctx, userID, bytes); err != nil {
		logs.Error(ctx, "service.group_service.pushToUser", "推送失败",
			zap.Int64("user_id", userID), zap.Error(err))
	}
}

// pushToMembers 向指定用户列表推送通知（排除 excludeUID）
func (s *GroupService) pushToMembers(ctx context.Context, memberIDs []int64, excludeUID int64, event string, data interface{}) {
	if s.pubsub == nil || len(memberIDs) == 0 {
		return
	}
	push := ws.NewPushMessage(event, data)
	bytes, err := ws.MarshalPush(push)
	if err != nil {
		logs.Error(ctx, "service.group_service.pushToMembers", "序列化推送消息失败", zap.Error(err))
		return
	}
	for _, uid := range memberIDs {
		if uid == excludeUID {
			continue
		}
		if pErr := s.pubsub.Publish(ctx, uid, bytes); pErr != nil {
			logs.Error(ctx, "service.group_service.pushToMembers", "推送失败",
				zap.Int64("user_id", uid), zap.Error(pErr))
		}
	}
}

// pushToGroupMembers 向群所有成员推送通知（通过 conversationID 查成员）
func (s *GroupService) pushToGroupMembers(ctx context.Context, conversationID int64, excludeUID int64, event string, data interface{}) {
	memberIDs, err := s.groupDAO.GetMemberIDs(ctx, conversationID)
	if err != nil {
		logs.Error(ctx, "service.group_service.pushToGroupMembers", "获取成员列表失败", zap.Error(err))
		return
	}
	s.pushToMembers(ctx, memberIDs, excludeUID, event, data)
}

// writeSystemMessage 写入系统消息到群会话
func (s *GroupService) writeSystemMessage(ctx context.Context, conversationID int64, content string) {
	if s.msgWriter == nil {
		return
	}
	msg := &imModel.Message{
		ConversationID: conversationID,
		SenderID:       0,
		Type:           constants.MessageTypeSystem,
		Content:        content,
		Status:         constants.MessageStatusNormal,
	}
	if err := s.msgWriter.Create(ctx, msg); err != nil {
		logs.Error(ctx, "service.group_service.writeSystemMessage", "写入系统消息失败", zap.Error(err))
	}
}

// getUserNickname 获取单个用户昵称（推送文案使用，查询失败返回默认值）
func (s *GroupService) getUserNickname(ctx context.Context, userID int64) string {
	if s.userInfo == nil {
		return "用户"
	}
	users, err := s.userInfo.GetUsersByIDs(ctx, []int64{userID})
	if err != nil || len(users) == 0 {
		return "用户"
	}
	return users[0].Nickname
}

// getUserNicknames 批量获取用户昵称列表
func (s *GroupService) getUserNicknames(ctx context.Context, userIDs []int64) []string {
	if s.userInfo == nil || len(userIDs) == 0 {
		return nil
	}
	users, err := s.userInfo.GetUsersByIDs(ctx, userIDs)
	if err != nil {
		return nil
	}
	nameMap := make(map[int64]string, len(users))
	for _, u := range users {
		nameMap[u.ID] = u.Nickname
	}
	names := make([]string, 0, len(userIDs))
	for _, id := range userIDs {
		if n, ok := nameMap[id]; ok {
			names = append(names, n)
		}
	}
	return names
}

// joinNames 将名称列表用顿号连接（中文习惯）
func joinNames(names []string) string {
	if len(names) == 0 {
		return ""
	}
	result := names[0]
	for i := 1; i < len(names); i++ {
		result += "、" + names[i]
	}
	return result
}

// toGroupDTO 将 model.Group 转换为 dto.GroupDTO
func (s *GroupService) toGroupDTO(g *model.Group, memberCount int) *dto.GroupDTO {
	return &dto.GroupDTO{
		ID:             g.ID,
		ConversationID: g.ConversationID,
		Name:           g.Name,
		Avatar:         g.Avatar,
		OwnerID:        g.OwnerID,
		Notice:         g.Notice,
		MaxMembers:     g.MaxMembers,
		MemberCount:    memberCount,
		IsSearchable:   g.IsSearchable,
		IsAllMuted:     g.IsAllMuted,
		Status:         g.Status,
		CreatedAt:      g.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}
