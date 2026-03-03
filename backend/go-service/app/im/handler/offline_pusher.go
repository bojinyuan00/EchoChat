package handler

import (
	"context"

	"github.com/echochat/backend/app/im/dao"
	"github.com/echochat/backend/app/im/service"
	"github.com/echochat/backend/pkg/logs"
	"github.com/echochat/backend/pkg/ws"
	"go.uber.org/zap"
)

// OfflinePusher 离线消息推送器
// 在用户 WebSocket 重新连接后，主动推送未读会话摘要
type OfflinePusher struct {
	imService *service.IMService
	convDAO   *dao.ConversationDAO
	pubsub    *ws.PubSub
}

// NewOfflinePusher 创建 OfflinePusher 实例
func NewOfflinePusher(imService *service.IMService, convDAO *dao.ConversationDAO, pubsub *ws.PubSub) *OfflinePusher {
	return &OfflinePusher{
		imService: imService,
		convDAO:   convDAO,
		pubsub:    pubsub,
	}
}

// PushOfflineMessages 向用户推送离线消息摘要
// 包含所有有未读消息的会话信息和全局未读总数
func (p *OfflinePusher) PushOfflineMessages(ctx context.Context, userID int64) {
	funcName := "handler.offline_pusher.PushOfflineMessages"
	logs.Info(ctx, funcName, "推送离线消息", zap.Int64("user_id", userID))

	unreadConvs, err := p.convDAO.GetUnreadConversations(ctx, userID)
	if err != nil {
		logs.Error(ctx, funcName, "查询未读会话失败", zap.Error(err))
		return
	}

	totalUnread, err := p.imService.GetTotalUnread(ctx, userID)
	if err != nil {
		logs.Error(ctx, funcName, "查询全局未读数失败", zap.Error(err))
		totalUnread = 0
	}

	if len(unreadConvs) == 0 && totalUnread == 0 {
		return
	}

	convList := make([]map[string]interface{}, 0, len(unreadConvs))
	for _, c := range unreadConvs {
		item := map[string]interface{}{
			"conversation_id": c.ID,
			"unread_count":    c.UnreadCount,
			"last_msg_content": c.LastMsgContent,
		}
		if c.LastMsgTime != nil {
			item["last_msg_time"] = c.LastMsgTime.Format("2006-01-02 15:04:05")
		}
		convList = append(convList, item)
	}

	push := ws.NewPushMessage("im.offline.sync", map[string]interface{}{
		"total_unread":  totalUnread,
		"conversations": convList,
	})
	data, err := ws.MarshalPush(push)
	if err != nil {
		logs.Error(ctx, funcName, "序列化离线消息失败", zap.Error(err))
		return
	}

	if err := p.pubsub.Publish(ctx, userID, data); err != nil {
		logs.Error(ctx, funcName, "推送离线消息失败",
			zap.Int64("user_id", userID), zap.Error(err))
	}

	logs.Info(ctx, funcName, "离线消息推送完成",
		zap.Int64("user_id", userID),
		zap.Int("unread_conv_count", len(unreadConvs)),
		zap.Int64("total_unread", totalUnread))
}
