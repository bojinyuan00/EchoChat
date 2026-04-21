// Package service 提供 meeting 模块的业务逻辑
package service

import (
	"context"

	"github.com/echochat/backend/app/meeting/dao"
	"github.com/echochat/backend/pkg/logs"
	"github.com/echochat/backend/pkg/ws"
	"go.uber.org/zap"
)

// MeetingBroadcaster 会议 WS 广播中枢
// Task 6 引入，替代原 MeetingService.broadcastToActiveParticipants 内联实现
// 语义：
//   - BroadcastToMeeting(ctx, roomID, event, payload, excludeUserIDs...) 向房间内所有活跃成员广播
//   - PublishToUser(ctx, userID, event, payload) 定向推送给指定用户（如 meeting.member.kicked 定向通知被踢者）
// 通过 PubSub.Publish 跨实例传递；本地 Hub 若订阅了目标用户频道则自动路由到 WS 连接
type MeetingBroadcaster struct {
	participantDAO *dao.MeetingParticipantDAO
	pubsub         *ws.PubSub
}

// NewMeetingBroadcaster 构造广播中枢
func NewMeetingBroadcaster(participantDAO *dao.MeetingParticipantDAO, pubsub *ws.PubSub) *MeetingBroadcaster {
	return &MeetingBroadcaster{
		participantDAO: participantDAO,
		pubsub:         pubsub,
	}
}

// BroadcastToMeeting 向房间内所有活跃参会者广播 WS 事件
// excludeUserIDs 中的用户将被跳过（通常排除发送者本人，避免"自己收到自己的事件"）
// 非阻塞：单个用户发送失败只记录 WARN 日志，不中断循环
func (b *MeetingBroadcaster) BroadcastToMeeting(ctx context.Context, roomID int64, event string, data interface{}, excludeUserIDs ...int64) {
	funcName := "service.meeting_broadcaster.BroadcastToMeeting"

	participants, err := b.participantDAO.ListActiveByRoom(ctx, roomID)
	if err != nil {
		logs.Warn(ctx, funcName, "拉取活跃参会者失败",
			zap.Int64("room_id", roomID),
			zap.String("event", event),
			zap.Error(err))
		return
	}
	if len(participants) == 0 {
		return
	}

	exclude := make(map[int64]struct{}, len(excludeUserIDs))
	for _, id := range excludeUserIDs {
		exclude[id] = struct{}{}
	}

	msg := ws.NewPushMessage(event, data)
	pushed := 0
	for _, p := range participants {
		if _, skip := exclude[p.UserID]; skip {
			continue
		}
		if err := b.pubsub.PublishToUser(ctx, p.UserID, msg); err != nil {
			logs.Warn(ctx, funcName, "WS 广播单用户失败",
				zap.Int64("user_id", p.UserID),
				zap.String("event", event),
				zap.Error(err))
			continue
		}
		pushed++
	}

	logs.Debug(ctx, funcName, "会议事件广播完成",
		zap.Int64("room_id", roomID),
		zap.String("event", event),
		zap.Int("pushed", pushed),
		zap.Int("total_active", len(participants)))
}

// PublishToUser 定向推送给单个用户
// 场景：meeting.member.kicked（被踢者收到的定向通知）、ACK 补偿推送等
// 返回 error 让调用方根据业务语义决定是否需要感知失败
func (b *MeetingBroadcaster) PublishToUser(ctx context.Context, userID int64, event string, data interface{}) error {
	msg := ws.NewPushMessage(event, data)
	if err := b.pubsub.PublishToUser(ctx, userID, msg); err != nil {
		logs.Warn(ctx, "service.meeting_broadcaster.PublishToUser", "定向 WS 推送失败",
			zap.Int64("user_id", userID),
			zap.String("event", event),
			zap.Error(err))
		return err
	}
	return nil
}
