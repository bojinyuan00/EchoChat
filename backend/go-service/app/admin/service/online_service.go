package service

import (
	"context"

	wsApp "github.com/echochat/backend/app/ws"
	"github.com/echochat/backend/pkg/logs"
)

type OnlineManageService struct {
	onlineService *wsApp.OnlineService
}

func NewOnlineManageService(onlineService *wsApp.OnlineService) *OnlineManageService {
	return &OnlineManageService{onlineService: onlineService}
}

func (s *OnlineManageService) GetOnlineUsers(ctx context.Context) ([]int64, error) {
	funcName := "service.online_manage_service.GetOnlineUsers"
	logs.Debug(ctx, funcName, "获取在线用户列表")
	return s.onlineService.GetOnlineUserIDs(ctx)
}

func (s *OnlineManageService) GetOnlineCount(ctx context.Context) (int64, error) {
	return s.onlineService.GetOnlineCount(ctx)
}
