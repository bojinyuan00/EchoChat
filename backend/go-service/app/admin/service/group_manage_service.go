// Package service 提供 admin 模块的核心业务逻辑
package service

import (
	"context"
	"math"
	"strconv"

	groupDAO "github.com/echochat/backend/app/group/dao"
	groupModel "github.com/echochat/backend/app/group/model"
	"github.com/echochat/backend/pkg/logs"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// GroupManageService 管理端群组管理服务
type GroupManageService struct {
	db       *gorm.DB
	groupDAO *groupDAO.GroupDAO
}

// NewGroupManageService 创建群组管理服务实例
func NewGroupManageService(db *gorm.DB, groupDAO *groupDAO.GroupDAO) *GroupManageService {
	return &GroupManageService{db: db, groupDAO: groupDAO}
}

// GroupListItem 管理端群组列表项
type GroupListItem struct {
	ID             int64  `json:"id"`
	ConversationID int64  `json:"conversation_id"`
	Name           string `json:"name"`
	Avatar         string `json:"avatar"`
	OwnerID        int64  `json:"owner_id"`
	OwnerName      string `json:"owner_name"`
	MemberCount    int64  `json:"member_count"`
	MaxMembers     int    `json:"max_members"`
	Status         int    `json:"status"`
	IsAllMuted     bool   `json:"is_all_muted"`
	CreatedAt      string `json:"created_at"`
}

// GroupDetailInfo 管理端群组详情
type GroupDetailInfo struct {
	GroupListItem
	Notice     string            `json:"notice"`
	IsSearchable bool            `json:"is_searchable"`
	Members    []GroupMemberInfo `json:"members"`
}

// GroupMemberInfo 群成员信息
type GroupMemberInfo struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Role     int    `json:"role"`
	IsMuted  bool   `json:"is_muted"`
	JoinedAt string `json:"joined_at"`
}

// ListGroups 获取群组列表（分页 + 搜索）
func (s *GroupManageService) ListGroups(ctx context.Context, page, pageSize int, keyword string) ([]GroupListItem, int64, error) {
	funcName := "service.group_manage_service.ListGroups"
	logs.Info(ctx, funcName, "获取群组列表",
		zap.Int("page", page), zap.Int("page_size", pageSize), zap.String("keyword", keyword))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	var groups []groupModel.Group
	var total int64

	query := s.db.WithContext(ctx).Model(&groupModel.Group{})
	if keyword != "" {
		query = query.Where("name ILIKE ?", "%"+keyword+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&groups).Error; err != nil {
		return nil, 0, err
	}

	// 批量查询群主名称，避免 N+1
	ownerIDs := make([]int64, 0, len(groups))
	convIDs := make([]int64, 0, len(groups))
	for _, g := range groups {
		ownerIDs = append(ownerIDs, g.OwnerID)
		convIDs = append(convIDs, g.ConversationID)
	}
	ownerNameMap := s.batchGetUsernames(ctx, ownerIDs)
	memberCountMap := s.batchGetMemberCounts(ctx, convIDs)

	list := make([]GroupListItem, 0, len(groups))
	for _, g := range groups {
		list = append(list, GroupListItem{
			ID:             g.ID,
			ConversationID: g.ConversationID,
			Name:           g.Name,
			Avatar:         g.Avatar,
			OwnerID:        g.OwnerID,
			OwnerName:      ownerNameMap[g.OwnerID],
			MemberCount:    memberCountMap[g.ConversationID],
			MaxMembers:     g.MaxMembers,
			Status:         g.Status,
			IsAllMuted:     g.IsAllMuted,
			CreatedAt:      g.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	logs.Info(ctx, funcName, "获取群组列表成功",
		zap.Int64("total", total), zap.Int("total_pages", totalPages))

	return list, total, nil
}

// GetGroupDetail 获取群组详情（含成员列表）
func (s *GroupManageService) GetGroupDetail(ctx context.Context, groupID int64) (*GroupDetailInfo, error) {
	funcName := "service.group_manage_service.GetGroupDetail"
	logs.Info(ctx, funcName, "获取群组详情", zap.Int64("group_id", groupID))

	group, err := s.groupDAO.GetByID(ctx, groupID)
	if err != nil {
		return nil, err
	}

	memberCount, _ := s.groupDAO.GetMemberCount(ctx, group.ConversationID)
	ownerName := s.getUsername(ctx, group.OwnerID)

	members, err := s.groupDAO.GetMembers(ctx, group.ConversationID)
	if err != nil {
		return nil, err
	}

	memberInfos := make([]GroupMemberInfo, 0, len(members))
	for _, m := range members {
		username := s.getUsername(ctx, m.UserID)
		memberInfos = append(memberInfos, GroupMemberInfo{
			UserID:   m.UserID,
			Username: username,
			Nickname: m.Nickname,
			Role:     m.Role,
			IsMuted:  m.IsMuted,
			JoinedAt: m.JoinedAt.Format("2006-01-02 15:04:05"),
		})
	}

	detail := &GroupDetailInfo{
		GroupListItem: GroupListItem{
			ID:             group.ID,
			ConversationID: group.ConversationID,
			Name:           group.Name,
			Avatar:         group.Avatar,
			OwnerID:        group.OwnerID,
			OwnerName:      ownerName,
			MemberCount:    memberCount,
			MaxMembers:     group.MaxMembers,
			Status:         group.Status,
			IsAllMuted:     group.IsAllMuted,
			CreatedAt:      group.CreatedAt.Format("2006-01-02 15:04:05"),
		},
		Notice:       group.Notice,
		IsSearchable: group.IsSearchable,
		Members:      memberInfos,
	}

	return detail, nil
}

// DissolveGroup 管理端解散群聊
func (s *GroupManageService) DissolveGroup(ctx context.Context, groupID int64) error {
	funcName := "service.group_manage_service.DissolveGroup"
	logs.Info(ctx, funcName, "管理端解散群聊", zap.Int64("group_id", groupID))
	return s.groupDAO.DissolveGroup(ctx, groupID)
}

// getUsername 通过用户 ID 获取用户名（优先昵称，其次用户名）
func (s *GroupManageService) getUsername(ctx context.Context, userID int64) string {
	var username string
	s.db.WithContext(ctx).Table("auth_users").
		Where("id = ?", userID).
		Pluck("COALESCE(NULLIF(nickname, ''), username)", &username)
	if username == "" {
		return strconv.FormatInt(userID, 10)
	}
	return username
}

// batchGetUsernames 批量查询用户名，返回 userID → 显示名 映射
func (s *GroupManageService) batchGetUsernames(ctx context.Context, userIDs []int64) map[int64]string {
	result := make(map[int64]string, len(userIDs))
	if len(userIDs) == 0 {
		return result
	}

	type row struct {
		ID       int64  `gorm:"column:id"`
		Username string `gorm:"column:display_name"`
	}
	var rows []row
	s.db.WithContext(ctx).Table("auth_users").
		Select("id, COALESCE(NULLIF(nickname, ''), username) AS display_name").
		Where("id IN ?", userIDs).
		Find(&rows)

	for _, r := range rows {
		result[r.ID] = r.Username
	}
	for _, uid := range userIDs {
		if _, ok := result[uid]; !ok {
			result[uid] = strconv.FormatInt(uid, 10)
		}
	}
	return result
}

// batchGetMemberCounts 批量查询会话的成员数，返回 conversationID → count 映射
func (s *GroupManageService) batchGetMemberCounts(ctx context.Context, conversationIDs []int64) map[int64]int64 {
	result := make(map[int64]int64, len(conversationIDs))
	if len(conversationIDs) == 0 {
		return result
	}

	type row struct {
		ConversationID int64 `gorm:"column:conversation_id"`
		Count          int64 `gorm:"column:cnt"`
	}
	var rows []row
	s.db.WithContext(ctx).Table("im_conversation_members").
		Select("conversation_id, COUNT(*) AS cnt").
		Where("conversation_id IN ?", conversationIDs).
		Group("conversation_id").
		Find(&rows)

	for _, r := range rows {
		result[r.ConversationID] = r.Count
	}
	return result
}
