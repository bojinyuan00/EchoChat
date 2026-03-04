// Package dao 提供 group 模块的数据库访问操作
package dao

import (
	"context"
	"time"

	"github.com/echochat/backend/app/constants"
	"github.com/echochat/backend/app/dto"
	"github.com/echochat/backend/app/group/model"
	imModel "github.com/echochat/backend/app/im/model"
	"github.com/echochat/backend/pkg/logs"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// GroupDAO 群聊数据访问对象
type GroupDAO struct {
	db *gorm.DB
}

// NewGroupDAO 创建 GroupDAO 实例
func NewGroupDAO(db *gorm.DB) *GroupDAO {
	return &GroupDAO{db: db}
}

// CreateGroupWithMembers 在事务中创建群聊（会话 + 群信息 + 群成员）
// 创建者自动成为群主（role=2）
func (d *GroupDAO) CreateGroupWithMembers(ctx context.Context, ownerID int64, name, avatar string, memberIDs []int64) (*model.Group, error) {
	funcName := "dao.group_dao.CreateGroupWithMembers"
	logs.Info(ctx, funcName, "创建群聊",
		zap.Int64("owner_id", ownerID), zap.String("name", name), zap.Int("member_count", len(memberIDs)+1))

	var group model.Group
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		conv := &imModel.Conversation{
			Type:      constants.ConversationTypeGroup,
			CreatorID: ownerID,
		}
		if err := tx.Create(conv).Error; err != nil {
			logs.Error(ctx, funcName, "创建会话失败", zap.Error(err))
			return err
		}

		group = model.Group{
			ConversationID: conv.ID,
			Name:           name,
			Avatar:         avatar,
			OwnerID:        ownerID,
			MaxMembers:     constants.GroupDefaultMaxMembers,
			Status:         constants.GroupStatusNormal,
		}
		if err := tx.Create(&group).Error; err != nil {
			logs.Error(ctx, funcName, "创建群信息失败", zap.Error(err))
			return err
		}

		now := time.Now()
		ownerMember := &imModel.ConversationMember{
			ConversationID: conv.ID,
			UserID:         ownerID,
			Role:           constants.GroupRoleOwner,
			JoinedAt:       &now,
		}
		if err := tx.Create(ownerMember).Error; err != nil {
			logs.Error(ctx, funcName, "创建群主成员记录失败", zap.Error(err))
			return err
		}

		for _, uid := range memberIDs {
			member := &imModel.ConversationMember{
				ConversationID: conv.ID,
				UserID:         uid,
				Role:           constants.GroupRoleNormal,
				JoinedAt:       &now,
			}
			if err := tx.Create(member).Error; err != nil {
				logs.Error(ctx, funcName, "创建群成员记录失败",
					zap.Int64("user_id", uid), zap.Error(err))
				return err
			}
		}

		return nil
	})

	if err != nil {
		logs.Error(ctx, funcName, "创建群聊失败", zap.Error(err))
		return nil, err
	}
	return &group, nil
}

// GetByID 根据群 ID 获取群信息
func (d *GroupDAO) GetByID(ctx context.Context, groupID int64) (*model.Group, error) {
	funcName := "dao.group_dao.GetByID"
	logs.Debug(ctx, funcName, "获取群信息", zap.Int64("group_id", groupID))

	var group model.Group
	err := d.db.WithContext(ctx).First(&group, groupID).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

// GetByConversationID 根据会话 ID 获取群信息
func (d *GroupDAO) GetByConversationID(ctx context.Context, conversationID int64) (*model.Group, error) {
	funcName := "dao.group_dao.GetByConversationID"
	logs.Debug(ctx, funcName, "按会话 ID 获取群信息", zap.Int64("conversation_id", conversationID))

	var group model.Group
	err := d.db.WithContext(ctx).Where("conversation_id = ?", conversationID).First(&group).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

// UpdateGroupInfo 更新群基本信息（名称/头像/公告/可搜索性）
func (d *GroupDAO) UpdateGroupInfo(ctx context.Context, groupID int64, updates map[string]interface{}) error {
	funcName := "dao.group_dao.UpdateGroupInfo"
	logs.Info(ctx, funcName, "更新群信息", zap.Int64("group_id", groupID))

	return d.db.WithContext(ctx).
		Model(&model.Group{}).
		Where("id = ?", groupID).
		Updates(updates).Error
}

// DissolveGroup 解散群聊（标记状态为已解散）
func (d *GroupDAO) DissolveGroup(ctx context.Context, groupID int64) error {
	funcName := "dao.group_dao.DissolveGroup"
	logs.Info(ctx, funcName, "解散群聊", zap.Int64("group_id", groupID))

	return d.db.WithContext(ctx).
		Model(&model.Group{}).
		Where("id = ?", groupID).
		Update("status", constants.GroupStatusDissolved).Error
}

// TransferOwner 转让群主
func (d *GroupDAO) TransferOwner(ctx context.Context, groupID, oldOwnerID, newOwnerID int64, conversationID int64) error {
	funcName := "dao.group_dao.TransferOwner"
	logs.Info(ctx, funcName, "转让群主",
		zap.Int64("group_id", groupID), zap.Int64("old_owner", oldOwnerID), zap.Int64("new_owner", newOwnerID))

	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Group{}).
			Where("id = ?", groupID).
			Update("owner_id", newOwnerID).Error; err != nil {
			return err
		}

		if err := tx.Model(&imModel.ConversationMember{}).
			Where("conversation_id = ? AND user_id = ?", conversationID, oldOwnerID).
			Update("role", constants.GroupRoleNormal).Error; err != nil {
			return err
		}

		if err := tx.Model(&imModel.ConversationMember{}).
			Where("conversation_id = ? AND user_id = ?", conversationID, newOwnerID).
			Update("role", constants.GroupRoleOwner).Error; err != nil {
			return err
		}

		return nil
	})
}

// SetAllMuted 设置/取消全体禁言
func (d *GroupDAO) SetAllMuted(ctx context.Context, groupID int64, isMuted bool) error {
	funcName := "dao.group_dao.SetAllMuted"
	logs.Info(ctx, funcName, "设置全体禁言", zap.Int64("group_id", groupID), zap.Bool("is_muted", isMuted))

	return d.db.WithContext(ctx).
		Model(&model.Group{}).
		Where("id = ?", groupID).
		Update("is_all_muted", isMuted).Error
}

// GetGroupBrief 获取群简要信息（满足 im/service.GroupInfoGetter 接口）
func (d *GroupDAO) GetGroupBrief(ctx context.Context, conversationID int64) (*dto.GroupBrief, error) {
	funcName := "dao.group_dao.GetGroupBrief"
	logs.Debug(ctx, funcName, "获取群简要信息", zap.Int64("conversation_id", conversationID))

	var group model.Group
	err := d.db.WithContext(ctx).Where("conversation_id = ?", conversationID).First(&group).Error
	if err != nil {
		return nil, err
	}
	return &dto.GroupBrief{
		ID:         group.ID,
		Name:       group.Name,
		Avatar:     group.Avatar,
		IsAllMuted: group.IsAllMuted,
		Status:     group.Status,
	}, nil
}

// GetMemberCount 获取群成员数量
func (d *GroupDAO) GetMemberCount(ctx context.Context, conversationID int64) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).
		Model(&imModel.ConversationMember{}).
		Where("conversation_id = ?", conversationID).
		Count(&count).Error
	return count, err
}

// GetMember 获取指定群成员记录
func (d *GroupDAO) GetMember(ctx context.Context, conversationID, userID int64) (*imModel.ConversationMember, error) {
	funcName := "dao.group_dao.GetMember"
	logs.Debug(ctx, funcName, "查询群成员",
		zap.Int64("conversation_id", conversationID), zap.Int64("user_id", userID))

	var member imModel.ConversationMember
	err := d.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		First(&member).Error
	if err != nil {
		return nil, err
	}
	return &member, nil
}

// GetMembers 获取群所有成员列表
func (d *GroupDAO) GetMembers(ctx context.Context, conversationID int64) ([]imModel.ConversationMember, error) {
	funcName := "dao.group_dao.GetMembers"
	logs.Debug(ctx, funcName, "获取群成员列表", zap.Int64("conversation_id", conversationID))

	var members []imModel.ConversationMember
	err := d.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("role DESC, joined_at ASC").
		Find(&members).Error
	return members, err
}

// GetMemberIDs 获取群所有成员 ID
func (d *GroupDAO) GetMemberIDs(ctx context.Context, conversationID int64) ([]int64, error) {
	var ids []int64
	err := d.db.WithContext(ctx).
		Model(&imModel.ConversationMember{}).
		Where("conversation_id = ?", conversationID).
		Pluck("user_id", &ids).Error
	return ids, err
}

// GetAdminIDs 获取群主和管理员 ID 列表（用于推送入群申请通知）
func (d *GroupDAO) GetAdminIDs(ctx context.Context, conversationID int64) ([]int64, error) {
	var ids []int64
	err := d.db.WithContext(ctx).
		Model(&imModel.ConversationMember{}).
		Where("conversation_id = ? AND role >= ?", conversationID, constants.GroupRoleAdmin).
		Pluck("user_id", &ids).Error
	return ids, err
}

// AddMember 添加群成员
func (d *GroupDAO) AddMember(ctx context.Context, conversationID, userID int64, role int) error {
	funcName := "dao.group_dao.AddMember"
	logs.Info(ctx, funcName, "添加群成员",
		zap.Int64("conversation_id", conversationID), zap.Int64("user_id", userID))

	now := time.Now()
	member := &imModel.ConversationMember{
		ConversationID: conversationID,
		UserID:         userID,
		Role:           role,
		JoinedAt:       &now,
	}
	return d.db.WithContext(ctx).Create(member).Error
}

// RemoveMember 移除群成员
func (d *GroupDAO) RemoveMember(ctx context.Context, conversationID, userID int64) error {
	funcName := "dao.group_dao.RemoveMember"
	logs.Info(ctx, funcName, "移除群成员",
		zap.Int64("conversation_id", conversationID), zap.Int64("user_id", userID))

	return d.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Delete(&imModel.ConversationMember{}).Error
}

// UpdateMemberRole 更新成员角色
func (d *GroupDAO) UpdateMemberRole(ctx context.Context, conversationID, userID int64, role int) error {
	funcName := "dao.group_dao.UpdateMemberRole"
	logs.Info(ctx, funcName, "更新成员角色",
		zap.Int64("conversation_id", conversationID), zap.Int64("user_id", userID), zap.Int("role", role))

	return d.db.WithContext(ctx).
		Model(&imModel.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Update("role", role).Error
}

// UpdateMemberMuted 更新成员禁言状态
func (d *GroupDAO) UpdateMemberMuted(ctx context.Context, conversationID, userID int64, isMuted bool) error {
	funcName := "dao.group_dao.UpdateMemberMuted"
	logs.Info(ctx, funcName, "更新成员禁言状态",
		zap.Int64("conversation_id", conversationID), zap.Int64("user_id", userID), zap.Bool("is_muted", isMuted))

	return d.db.WithContext(ctx).
		Model(&imModel.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Update("is_muted", isMuted).Error
}

// UpdateMemberNickname 更新成员群内昵称
func (d *GroupDAO) UpdateMemberNickname(ctx context.Context, conversationID, userID int64, nickname string) error {
	funcName := "dao.group_dao.UpdateMemberNickname"
	logs.Info(ctx, funcName, "更新群昵称",
		zap.Int64("conversation_id", conversationID), zap.Int64("user_id", userID))

	return d.db.WithContext(ctx).
		Model(&imModel.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Update("nickname", nickname).Error
}

// SearchGroups 搜索可发现的群聊（按群名称全文搜索）
func (d *GroupDAO) SearchGroups(ctx context.Context, keyword string, offset, limit int) ([]model.Group, int64, error) {
	funcName := "dao.group_dao.SearchGroups"
	logs.Debug(ctx, funcName, "搜索群聊", zap.String("keyword", keyword))

	var total int64
	query := d.db.WithContext(ctx).
		Model(&model.Group{}).
		Where("status = ? AND is_searchable = true AND to_tsvector('simple', name) @@ plainto_tsquery('simple', ?)",
			constants.GroupStatusNormal, keyword)

	if err := query.Count(&total).Error; err != nil {
		logs.Error(ctx, funcName, "搜索计数失败", zap.Error(err))
		return nil, 0, err
	}

	var groups []model.Group
	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&groups).Error
	if err != nil {
		logs.Error(ctx, funcName, "搜索查询失败", zap.Error(err))
		return nil, 0, err
	}
	return groups, total, nil
}
