// Package service 提供 meeting 模块的业务逻辑
package service

import (
	"context"
	"encoding/json"

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

// ====== MediaOrchestrator：Go → Node media-server 统一封装（设计 §6.6）======

// TransportInfo Node 端创建的 Transport 元信息
// 字段与 mediasoup-client 的 Device.createSendTransport / createRecvTransport 入参兼容
type TransportInfo struct {
	ID             string          `json:"id"`              // Transport ID
	IceParameters  json.RawMessage `json:"iceParameters"`   // mediasoup ICE 参数
	IceCandidates  json.RawMessage `json:"iceCandidates"`   // ICE 候选列表
	DtlsParameters json.RawMessage `json:"dtlsParameters"`  // DTLS 指纹参数
	SctpParameters json.RawMessage `json:"sctpParameters,omitempty"`
}

// ConsumerInfo Node 端创建的 Consumer 元信息
type ConsumerInfo struct {
	ID             string          `json:"id"`
	ProducerID     string          `json:"producerId"`
	Kind           string          `json:"kind"`           // "audio" | "video"
	RtpParameters  json.RawMessage `json:"rtpParameters"`
	Type           string          `json:"type,omitempty"` // "simple" | "simulcast" | "svc"
	ProducerPaused bool            `json:"producerPaused,omitempty"`
}

// CreateTransportReq 创建 Transport 请求
type CreateTransportReq struct {
	RoomCode  string `json:"roomCode"`
	UserID    int64  `json:"userId"`
	Direction string `json:"direction"` // "send" | "recv"
}

// CreateProducerReq 创建 Producer 请求
type CreateProducerReq struct {
	RoomCode      string          `json:"roomCode"`
	UserID        int64           `json:"userId"`
	TransportID   string          `json:"transportId"`
	Kind          string          `json:"kind"`          // "audio" | "video"
	RtpParameters json.RawMessage `json:"rtpParameters"` // 直接转发给 Node，由其做结构校验
}

// CreateConsumerReq 创建 Consumer 请求
type CreateConsumerReq struct {
	RoomCode        string          `json:"roomCode"`
	UserID          int64           `json:"userId"`
	TransportID     string          `json:"transportId"`
	ProducerID      string          `json:"producerId"`
	RtpCapabilities json.RawMessage `json:"rtpCapabilities"`
}

// MediaOrchestrator 媒体服务器编排接口（设计 §6.6 NodeClient）
// Task 7 (2026-04-21) 起默认实现为 HTTPMediaOrchestrator（通过 X-Internal-Token
// 调用 Node media-server 的 /internal/v1/* REST API）。
// NoopMediaOrchestrator 仅保留用于本地调试（无 Node 服务时可临时切换）：
//   - CreateRouter 返回 "noop-router-{code}"；其他方法返回可解析的占位数据
//   - 所有方法均幂等：重复调用不报错，符合 WS 信令重试语义
type MediaOrchestrator interface {
	// CreateRouter 为会议房间创建 mediasoup Router
	CreateRouter(ctx context.Context, roomCode string) (routerID string, err error)
	// CloseRouter 关闭房间对应 Router 及其下所有资源（幂等）
	CloseRouter(ctx context.Context, roomCode string) error

	// CreateTransport 为用户创建 send/recv WebRTC Transport
	CreateTransport(ctx context.Context, req *CreateTransportReq) (*TransportInfo, error)
	// ConnectTransport Transport DTLS 握手（幂等：重复 connect 对已连接 transport 视为成功）
	ConnectTransport(ctx context.Context, transportID string, dtlsParameters json.RawMessage) error

	// CreateProducer 在指定 send Transport 上创建 Producer
	CreateProducer(ctx context.Context, req *CreateProducerReq) (producerID string, err error)
	// CloseProducer 关闭指定 Producer（幂等）
	CloseProducer(ctx context.Context, producerID string) error

	// CreateConsumer 在指定 recv Transport 上创建 Consumer（订阅远端 Producer）
	CreateConsumer(ctx context.Context, req *CreateConsumerReq) (*ConsumerInfo, error)
	// CloseConsumer 关闭指定 Consumer（幂等）
	CloseConsumer(ctx context.Context, consumerID string) error
}

// NoopMediaOrchestrator 本地调试占位实现（Task 7 后默认不再使用）
// 返回伪造的 ID 与固定占位数据（JSON：空对象 / 空数组），所有操作仅写日志不调用 Node
// Task 7 已将全局 wire 绑定切换为 HTTPMediaOrchestrator，此类型仅作为无 Node 服务时
// 的临时回退（需手动修改 provider.go 的 wire.Bind 才会生效）
type NoopMediaOrchestrator struct{}

// NewNoopMediaOrchestrator 构造占位的 MediaOrchestrator
func NewNoopMediaOrchestrator() *NoopMediaOrchestrator {
	return &NoopMediaOrchestrator{}
}

// CreateRouter 返回以 "noop-router-" 为前缀的伪造 RouterID
func (n *NoopMediaOrchestrator) CreateRouter(_ context.Context, roomCode string) (string, error) {
	return "noop-router-" + roomCode, nil
}

// CloseRouter 占位实现：直接返回 nil
func (n *NoopMediaOrchestrator) CloseRouter(_ context.Context, _ string) error {
	return nil
}

// CreateTransport 占位：返回以 "noop-transport-" 为前缀的伪造 ID，带最小合法 JSON 结构
func (n *NoopMediaOrchestrator) CreateTransport(_ context.Context, req *CreateTransportReq) (*TransportInfo, error) {
	return &TransportInfo{
		ID:             "noop-transport-" + req.Direction,
		IceParameters:  json.RawMessage(`{}`),
		IceCandidates:  json.RawMessage(`[]`),
		DtlsParameters: json.RawMessage(`{}`),
	}, nil
}

// ConnectTransport 占位：直接返回 nil
func (n *NoopMediaOrchestrator) ConnectTransport(_ context.Context, _ string, _ json.RawMessage) error {
	return nil
}

// CreateProducer 占位：返回伪造 ID
func (n *NoopMediaOrchestrator) CreateProducer(_ context.Context, req *CreateProducerReq) (string, error) {
	return "noop-producer-" + req.Kind, nil
}

// CloseProducer 占位：直接返回 nil
func (n *NoopMediaOrchestrator) CloseProducer(_ context.Context, _ string) error {
	return nil
}

// CreateConsumer 占位：返回伪造的 ConsumerInfo（含目标 producerID 回填）
func (n *NoopMediaOrchestrator) CreateConsumer(_ context.Context, req *CreateConsumerReq) (*ConsumerInfo, error) {
	return &ConsumerInfo{
		ID:            "noop-consumer-" + req.ProducerID,
		ProducerID:    req.ProducerID,
		Kind:          "video",
		RtpParameters: json.RawMessage(`{}`),
		Type:          "simple",
	}, nil
}

// CloseConsumer 占位：直接返回 nil
func (n *NoopMediaOrchestrator) CloseConsumer(_ context.Context, _ string) error {
	return nil
}
