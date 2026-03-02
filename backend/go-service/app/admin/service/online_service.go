package service

import (
	"context"

	authModel "github.com/echochat/backend/app/auth/model"
	wsApp "github.com/echochat/backend/app/ws"
	"github.com/echochat/backend/pkg/logs"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// OnlineUserInfo 在线用户信息（包含用户名，用于管理端展示）
type OnlineUserInfo struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
}

// OnlineManageService 管理端在线状态查询服务
type OnlineManageService struct {
	onlineService *wsApp.OnlineService
	db            *gorm.DB
}

// NewOnlineManageService 创建 OnlineManageService 实例
func NewOnlineManageService(onlineService *wsApp.OnlineService, db *gorm.DB) *OnlineManageService {
	return &OnlineManageService{
		onlineService: onlineService,
		db:            db,
	}
}

// GetOnlineUsers 获取在线用户列表（含用户名）
func (s *OnlineManageService) GetOnlineUsers(ctx context.Context) ([]OnlineUserInfo, error) {
	funcName := "service.online_manage_service.GetOnlineUsers"
	logs.Debug(ctx, funcName, "获取在线用户列表")

	ids, err := s.onlineService.GetOnlineUserIDs(ctx)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []OnlineUserInfo{}, nil
	}

	var users []authModel.User
	if err := s.db.WithContext(ctx).
		Model(&authModel.User{}).
		Select("id, username").
		Where("id IN ?", ids).
		Find(&users).Error; err != nil {
		logs.Error(ctx, funcName, "查询在线用户信息失败", zap.Error(err))
		return nil, err
	}

	userMap := make(map[int64]string, len(users))
	for _, u := range users {
		userMap[u.ID] = u.Username
	}

	result := make([]OnlineUserInfo, 0, len(ids))
	for _, id := range ids {
		result = append(result, OnlineUserInfo{
			UserID:   id,
			Username: userMap[id],
		})
	}
	return result, nil
}

// GetOnlineCount 获取在线用户总数
func (s *OnlineManageService) GetOnlineCount(ctx context.Context) (int64, error) {
	return s.onlineService.GetOnlineCount(ctx)
}
