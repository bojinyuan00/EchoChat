// Package service 提供 contact 模块的业务逻辑
package service

import (
	"context"
	"errors"
	"sort"

	"github.com/echochat/backend/app/contact/dao"
	"github.com/echochat/backend/app/dto"
	"github.com/echochat/backend/pkg/logs"
	"github.com/echochat/backend/pkg/ws"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrSelfRequest     = errors.New("不能添加自己为好友")
	ErrAlreadyFriend   = errors.New("已经是好友了")
	ErrPendingExists   = errors.New("已有待处理的好友申请")
	ErrBlocked         = errors.New("对方已将你拉黑")
	ErrRequestNotFound = errors.New("好友申请不存在")
	ErrFriendNotFound  = errors.New("好友关系不存在")
	ErrGroupNotFound   = errors.New("分组不存在")
	ErrUserNotFound    = errors.New("用户不存在")
)

// ContactService 联系人业务服务
type ContactService struct {
	friendshipDAO  *dao.FriendshipDAO
	friendGroupDAO *dao.FriendGroupDAO
	pubsub         *ws.PubSub
}

// NewContactService 创建 ContactService 实例
func NewContactService(
	friendshipDAO *dao.FriendshipDAO,
	friendGroupDAO *dao.FriendGroupDAO,
	pubsub *ws.PubSub,
) *ContactService {
	return &ContactService{
		friendshipDAO:  friendshipDAO,
		friendGroupDAO: friendGroupDAO,
		pubsub:         pubsub,
	}
}

// SendFriendRequest 发送好友申请
func (s *ContactService) SendFriendRequest(ctx context.Context, userID, targetID int64, message string) error {
	funcName := "service.contact_service.SendFriendRequest"
	logs.Info(ctx, funcName, "发送好友申请",
		zap.Int64("user_id", userID), zap.Int64("target_id", targetID))

	if userID == targetID {
		return ErrSelfRequest
	}

	blocked, err := s.friendshipDAO.IsBlocked(ctx, userID, targetID)
	if err != nil {
		logs.Error(ctx, funcName, "检查拉黑状态失败", zap.Error(err))
		return err
	}
	if blocked {
		return ErrBlocked
	}

	isFriend, err := s.friendshipDAO.IsFriend(ctx, userID, targetID)
	if err != nil {
		return err
	}
	if isFriend {
		return ErrAlreadyFriend
	}

	pending, err := s.friendshipDAO.HasPendingRequest(ctx, userID, targetID)
	if err != nil {
		return err
	}
	if pending {
		return ErrPendingExists
	}

	_, err = s.friendshipDAO.CreateRequest(ctx, userID, targetID, message)
	if err != nil {
		return err
	}

	push := ws.NewPushMessage("notify.friend.request", map[string]interface{}{
		"from_user_id": userID,
		"message":      message,
	})
	if pubErr := s.pubsub.PublishToUser(ctx, targetID, push); pubErr != nil {
		logs.Warn(ctx, funcName, "推送好友申请通知失败", zap.Error(pubErr))
	}

	return nil
}

// AcceptFriendRequest 接受好友申请
func (s *ContactService) AcceptFriendRequest(ctx context.Context, requestID, userID int64) error {
	funcName := "service.contact_service.AcceptFriendRequest"
	logs.Info(ctx, funcName, "接受好友申请",
		zap.Int64("request_id", requestID), zap.Int64("user_id", userID))

	req, err := s.friendshipDAO.GetRequestByID(ctx, requestID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRequestNotFound
		}
		return err
	}

	if req.FriendID != userID {
		return ErrRequestNotFound
	}

	if err := s.friendshipDAO.AcceptRequest(ctx, requestID, userID); err != nil {
		return err
	}

	push := ws.NewPushMessage("contact.request.accepted", map[string]interface{}{
		"user_id": userID,
	})
	if pubErr := s.pubsub.PublishToUser(ctx, req.UserID, push); pubErr != nil {
		logs.Warn(ctx, funcName, "推送申请接受通知失败", zap.Error(pubErr))
	}

	return nil
}

// RejectFriendRequest 拒绝好友申请
func (s *ContactService) RejectFriendRequest(ctx context.Context, requestID, userID int64) error {
	funcName := "service.contact_service.RejectFriendRequest"
	logs.Info(ctx, funcName, "拒绝好友申请",
		zap.Int64("request_id", requestID), zap.Int64("user_id", userID))

	err := s.friendshipDAO.RejectRequest(ctx, requestID, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrRequestNotFound
	}
	return err
}

// GetFriendList 获取好友列表
func (s *ContactService) GetFriendList(ctx context.Context, userID int64, groupID *int64) ([]dto.FriendInfo, error) {
	funcName := "service.contact_service.GetFriendList"
	logs.Debug(ctx, funcName, "获取好友列表", zap.Int64("user_id", userID))

	friends, err := s.friendshipDAO.GetFriendList(ctx, userID, groupID)
	if err != nil {
		return nil, err
	}

	result := make([]dto.FriendInfo, 0, len(friends))
	for _, f := range friends {
		result = append(result, dto.FriendInfo{
			ID:        f.ID,
			UserID:    f.UserID,
			Username:  f.Username,
			Nickname:  f.Nickname,
			Avatar:    f.Avatar,
			Remark:    f.Remark,
			GroupID:   f.GroupID,
			CreatedAt: f.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return result, nil
}

// GetPendingRequests 获取待处理的好友申请
func (s *ContactService) GetPendingRequests(ctx context.Context, userID int64) ([]dto.FriendRequestInfo, error) {
	funcName := "service.contact_service.GetPendingRequests"
	logs.Debug(ctx, funcName, "获取待处理申请", zap.Int64("user_id", userID))

	requests, err := s.friendshipDAO.GetPendingRequests(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]dto.FriendRequestInfo, 0, len(requests))
	for _, r := range requests {
		result = append(result, dto.FriendRequestInfo{
			ID:        r.ID,
			UserID:    r.UserID,
			Username:  r.Username,
			Nickname:  r.Nickname,
			Avatar:    r.Avatar,
			Message:   r.Message,
			Status:    r.Status,
			CreatedAt: r.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return result, nil
}

// DeleteFriend 删除好友
func (s *ContactService) DeleteFriend(ctx context.Context, userID, friendID int64) error {
	funcName := "service.contact_service.DeleteFriend"
	logs.Info(ctx, funcName, "删除好友",
		zap.Int64("user_id", userID), zap.Int64("friend_id", friendID))
	return s.friendshipDAO.DeleteFriend(ctx, userID, friendID)
}

// UpdateRemark 更新好友备注
func (s *ContactService) UpdateRemark(ctx context.Context, userID, friendID int64, remark string) error {
	funcName := "service.contact_service.UpdateRemark"
	logs.Info(ctx, funcName, "更新好友备注",
		zap.Int64("user_id", userID), zap.Int64("friend_id", friendID))
	return s.friendshipDAO.UpdateRemark(ctx, userID, friendID, remark)
}

// BlockUser 拉黑用户
func (s *ContactService) BlockUser(ctx context.Context, userID, targetID int64) error {
	funcName := "service.contact_service.BlockUser"
	logs.Info(ctx, funcName, "拉黑用户",
		zap.Int64("user_id", userID), zap.Int64("target_id", targetID))

	if userID == targetID {
		return ErrSelfRequest
	}
	return s.friendshipDAO.BlockUser(ctx, userID, targetID)
}

// UnblockUser 取消拉黑
func (s *ContactService) UnblockUser(ctx context.Context, userID, targetID int64) error {
	funcName := "service.contact_service.UnblockUser"
	logs.Info(ctx, funcName, "取消拉黑",
		zap.Int64("user_id", userID), zap.Int64("target_id", targetID))
	return s.friendshipDAO.UnblockUser(ctx, userID, targetID)
}

// GetBlockList 获取黑名单
func (s *ContactService) GetBlockList(ctx context.Context, userID int64) ([]dto.FriendInfo, error) {
	funcName := "service.contact_service.GetBlockList"
	logs.Debug(ctx, funcName, "获取黑名单", zap.Int64("user_id", userID))

	blocked, err := s.friendshipDAO.GetBlockList(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]dto.FriendInfo, 0, len(blocked))
	for _, f := range blocked {
		result = append(result, dto.FriendInfo{
			ID:       f.ID,
			UserID:   f.UserID,
			Username: f.Username,
			Nickname: f.Nickname,
			Avatar:   f.Avatar,
		})
	}
	return result, nil
}

// SearchUsers 搜索用户
func (s *ContactService) SearchUsers(ctx context.Context, userID int64, keyword string, page, pageSize int) ([]dto.SearchUserInfo, int64, error) {
	funcName := "service.contact_service.SearchUsers"
	logs.Debug(ctx, funcName, "搜索用户", zap.String("keyword", keyword))

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 50 {
		pageSize = 20
	}

	users, total, err := s.friendshipDAO.SearchUsers(ctx, keyword, userID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	friendIDs, _ := s.friendshipDAO.GetFriendIDs(ctx, userID)
	friendSet := make(map[int64]bool, len(friendIDs))
	for _, id := range friendIDs {
		friendSet[id] = true
	}

	result := make([]dto.SearchUserInfo, 0, len(users))
	for _, u := range users {
		result = append(result, dto.SearchUserInfo{
			ID:       u.ID,
			Username: u.Username,
			Nickname: u.Nickname,
			Avatar:   u.Avatar,
			IsFriend: friendSet[u.ID],
		})
	}
	return result, total, nil
}

// GetRecommendFriends 好友推荐（基于共同好友数量排序）
func (s *ContactService) GetRecommendFriends(ctx context.Context, userID int64) ([]dto.SearchUserInfo, error) {
	funcName := "service.contact_service.GetRecommendFriends"
	logs.Debug(ctx, funcName, "好友推荐", zap.Int64("user_id", userID))

	friendIDs, err := s.friendshipDAO.GetFriendIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(friendIDs) == 0 {
		return []dto.SearchUserInfo{}, nil
	}

	candidateCount := make(map[int64]int)
	friendSet := make(map[int64]bool, len(friendIDs))
	for _, id := range friendIDs {
		friendSet[id] = true
	}

	for _, fid := range friendIDs {
		fofIDs, err := s.friendshipDAO.GetFriendIDs(ctx, fid)
		if err != nil {
			continue
		}
		for _, fof := range fofIDs {
			if fof != userID && !friendSet[fof] {
				candidateCount[fof]++
			}
		}
	}

	type candidate struct {
		id    int64
		count int
	}
	candidates := make([]candidate, 0, len(candidateCount))
	for id, count := range candidateCount {
		candidates = append(candidates, candidate{id: id, count: count})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].count > candidates[j].count
	})

	limit := 10
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	candidateIDs := make([]int64, len(candidates))
	for i, c := range candidates {
		candidateIDs[i] = c.id
	}

	users, err := s.friendshipDAO.GetUsersByIDs(ctx, candidateIDs)
	if err != nil {
		logs.Error(ctx, funcName, "批量查询推荐用户信息失败", zap.Error(err))
		return nil, err
	}

	userMap := make(map[int64]dto.SearchUserInfo, len(users))
	for _, u := range users {
		userMap[u.ID] = dto.SearchUserInfo{
			ID:       u.ID,
			Username: u.Username,
			Nickname: u.Nickname,
			Avatar:   u.Avatar,
			IsFriend: false,
		}
	}

	result := make([]dto.SearchUserInfo, 0, len(candidates))
	for _, c := range candidates {
		if info, ok := userMap[c.id]; ok {
			result = append(result, info)
		}
	}

	return result, nil
}

// CreateGroup 创建好友分组
func (s *ContactService) CreateGroup(ctx context.Context, userID int64, name string) (*dto.GroupInfo, error) {
	funcName := "service.contact_service.CreateGroup"
	logs.Info(ctx, funcName, "创建好友分组",
		zap.Int64("user_id", userID), zap.String("name", name))

	group, err := s.friendGroupDAO.CreateGroup(ctx, userID, name)
	if err != nil {
		return nil, err
	}
	return &dto.GroupInfo{
		ID:        group.ID,
		Name:      group.Name,
		SortOrder: group.SortOrder,
	}, nil
}

// GetGroups 获取好友分组列表
func (s *ContactService) GetGroups(ctx context.Context, userID int64) ([]dto.GroupInfo, error) {
	funcName := "service.contact_service.GetGroups"
	logs.Debug(ctx, funcName, "获取好友分组", zap.Int64("user_id", userID))

	groups, err := s.friendGroupDAO.GetGroups(ctx, userID)
	if err != nil {
		return nil, err
	}

	countMap, _ := s.friendshipDAO.CountFriendsByGroup(ctx, userID)

	result := make([]dto.GroupInfo, 0, len(groups))
	for _, g := range groups {
		result = append(result, dto.GroupInfo{
			ID:          g.ID,
			Name:        g.Name,
			SortOrder:   g.SortOrder,
			FriendCount: countMap[g.ID],
		})
	}
	return result, nil
}

// UpdateGroup 更新好友分组
func (s *ContactService) UpdateGroup(ctx context.Context, userID, groupID int64, name string, sortOrder *int) error {
	return s.friendGroupDAO.UpdateGroup(ctx, groupID, userID, name, sortOrder)
}

// DeleteGroup 删除好友分组
func (s *ContactService) DeleteGroup(ctx context.Context, userID, groupID int64) error {
	return s.friendGroupDAO.DeleteGroup(ctx, groupID, userID)
}

// MoveToGroup 移动好友到分组
func (s *ContactService) MoveToGroup(ctx context.Context, userID, friendID int64, groupID *int64) error {
	return s.friendGroupDAO.MoveToGroup(ctx, userID, friendID, groupID)
}
