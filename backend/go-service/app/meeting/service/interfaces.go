// Package service 提供 meeting 模块的业务逻辑
package service

import (
	"context"

	authModel "github.com/echochat/backend/app/auth/model"
	notifyService "github.com/echochat/backend/app/notify/service"
)

// NotifyPusher 通知推送接口
// 由 notify.service.NotifyService 隐式实现，供 meeting 模块注入使用
// 会议邀请、主持人变更、踢出会议等事件通过此接口落地通知中心
type NotifyPusher interface {
	Push(ctx context.Context, payload *notifyService.PushPayload)
	PushBatch(ctx context.Context, payloads []*notifyService.PushPayload)
}

// UserInfoResolver 查询用户昵称 / 头像的接口
// 由 contact.FriendshipDAO 隐式实现（已有 GetUsersByIDs 方法）
// 用于拉取会议邀请中主持人信息、成员列表头像等
type UserInfoResolver interface {
	GetUsersByIDs(ctx context.Context, userIDs []int64) ([]authModel.User, error)
}

// OnlineChecker 查询用户在线状态的接口
// 由 ws.OnlineService 隐式实现（签名对齐现有实现：不返回 error，查询失败按离线处理）
// 用于会议邀请时判断收件人是否在线（离线则降级为仅通知入库，上线时通过未读补偿推送）
type OnlineChecker interface {
	IsOnline(ctx context.Context, userID int64) bool
}
