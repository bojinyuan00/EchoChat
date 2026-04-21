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

// MediaOrchestrator 媒体服务器编排接口
// Task 7 落地 Go → Node media-server HTTP Client 后由 node_client.NodeClient 实现
// Task 5 阶段使用 NoopMediaOrchestrator 占位，仅返回假的 RouterID，不产生真实媒体资源
// 语义：会议房间在 Go 侧落库后，通过该接口驱动 Node 端 mediasoup Router 创建与销毁
type MediaOrchestrator interface {
	// CreateRouter 为会议房间创建 mediasoup Router
	// 入参：room_code 作为 Node 端聚合键；返回 Router ID 与推荐编解码参数（Task 7 对接时补充）
	CreateRouter(ctx context.Context, roomCode string) (routerID string, err error)

	// CloseRouter 关闭会议房间对应的 mediasoup Router 及其下所有 transport/producer/consumer
	// 幂等：重复关闭不返回错误
	CloseRouter(ctx context.Context, roomCode string) error
}

// NoopMediaOrchestrator 占位实现：Task 5 完成生命周期接口时使用
// 返回伪造的 RouterID，所有操作仅写日志不调用 Node
// Task 7 完成后全局 wire 切换到真实 NodeClient 实现
type NoopMediaOrchestrator struct{}

// NewNoopMediaOrchestrator 构造占位的 MediaOrchestrator
func NewNoopMediaOrchestrator() *NoopMediaOrchestrator {
	return &NoopMediaOrchestrator{}
}

// CreateRouter 返回以 "noop-router-" 为前缀的伪造 RouterID
// 调用方可据此区分真实 / 占位实现，便于调试与切换
func (n *NoopMediaOrchestrator) CreateRouter(_ context.Context, roomCode string) (string, error) {
	return "noop-router-" + roomCode, nil
}

// CloseRouter 占位实现：直接返回 nil
func (n *NoopMediaOrchestrator) CloseRouter(_ context.Context, _ string) error {
	return nil
}
