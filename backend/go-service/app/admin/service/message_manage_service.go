// Package service 提供 admin 模块的核心业务逻辑
package service

import (
	"context"
	"errors"

	adminDAO "github.com/echochat/backend/app/admin/dao"
	authDAO "github.com/echochat/backend/app/auth/dao"
	imDAO "github.com/echochat/backend/app/im/dao"
	"github.com/echochat/backend/app/constants"
	"github.com/echochat/backend/app/dto"
	"github.com/echochat/backend/pkg/logs"
	"github.com/echochat/backend/pkg/ws"
	"go.uber.org/zap"
)

var (
	ErrMessageNotFound = errors.New("消息不存在")
	ErrMessageRecalled = errors.New("消息已被撤回")
	ErrMessageDeleted  = errors.New("消息已被删除")
)

// MessageManageService 管理端消息管理服务
type MessageManageService struct {
	msgDAO  *adminDAO.MessageManageDAO
	userDAO *authDAO.UserDAO
	convDAO *imDAO.ConversationDAO
	pubSub  *ws.PubSub
}

// NewMessageManageService 创建 MessageManageService 实例
func NewMessageManageService(msgDAO *adminDAO.MessageManageDAO, userDAO *authDAO.UserDAO, convDAO *imDAO.ConversationDAO, pubSub *ws.PubSub) *MessageManageService {
	return &MessageManageService{
		msgDAO:  msgDAO,
		userDAO: userDAO,
		convDAO: convDAO,
		pubSub:  pubSub,
	}
}

// GetMessageList 获取消息列表（分页+筛选）
func (s *MessageManageService) GetMessageList(ctx context.Context, req *dto.AdminMessageListRequest) (*dto.AdminMessageListResponse, error) {
	funcName := "service.message_manage_service.GetMessageList"
	logs.Debug(ctx, funcName, "获取消息列表")

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	messages, total, err := s.msgDAO.ListMessages(ctx, req)
	if err != nil {
		return nil, err
	}

	senderIDs := make([]int64, 0, len(messages))
	senderSet := make(map[int64]bool)
	for _, m := range messages {
		if m.SenderID > 0 && !senderSet[m.SenderID] {
			senderIDs = append(senderIDs, m.SenderID)
			senderSet[m.SenderID] = true
		}
	}

	userMap := make(map[int64]struct {
		Nickname string
		Avatar   string
	})
	if len(senderIDs) > 0 {
		users, uErr := s.userDAO.FindByIDs(ctx, senderIDs)
		if uErr != nil {
			logs.Error(ctx, funcName, "批量查询用户信息失败", zap.Error(uErr))
		} else {
			for _, u := range users {
				userMap[u.ID] = struct {
					Nickname string
					Avatar   string
				}{Nickname: u.Nickname, Avatar: u.Avatar}
			}
		}
	}

	list := make([]dto.AdminMessageDTO, 0, len(messages))
	for _, m := range messages {
		item := dto.AdminMessageDTO{
			ID:             m.ID,
			ConversationID: m.ConversationID,
			SenderID:       m.SenderID,
			Type:           m.Type,
			TypeLabel:      constants.MessageTypeMap[m.Type],
			Content:        m.Content,
			Extra:          m.Extra,
			Status:         m.Status,
			StatusLabel:    constants.MessageStatusMap[m.Status],
			CreatedAt:      m.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if info, ok := userMap[m.SenderID]; ok {
			item.SenderNickname = info.Nickname
			item.SenderAvatar = info.Avatar
		}
		list = append(list, item)
	}

	return &dto.AdminMessageListResponse{
		Total:    total,
		List:     list,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// GetMessageDetail 获取消息详情
func (s *MessageManageService) GetMessageDetail(ctx context.Context, id int64) (*dto.AdminMessageDTO, error) {
	funcName := "service.message_manage_service.GetMessageDetail"
	logs.Debug(ctx, funcName, "获取消息详情", zap.Int64("id", id))

	msg, err := s.msgDAO.GetMessageByID(ctx, id)
	if err != nil {
		return nil, ErrMessageNotFound
	}

	item := &dto.AdminMessageDTO{
		ID:             msg.ID,
		ConversationID: msg.ConversationID,
		SenderID:       msg.SenderID,
		Type:           msg.Type,
		TypeLabel:      constants.MessageTypeMap[msg.Type],
		Content:        msg.Content,
		Extra:          msg.Extra,
		Status:         msg.Status,
		StatusLabel:    constants.MessageStatusMap[msg.Status],
		CreatedAt:      msg.CreatedAt.Format("2006-01-02 15:04:05"),
	}

	if msg.SenderID > 0 {
		users, uErr := s.userDAO.FindByIDs(ctx, []int64{msg.SenderID})
		if uErr == nil && len(users) > 0 {
			item.SenderNickname = users[0].Nickname
			item.SenderAvatar = users[0].Avatar
		}
	}

	return item, nil
}

// DeleteMessage 删除消息（软删除，状态改为已删除）
func (s *MessageManageService) DeleteMessage(ctx context.Context, id int64) error {
	funcName := "service.message_manage_service.DeleteMessage"
	logs.Info(ctx, funcName, "删除消息", zap.Int64("id", id))

	msg, err := s.msgDAO.GetMessageByID(ctx, id)
	if err != nil {
		return ErrMessageNotFound
	}
	if msg.Status == constants.MessageStatusDeleted {
		return ErrMessageDeleted
	}

	return s.msgDAO.UpdateMessageStatus(ctx, id, constants.MessageStatusDeleted)
}

// RecallMessage 管理员撤回消息（更新状态 + 推送 WS 通知给会话成员）
func (s *MessageManageService) RecallMessage(ctx context.Context, id int64) error {
	funcName := "service.message_manage_service.RecallMessage"
	logs.Info(ctx, funcName, "撤回消息", zap.Int64("id", id))

	msg, err := s.msgDAO.GetMessageByID(ctx, id)
	if err != nil {
		return ErrMessageNotFound
	}
	if msg.Status == constants.MessageStatusRecalled {
		return ErrMessageRecalled
	}
	if msg.Status == constants.MessageStatusDeleted {
		return ErrMessageDeleted
	}

	if err := s.msgDAO.UpdateMessageStatus(ctx, id, constants.MessageStatusRecalled); err != nil {
		return err
	}

	go s.pushRecallNotification(ctx, msg.ID, msg.ConversationID, msg.SenderID)

	return nil
}

// pushRecallNotification 向会话成员推送撤回通知
func (s *MessageManageService) pushRecallNotification(ctx context.Context, messageID, conversationID, senderID int64) {
	funcName := "service.message_manage_service.pushRecallNotification"

	memberIDs, err := s.convDAO.GetConversationMemberIDs(ctx, conversationID)
	if err != nil {
		logs.Error(ctx, funcName, "获取会话成员失败", zap.Error(err))
		return
	}

	pushData := map[string]interface{}{
		"message_id":      messageID,
		"conversation_id": conversationID,
		"operator_id":     int64(0),
		"sender_id":       senderID,
		"recall_text":     "管理员撤回了一条消息",
	}

	for _, uid := range memberIDs {
		pushMsg := ws.NewPushMessage("im.message.recalled", pushData)
		if err := s.pubSub.PublishToUser(ctx, uid, pushMsg); err != nil {
			logs.Error(ctx, funcName, "推送撤回通知失败", zap.Int64("user_id", uid), zap.Error(err))
		}
	}
}

// GetStats 获取消息统计数据
func (s *MessageManageService) GetStats(ctx context.Context, req *dto.AdminMessageStatsRequest) (*dto.AdminMessageStatsResponse, error) {
	funcName := "service.message_manage_service.GetStats"
	logs.Debug(ctx, funcName, "获取消息统计")

	days := req.Days
	if days <= 0 {
		days = 7
	}
	if days > 90 {
		days = 90
	}

	totalCount, err := s.msgDAO.GetTotalCount(ctx)
	if err != nil {
		return nil, err
	}

	todayCount, err := s.msgDAO.GetTodayCount(ctx)
	if err != nil {
		return nil, err
	}

	typeDist, err := s.msgDAO.GetTypeDistribution(ctx)
	if err != nil {
		return nil, err
	}
	for i := range typeDist {
		typeDist[i].Label = constants.MessageTypeMap[typeDist[i].Type]
	}

	dailyTrend, err := s.msgDAO.GetDailyTrend(ctx, days)
	if err != nil {
		return nil, err
	}

	activeUsers, err := s.msgDAO.GetActiveUsers(ctx, days)
	if err != nil {
		return nil, err
	}

	activeGroups, err := s.msgDAO.GetActiveGroups(ctx, days)
	if err != nil {
		return nil, err
	}

	return &dto.AdminMessageStatsResponse{
		TotalCount:       totalCount,
		TodayCount:       todayCount,
		TypeDistribution: typeDist,
		DailyTrend:       dailyTrend,
		ActiveUsers:      activeUsers,
		ActiveGroups:     activeGroups,
	}, nil
}
