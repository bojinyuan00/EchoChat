package service

import (
	"context"

	"github.com/echochat/backend/app/contact/model"
	"github.com/echochat/backend/pkg/logs"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ContactManageService struct {
	db *gorm.DB
}

func NewContactManageService(db *gorm.DB) *ContactManageService {
	return &ContactManageService{db: db}
}

type AdminFriendship struct {
	ID             int64  `json:"id"`
	UserID         int64  `json:"user_id"`
	UserUsername   string `json:"user_username"`
	FriendID       int64  `json:"friend_id"`
	FriendUsername string `json:"friend_username"`
	Status         int    `json:"status"`
	CreatedAt      string `json:"created_at"`
}

// GetAllFriendships 获取所有好友关系（分页）
func (s *ContactManageService) GetAllFriendships(ctx context.Context, page, pageSize int) ([]AdminFriendship, int64, error) {
	funcName := "service.contact_manage_service.GetAllFriendships"
	logs.Debug(ctx, funcName, "管理端查询好友关系")

	var total int64
	s.db.WithContext(ctx).Model(&model.Friendship{}).Count(&total)

	var results []AdminFriendship
	offset := (page - 1) * pageSize
	err := s.db.WithContext(ctx).
		Table("contact_friendships f").
		Select("f.id, f.user_id, u1.username as user_username, f.friend_id, u2.username as friend_username, f.status, TO_CHAR(f.created_at, 'YYYY-MM-DD HH24:MI:SS') as created_at").
		Joins("JOIN auth_users u1 ON u1.id = f.user_id").
		Joins("JOIN auth_users u2 ON u2.id = f.friend_id").
		Order("f.created_at DESC").
		Offset(offset).Limit(pageSize).
		Scan(&results).Error

	return results, total, err
}

// DeleteFriendship 管理员删除好友关系（双向删除）
func (s *ContactManageService) DeleteFriendship(ctx context.Context, friendshipID int64) error {
	funcName := "service.contact_manage_service.DeleteFriendship"
	logs.Info(ctx, funcName, "管理员删除好友关系", zap.Int64("friendship_id", friendshipID))

	var f model.Friendship
	if err := s.db.WithContext(ctx).First(&f, friendshipID).Error; err != nil {
		return err
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		tx.Where("user_id = ? AND friend_id = ?", f.UserID, f.FriendID).Delete(&model.Friendship{})
		tx.Where("user_id = ? AND friend_id = ?", f.FriendID, f.UserID).Delete(&model.Friendship{})
		return nil
	})
}
