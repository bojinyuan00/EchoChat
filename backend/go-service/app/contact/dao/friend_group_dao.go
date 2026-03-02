package dao

import (
	"context"

	"github.com/echochat/backend/app/contact/model"
	"github.com/echochat/backend/pkg/logs"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// FriendGroupDAO 好友分组数据访问对象
type FriendGroupDAO struct {
	db *gorm.DB
}

// NewFriendGroupDAO 创建 FriendGroupDAO 实例
func NewFriendGroupDAO(db *gorm.DB) *FriendGroupDAO {
	return &FriendGroupDAO{db: db}
}

// CreateGroup 创建好友分组
func (d *FriendGroupDAO) CreateGroup(ctx context.Context, userID int64, name string) (*model.FriendGroup, error) {
	funcName := "dao.friend_group_dao.CreateGroup"
	logs.Info(ctx, funcName, "创建好友分组",
		zap.Int64("user_id", userID), zap.String("name", name))

	group := &model.FriendGroup{
		UserID: userID,
		Name:   name,
	}
	err := d.db.WithContext(ctx).Create(group).Error
	if err != nil {
		logs.Error(ctx, funcName, "创建分组失败", zap.Error(err))
	}
	return group, err
}

// GetGroups 获取用户的所有好友分组
func (d *FriendGroupDAO) GetGroups(ctx context.Context, userID int64) ([]model.FriendGroup, error) {
	funcName := "dao.friend_group_dao.GetGroups"
	logs.Debug(ctx, funcName, "查询好友分组", zap.Int64("user_id", userID))

	var groups []model.FriendGroup
	err := d.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("sort_order ASC, id ASC").
		Find(&groups).Error
	return groups, err
}

// UpdateGroup 更新分组信息
func (d *FriendGroupDAO) UpdateGroup(ctx context.Context, groupID int64, userID int64, name string, sortOrder *int) error {
	funcName := "dao.friend_group_dao.UpdateGroup"
	logs.Info(ctx, funcName, "更新好友分组",
		zap.Int64("group_id", groupID), zap.String("name", name))

	updates := map[string]interface{}{"name": name}
	if sortOrder != nil {
		updates["sort_order"] = *sortOrder
	}

	result := d.db.WithContext(ctx).
		Model(&model.FriendGroup{}).
		Where("id = ? AND user_id = ?", groupID, userID).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// DeleteGroup 删除分组（同时将该分组下的好友移出分组）
func (d *FriendGroupDAO) DeleteGroup(ctx context.Context, groupID int64, userID int64) error {
	funcName := "dao.friend_group_dao.DeleteGroup"
	logs.Info(ctx, funcName, "删除好友分组",
		zap.Int64("group_id", groupID), zap.Int64("user_id", userID))

	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		tx.Model(&model.Friendship{}).
			Where("user_id = ? AND group_id = ?", userID, groupID).
			Update("group_id", nil)

		result := tx.Where("id = ? AND user_id = ?", groupID, userID).
			Delete(&model.FriendGroup{})
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return result.Error
	})
}

// MoveToGroup 移动好友到指定分组
func (d *FriendGroupDAO) MoveToGroup(ctx context.Context, userID, friendID int64, groupID *int64) error {
	funcName := "dao.friend_group_dao.MoveToGroup"
	logs.Info(ctx, funcName, "移动好友到分组",
		zap.Int64("user_id", userID), zap.Int64("friend_id", friendID))

	return d.db.WithContext(ctx).
		Model(&model.Friendship{}).
		Where("user_id = ? AND friend_id = ? AND status = 1", userID, friendID).
		Update("group_id", groupID).Error
}

// GetGroupByID 根据 ID 获取分组
func (d *FriendGroupDAO) GetGroupByID(ctx context.Context, groupID, userID int64) (*model.FriendGroup, error) {
	var group model.FriendGroup
	err := d.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", groupID, userID).
		First(&group).Error
	return &group, err
}
