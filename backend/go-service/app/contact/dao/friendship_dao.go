// Package dao 提供 contact 模块的数据库访问操作
package dao

import (
	"context"
	"time"

	authModel "github.com/echochat/backend/app/auth/model"
	"github.com/echochat/backend/app/constants"
	"github.com/echochat/backend/app/contact/model"
	"github.com/echochat/backend/pkg/logs"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// FriendshipDAO 好友关系数据访问对象
type FriendshipDAO struct {
	db *gorm.DB
}

// NewFriendshipDAO 创建 FriendshipDAO 实例
func NewFriendshipDAO(db *gorm.DB) *FriendshipDAO {
	return &FriendshipDAO{db: db}
}

// CreateRequest 创建好友申请（单向 A→B，status=0）
func (d *FriendshipDAO) CreateRequest(ctx context.Context, userID, friendID int64, message string) (*model.Friendship, error) {
	funcName := "dao.friendship_dao.CreateRequest"
	logs.Info(ctx, funcName, "创建好友申请",
		zap.Int64("user_id", userID), zap.Int64("friend_id", friendID))

	f := &model.Friendship{
		UserID:   userID,
		FriendID: friendID,
		Status:   constants.FriendshipStatusPending,
		Message:  message,
	}
	err := d.db.WithContext(ctx).Create(f).Error
	if err != nil {
		logs.Error(ctx, funcName, "创建好友申请失败", zap.Error(err))
	}
	return f, err
}

// AcceptRequest 接受好友申请（事务内：更新 A→B 为 accepted + 创建 B→A）
func (d *FriendshipDAO) AcceptRequest(ctx context.Context, requestID, userID int64) error {
	funcName := "dao.friendship_dao.AcceptRequest"
	logs.Info(ctx, funcName, "接受好友申请",
		zap.Int64("request_id", requestID), zap.Int64("user_id", userID))

	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var req model.Friendship
		if err := tx.First(&req, requestID).Error; err != nil {
			return err
		}
		if req.FriendID != userID {
			return gorm.ErrRecordNotFound
		}

		now := time.Now()
		if err := tx.Model(&req).Updates(map[string]interface{}{
			"status":     constants.FriendshipStatusAccepted,
			"updated_at": now,
		}).Error; err != nil {
			return err
		}

		reverse := &model.Friendship{
			UserID:   userID,
			FriendID: req.UserID,
			Status:   constants.FriendshipStatusAccepted,
		}
		return tx.Create(reverse).Error
	})
}

// RejectRequest 拒绝好友申请
func (d *FriendshipDAO) RejectRequest(ctx context.Context, requestID, userID int64) error {
	funcName := "dao.friendship_dao.RejectRequest"
	logs.Info(ctx, funcName, "拒绝好友申请",
		zap.Int64("request_id", requestID), zap.Int64("user_id", userID))

	result := d.db.WithContext(ctx).
		Model(&model.Friendship{}).
		Where("id = ? AND friend_id = ? AND status = ?", requestID, userID, constants.FriendshipStatusPending).
		Update("status", constants.FriendshipStatusRejected)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// GetFriendList 获取好友列表（status=1 的记录，JOIN 用户表获取详情）
func (d *FriendshipDAO) GetFriendList(ctx context.Context, userID int64, groupID *int64) ([]FriendWithUser, error) {
	funcName := "dao.friendship_dao.GetFriendList"
	logs.Debug(ctx, funcName, "查询好友列表", zap.Int64("user_id", userID))

	var results []FriendWithUser
	query := d.db.WithContext(ctx).
		Table("contact_friendships f").
		Select("f.id, f.friend_id as user_id, u.username, u.nickname, u.avatar, f.remark, f.group_id, f.created_at").
		Joins("JOIN auth_users u ON u.id = f.friend_id").
		Where("f.user_id = ? AND f.status = ?", userID, constants.FriendshipStatusAccepted)

	if groupID != nil {
		query = query.Where("f.group_id = ?", *groupID)
	}

	err := query.Order("u.nickname ASC").Scan(&results).Error
	if err != nil {
		logs.Error(ctx, funcName, "查询好友列表失败", zap.Error(err))
	}
	return results, err
}

// FriendWithUser 好友列表查询结果（JOIN 用户表）
type FriendWithUser struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
	Nickname  string    `json:"nickname"`
	Avatar    string    `json:"avatar"`
	Remark    string    `json:"remark"`
	GroupID   *int64    `json:"group_id"`
	CreatedAt time.Time `json:"created_at"`
}

// GetPendingRequests 获取收到的待处理好友申请
func (d *FriendshipDAO) GetPendingRequests(ctx context.Context, userID int64) ([]RequestWithUser, error) {
	funcName := "dao.friendship_dao.GetPendingRequests"
	logs.Debug(ctx, funcName, "查询待处理申请", zap.Int64("user_id", userID))

	var results []RequestWithUser
	err := d.db.WithContext(ctx).
		Table("contact_friendships f").
		Select("f.id, f.user_id, u.username, u.nickname, u.avatar, f.message, f.status, f.created_at").
		Joins("JOIN auth_users u ON u.id = f.user_id").
		Where("f.friend_id = ? AND f.status = ?", userID, constants.FriendshipStatusPending).
		Order("f.created_at DESC").
		Scan(&results).Error

	if err != nil {
		logs.Error(ctx, funcName, "查询待处理申请失败", zap.Error(err))
	}
	return results, err
}

// RequestWithUser 好友申请查询结果（JOIN 申请方用户表）
type RequestWithUser struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
	Nickname  string    `json:"nickname"`
	Avatar    string    `json:"avatar"`
	Message   string    `json:"message"`
	Status    int       `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// DeleteFriend 删除好友（事务内双向删除）
func (d *FriendshipDAO) DeleteFriend(ctx context.Context, userID, friendID int64) error {
	funcName := "dao.friendship_dao.DeleteFriend"
	logs.Info(ctx, funcName, "删除好友",
		zap.Int64("user_id", userID), zap.Int64("friend_id", friendID))

	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ? AND friend_id = ?", userID, friendID).
			Delete(&model.Friendship{}).Error; err != nil {
			return err
		}
		return tx.Where("user_id = ? AND friend_id = ?", friendID, userID).
			Delete(&model.Friendship{}).Error
	})
}

// UpdateRemark 设置好友备注
func (d *FriendshipDAO) UpdateRemark(ctx context.Context, userID, friendID int64, remark string) error {
	funcName := "dao.friendship_dao.UpdateRemark"
	logs.Info(ctx, funcName, "更新好友备注",
		zap.Int64("user_id", userID), zap.Int64("friend_id", friendID))

	return d.db.WithContext(ctx).
		Model(&model.Friendship{}).
		Where("user_id = ? AND friend_id = ? AND status = ?", userID, friendID, constants.FriendshipStatusAccepted).
		Update("remark", remark).Error
}

// BlockUser 拉黑用户（事务内：删除双向好友 + 创建单向拉黑记录）
func (d *FriendshipDAO) BlockUser(ctx context.Context, userID, targetID int64) error {
	funcName := "dao.friendship_dao.BlockUser"
	logs.Info(ctx, funcName, "拉黑用户",
		zap.Int64("user_id", userID), zap.Int64("target_id", targetID))

	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		tx.Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)",
			userID, targetID, targetID, userID).
			Delete(&model.Friendship{})

		block := &model.Friendship{
			UserID:   userID,
			FriendID: targetID,
			Status:   constants.FriendshipStatusBlocked,
		}
		return tx.Create(block).Error
	})
}

// UnblockUser 取消拉黑
func (d *FriendshipDAO) UnblockUser(ctx context.Context, userID, targetID int64) error {
	funcName := "dao.friendship_dao.UnblockUser"
	logs.Info(ctx, funcName, "取消拉黑",
		zap.Int64("user_id", userID), zap.Int64("target_id", targetID))

	return d.db.WithContext(ctx).
		Where("user_id = ? AND friend_id = ? AND status = ?", userID, targetID, constants.FriendshipStatusBlocked).
		Delete(&model.Friendship{}).Error
}

// GetBlockList 获取黑名单列表
func (d *FriendshipDAO) GetBlockList(ctx context.Context, userID int64) ([]FriendWithUser, error) {
	funcName := "dao.friendship_dao.GetBlockList"
	logs.Debug(ctx, funcName, "查询黑名单", zap.Int64("user_id", userID))

	var results []FriendWithUser
	err := d.db.WithContext(ctx).
		Table("contact_friendships f").
		Select("f.id, f.friend_id as user_id, u.username, u.nickname, u.avatar, f.remark, f.group_id, f.created_at").
		Joins("JOIN auth_users u ON u.id = f.friend_id").
		Where("f.user_id = ? AND f.status = ?", userID, constants.FriendshipStatusBlocked).
		Scan(&results).Error
	return results, err
}

// IsBlocked 检查 userID 是否被 targetID 拉黑
func (d *FriendshipDAO) IsBlocked(ctx context.Context, userID, targetID int64) (bool, error) {
	var count int64
	err := d.db.WithContext(ctx).
		Model(&model.Friendship{}).
		Where("user_id = ? AND friend_id = ? AND status = ?", targetID, userID, constants.FriendshipStatusBlocked).
		Count(&count).Error
	return count > 0, err
}

// IsFriend 检查是否互为好友
func (d *FriendshipDAO) IsFriend(ctx context.Context, userID, friendID int64) (bool, error) {
	var count int64
	err := d.db.WithContext(ctx).
		Model(&model.Friendship{}).
		Where("user_id = ? AND friend_id = ? AND status = ?", userID, friendID, constants.FriendshipStatusAccepted).
		Count(&count).Error
	return count > 0, err
}

// HasPendingRequest 检查是否已有待处理的申请（A→B 或 B→A）
func (d *FriendshipDAO) HasPendingRequest(ctx context.Context, userID, friendID int64) (bool, error) {
	var count int64
	err := d.db.WithContext(ctx).
		Model(&model.Friendship{}).
		Where("((user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)) AND status = ?",
			userID, friendID, friendID, userID, constants.FriendshipStatusPending).
		Count(&count).Error
	return count > 0, err
}

// GetRequestByID 根据 ID 获取好友申请
func (d *FriendshipDAO) GetRequestByID(ctx context.Context, id int64) (*model.Friendship, error) {
	var f model.Friendship
	err := d.db.WithContext(ctx).First(&f, id).Error
	return &f, err
}

// GetCommonFriends 获取共同好友 ID 列表
func (d *FriendshipDAO) GetCommonFriends(ctx context.Context, userID, targetID int64) ([]int64, error) {
	funcName := "dao.friendship_dao.GetCommonFriends"
	logs.Debug(ctx, funcName, "查询共同好友",
		zap.Int64("user_id", userID), zap.Int64("target_id", targetID))

	var ids []int64
	err := d.db.WithContext(ctx).
		Raw(`SELECT a.friend_id FROM contact_friendships a
			 JOIN contact_friendships b ON a.friend_id = b.friend_id
			 WHERE a.user_id = ? AND a.status = ? AND b.user_id = ? AND b.status = ?`,
			userID, constants.FriendshipStatusAccepted, targetID, constants.FriendshipStatusAccepted).
		Scan(&ids).Error
	return ids, err
}

// SearchUsers 搜索用户（按用户名或昵称模糊匹配，排除自己）
func (d *FriendshipDAO) SearchUsers(ctx context.Context, keyword string, excludeUserID int64, page, pageSize int) ([]authModel.User, int64, error) {
	funcName := "dao.friendship_dao.SearchUsers"
	logs.Debug(ctx, funcName, "搜索用户", zap.String("keyword", keyword))

	var users []authModel.User
	var total int64

	query := d.db.WithContext(ctx).
		Model(&authModel.User{}).
		Where("id != ? AND status = 1 AND (username LIKE ? OR nickname LIKE ?)",
			excludeUserID, "%"+keyword+"%", "%"+keyword+"%")

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Select("id, username, nickname, avatar").
		Offset(offset).Limit(pageSize).
		Find(&users).Error

	return users, total, err
}

// GetFriendIDs 获取用户的所有好友 ID 列表
func (d *FriendshipDAO) GetFriendIDs(ctx context.Context, userID int64) ([]int64, error) {
	var ids []int64
	err := d.db.WithContext(ctx).
		Model(&model.Friendship{}).
		Where("user_id = ? AND status = ?", userID, constants.FriendshipStatusAccepted).
		Pluck("friend_id", &ids).Error
	return ids, err
}

// CountFriendsByGroup 统计每个分组的好友数
func (d *FriendshipDAO) CountFriendsByGroup(ctx context.Context, userID int64) (map[int64]int, error) {
	type GroupCount struct {
		GroupID int64 `gorm:"column:group_id"`
		Count   int   `gorm:"column:cnt"`
	}
	var counts []GroupCount
	err := d.db.WithContext(ctx).
		Model(&model.Friendship{}).
		Select("group_id, COUNT(*) as cnt").
		Where("user_id = ? AND status = ? AND group_id IS NOT NULL", userID, constants.FriendshipStatusAccepted).
		Group("group_id").
		Scan(&counts).Error

	result := make(map[int64]int)
	for _, c := range counts {
		result[c.GroupID] = c.Count
	}
	return result, err
}
