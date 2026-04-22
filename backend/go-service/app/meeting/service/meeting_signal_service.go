// Package service 提供 meeting 模块的业务逻辑
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/echochat/backend/app/constants"
	"github.com/echochat/backend/app/meeting/dao"
	"github.com/echochat/backend/app/meeting/model"
	"github.com/echochat/backend/pkg/logs"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// MeetingSignalService WS 信令事件业务处理
// Task 6 落地：处理设计 §6.3 的 11 个 meeting.* 事件，分 3 组：
//   - 房间组（meeting.room.join/leave）：绑定 WS 连接 ↔ roomCode，辅助断连清理
//   - 成员组（meeting.member.state.changed）：麦/视频状态变更 + host 静音他人
//   - 媒体组（meeting.transport.*/produce.*/consume.*/producer.close）：
//     对接 MediaOrchestrator，实现 mediasoup signaling 桥接
//
// 所有方法统一返回 (ackData, error)：error 非 nil 表示业务失败，由 Handler 映射为 ACK code=-1
// 广播副作用（如 meeting.member.producer.new）在方法内部通过 broadcaster 发出，不进 ACK
type MeetingSignalService struct {
	roomDAO        *dao.MeetingRoomDAO
	participantDAO *dao.MeetingParticipantDAO
	redis          *redis.Client

	broadcaster       *MeetingBroadcaster
	mediaOrchestrator MediaOrchestrator
	lifecycleSvc      *MeetingLifecycleService
}

// NewMeetingSignalService 构造 WS 信令服务
// Task 8 注入 lifecycleSvc：host 掉线 / 重连的生命周期联动
func NewMeetingSignalService(
	roomDAO *dao.MeetingRoomDAO,
	participantDAO *dao.MeetingParticipantDAO,
	redis *redis.Client,
	broadcaster *MeetingBroadcaster,
	mediaOrchestrator MediaOrchestrator,
	lifecycleSvc *MeetingLifecycleService,
) *MeetingSignalService {
	return &MeetingSignalService{
		roomDAO:           roomDAO,
		participantDAO:    participantDAO,
		redis:             redis,
		broadcaster:       broadcaster,
		mediaOrchestrator: mediaOrchestrator,
		lifecycleSvc:      lifecycleSvc,
	}
}

// Redis 资源追踪 key（设计 §九 - 断线清理）
// Set 内元素格式："transport:{id}" / "producer:{id}" / "consumer:{id}"
func resourceTrackKey(roomCode string, userID int64) string {
	return fmt.Sprintf("echo:meeting:resource:%s:%d", roomCode, userID)
}

// resourceTTL 单个用户资源追踪集合 TTL
// 设计：会议期间维持可达即可；若用户长期不活跃由断线清理接管
const resourceTTL = time.Hour

// trackResource 记录用户在会议中持有的媒体资源 ID
func (s *MeetingSignalService) trackResource(ctx context.Context, roomCode string, userID int64, kind, id string) {
	key := resourceTrackKey(roomCode, userID)
	member := kind + ":" + id
	if err := s.redis.SAdd(ctx, key, member).Err(); err != nil {
		logs.Warn(ctx, "service.meeting_signal_service.trackResource", "追踪媒体资源失败",
			zap.String("key", key), zap.String("member", member), zap.Error(err))
		return
	}
	_ = s.redis.Expire(ctx, key, resourceTTL).Err()
}

// untrackResource 从集合中移除资源 ID（关闭 producer/consumer 时）
func (s *MeetingSignalService) untrackResource(ctx context.Context, roomCode string, userID int64, kind, id string) {
	key := resourceTrackKey(roomCode, userID)
	member := kind + ":" + id
	_ = s.redis.SRem(ctx, key, member).Err()
}

// loadRoomAndParticipant 通用前置校验：拉取房间 + 确认用户是活跃参会者
// 所有信令事件在进入业务前都要过这一关；返回的 *MeetingRoom 供后续广播使用 roomID
func (s *MeetingSignalService) loadRoomAndParticipant(ctx context.Context, roomCode string, userID int64) (*model.MeetingRoom, error) {
	room, err := s.roomDAO.GetByCode(ctx, roomCode)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, ErrMeetingNotFound
	}
	if room.Status == constants.MeetingStatusEnded {
		return nil, ErrMeetingEnded
	}
	p, err := s.participantDAO.GetByRoomAndUser(ctx, room.ID, userID)
	if err != nil {
		return nil, err
	}
	if p == nil || !p.IsActive() {
		return nil, ErrNotInMeeting
	}
	return room, nil
}

// ========== 房间组（2 个 C→S）==========

// OnRoomJoin 处理 meeting.room.join 事件
// 语义：客户端 REST 加入会议成功后，通过 WS 宣告在线；服务端记录 userID ↔ roomCode 映射
// 仅做存在性校验 + 心跳意义上的资源 key 刷新，不产生副作用
func (s *MeetingSignalService) OnRoomJoin(ctx context.Context, userID int64, roomCode string) error {
	room, err := s.loadRoomAndParticipant(ctx, roomCode, userID)
	if err != nil {
		return err
	}
	// 触发资源追踪 key 续期（空集合 TTL 续期无副作用）
	key := resourceTrackKey(roomCode, userID)
	_ = s.redis.Expire(ctx, key, resourceTTL).Err()

	// Task 8：若本次 WS 在线的用户是会议 host，尝试撤销 host 宽限期
	// 本方法幂等：若此前无 host_grace key（正常场景）DEL 返回 0，不产生副作用
	if s.lifecycleSvc != nil && room.HostID == userID {
		s.lifecycleSvc.OnHostReconnect(ctx, roomCode, userID)
	}

	logs.Info(ctx, "service.meeting_signal_service.OnRoomJoin", "用户宣告 WS 在线",
		zap.String("room_code", roomCode),
		zap.Int64("user_id", userID),
		zap.Int64("room_id", room.ID))
	return nil
}

// OnWSDisconnect WS 断线钩子，实现 ws.MeetingDisconnectHook 接口（Task 8）
// 由 ws.handler 的 SetOnDisconnect 回调触发；场景：用户的最后一条 WS 连接被移除
// 职责：
//  1. 若该用户当前存在活跃 meeting_participants 记录：清理其媒体资源 + 若是 host 启动 host 宽限期
//  2. 普通成员：仅清媒体资源（设计 Q1=A：不动 meeting_participants，长期不活跃由 4h 兜底清理）
//
// 容错：所有异常仅 Warn 日志，不返回 error；保证 ws.handler 主断连流程不受影响
func (s *MeetingSignalService) OnWSDisconnect(ctx context.Context, userID int64) {
	funcName := "service.meeting_signal_service.OnWSDisconnect"

	active, err := s.participantDAO.FindActiveByUser(ctx, userID)
	if err != nil {
		logs.Warn(ctx, funcName, "查询活跃会议失败", zap.Int64("user_id", userID), zap.Error(err))
		return
	}
	if active == nil {
		// 用户当前不在任何会议
		return
	}
	room, err := s.roomDAO.GetByID(ctx, active.RoomID)
	if err != nil || room == nil {
		logs.Warn(ctx, funcName, "加载 room 失败或已不存在",
			zap.Int64("room_id", active.RoomID), zap.Error(err))
		return
	}
	if room.Status == constants.MeetingStatusEnded {
		return
	}

	s.cleanupUserResources(ctx, room.RoomCode, userID)

	if s.lifecycleSvc != nil && room.HostID == userID {
		s.lifecycleSvc.OnHostDisconnect(ctx, room.RoomCode, userID)
	}

	logs.Info(ctx, funcName, "WS 断线已处理",
		zap.String("room_code", room.RoomCode),
		zap.Int64("user_id", userID),
		zap.Bool("is_host", room.HostID == userID))
}

// OnRoomLeave 处理 meeting.room.leave 事件
// 语义：WS 层面的主动离会（等价 REST leave 但不强制要求落库事务；
// 当前实现：仅清理该用户在本会议的所有媒体资源（batch close producer/consumer）+ 广播 meeting.member.left
// 参会者表的 LeaveRoom 逻辑仍由 REST API 负责（避免 WS 并发引起 left_at 重复写入）
func (s *MeetingSignalService) OnRoomLeave(ctx context.Context, userID int64, roomCode string) error {
	room, err := s.loadRoomAndParticipant(ctx, roomCode, userID)
	if err != nil {
		return err
	}
	s.cleanupUserResources(ctx, roomCode, userID)

	go s.broadcaster.BroadcastToMeeting(context.Background(), room.ID, constants.MeetingWSEventMemberLeft, map[string]interface{}{
		"room_code": roomCode,
		"user_id":   userID,
		"reason":    "ws_disconnect",
	}, userID)
	return nil
}

// ========== 成员组（1 个双向）==========

// MemberStateChangePayload meeting.member.state.changed 请求载荷
// host 可通过 target_user_id 静音他人 / 关其摄像头；非 host 传该字段将被拒绝
type MemberStateChangePayload struct {
	RoomCode     string `json:"room_code"`
	TargetUserID int64  `json:"target_user_id,omitempty"` // 可选：host 强制他人状态
	AudioEnabled *bool  `json:"audio_enabled,omitempty"`  // nil 表示不改
	VideoEnabled *bool  `json:"video_enabled,omitempty"`
}

// OnMemberStateChanged 处理 meeting.member.state.changed 事件
// 权限：操作自己无限制；操作他人必须是 host
// 行为：广播 meeting.member.state.changed 给房间其他成员（发起者自己不收到回显）
func (s *MeetingSignalService) OnMemberStateChanged(ctx context.Context, fromUserID int64, payload *MemberStateChangePayload) error {
	room, err := s.loadRoomAndParticipant(ctx, payload.RoomCode, fromUserID)
	if err != nil {
		return err
	}

	targetID := fromUserID
	if payload.TargetUserID != 0 && payload.TargetUserID != fromUserID {
		if room.HostID != fromUserID {
			return ErrNotMeetingHost
		}
		targetP, err := s.participantDAO.GetByRoomAndUser(ctx, room.ID, payload.TargetUserID)
		if err != nil {
			return err
		}
		if targetP == nil || !targetP.IsActive() {
			return ErrTransferTargetInvalid
		}
		targetID = payload.TargetUserID
	}

	data := map[string]interface{}{
		"room_code":       payload.RoomCode,
		"user_id":         targetID,
		"changed_by":      fromUserID,
	}
	if payload.AudioEnabled != nil {
		data["audio_enabled"] = *payload.AudioEnabled
	}
	if payload.VideoEnabled != nil {
		data["video_enabled"] = *payload.VideoEnabled
	}

	go s.broadcaster.BroadcastToMeeting(context.Background(), room.ID, constants.MeetingWSEventMemberStateChange, data, fromUserID)
	return nil
}

// ========== 媒体组（5 个，mediasoup signaling）==========

// TransportCreatePayload meeting.transport.create 请求载荷
type TransportCreatePayload struct {
	RoomCode  string `json:"room_code"`
	Direction string `json:"direction"` // "send" | "recv"
}

// OnTransportCreate 处理 meeting.transport.create 事件
func (s *MeetingSignalService) OnTransportCreate(ctx context.Context, userID int64, payload *TransportCreatePayload) (*TransportInfo, error) {
	if payload.Direction != "send" && payload.Direction != "recv" {
		return nil, fmt.Errorf("direction 非法，必须是 send 或 recv")
	}
	if _, err := s.loadRoomAndParticipant(ctx, payload.RoomCode, userID); err != nil {
		return nil, err
	}
	info, err := s.mediaOrchestrator.CreateTransport(ctx, &CreateTransportReq{
		RoomCode:  payload.RoomCode,
		UserID:    userID,
		Direction: payload.Direction,
	})
	if err != nil {
		return nil, err
	}
	s.trackResource(ctx, payload.RoomCode, userID, "transport", info.ID)
	return info, nil
}

// TransportConnectPayload meeting.transport.connect 请求载荷
type TransportConnectPayload struct {
	RoomCode       string          `json:"room_code"`
	TransportID    string          `json:"transport_id"`
	DtlsParameters json.RawMessage `json:"dtls_parameters"`
}

// OnTransportConnect 处理 meeting.transport.connect 事件
func (s *MeetingSignalService) OnTransportConnect(ctx context.Context, userID int64, payload *TransportConnectPayload) error {
	if payload.TransportID == "" {
		return fmt.Errorf("transport_id 不能为空")
	}
	if _, err := s.loadRoomAndParticipant(ctx, payload.RoomCode, userID); err != nil {
		return err
	}
	return s.mediaOrchestrator.ConnectTransport(ctx, payload.TransportID, payload.DtlsParameters)
}

// ProduceStartPayload meeting.produce.start 请求载荷
type ProduceStartPayload struct {
	RoomCode      string          `json:"room_code"`
	TransportID   string          `json:"transport_id"`
	Kind          string          `json:"kind"` // "audio" | "video"
	RtpParameters json.RawMessage `json:"rtp_parameters"`
}

// ProduceStartResult 返回给客户端的 producerID
type ProduceStartResult struct {
	ProducerID string `json:"producer_id"`
}

// OnProduceStart 处理 meeting.produce.start 事件
// 成功后广播 meeting.member.producer.new 给房间内其他成员，驱动对端自动创建 Consumer
func (s *MeetingSignalService) OnProduceStart(ctx context.Context, userID int64, payload *ProduceStartPayload) (*ProduceStartResult, error) {
	if payload.Kind != "audio" && payload.Kind != "video" {
		return nil, fmt.Errorf("kind 非法，必须是 audio 或 video")
	}
	if payload.TransportID == "" {
		return nil, fmt.Errorf("transport_id 不能为空")
	}
	room, err := s.loadRoomAndParticipant(ctx, payload.RoomCode, userID)
	if err != nil {
		return nil, err
	}
	producerID, err := s.mediaOrchestrator.CreateProducer(ctx, &CreateProducerReq{
		RoomCode:      payload.RoomCode,
		UserID:        userID,
		TransportID:   payload.TransportID,
		Kind:          payload.Kind,
		RtpParameters: payload.RtpParameters,
	})
	if err != nil {
		return nil, err
	}
	s.trackResource(ctx, payload.RoomCode, userID, "producer", producerID)

	go s.broadcaster.BroadcastToMeeting(context.Background(), room.ID, constants.MeetingWSEventMemberProducerNew, map[string]interface{}{
		"room_code":   payload.RoomCode,
		"user_id":     userID,
		"producer_id": producerID,
		"kind":        payload.Kind,
	}, userID)

	return &ProduceStartResult{ProducerID: producerID}, nil
}

// ConsumeStartPayload meeting.consume.start 请求载荷
type ConsumeStartPayload struct {
	RoomCode        string          `json:"room_code"`
	TransportID     string          `json:"transport_id"` // 客户端 recv Transport
	ProducerID      string          `json:"producer_id"`  // 要订阅的远端 Producer
	RtpCapabilities json.RawMessage `json:"rtp_capabilities"`
}

// OnConsumeStart 处理 meeting.consume.start 事件
func (s *MeetingSignalService) OnConsumeStart(ctx context.Context, userID int64, payload *ConsumeStartPayload) (*ConsumerInfo, error) {
	if payload.TransportID == "" || payload.ProducerID == "" {
		return nil, fmt.Errorf("transport_id 与 producer_id 均不能为空")
	}
	if _, err := s.loadRoomAndParticipant(ctx, payload.RoomCode, userID); err != nil {
		return nil, err
	}
	info, err := s.mediaOrchestrator.CreateConsumer(ctx, &CreateConsumerReq{
		RoomCode:        payload.RoomCode,
		UserID:          userID,
		TransportID:     payload.TransportID,
		ProducerID:      payload.ProducerID,
		RtpCapabilities: payload.RtpCapabilities,
	})
	if err != nil {
		return nil, err
	}
	s.trackResource(ctx, payload.RoomCode, userID, "consumer", info.ID)
	return info, nil
}

// ConsumeResumePayload meeting.consume.resume 请求载荷（Task 9）
// 前端 recv Transport 与 track 就绪后发送，用于把 Node 侧 paused Consumer 切到 active
type ConsumeResumePayload struct {
	RoomCode   string `json:"room_code"`
	ConsumerID string `json:"consumer_id"`
}

// OnConsumeResume 处理 meeting.consume.resume 事件（Task 9）
// 语义：告知 media-server 把指定 Consumer 从 paused 切到 active
// 权限：仅当 userID 是会议活跃成员且 consumerID 归属该用户时允许
// 幂等：Node 对已 active Consumer 再次 resume 不报错；Consumer 不存在则 ACK 返回友好错误
func (s *MeetingSignalService) OnConsumeResume(ctx context.Context, userID int64, payload *ConsumeResumePayload) error {
	if payload.ConsumerID == "" {
		return fmt.Errorf("consumer_id 不能为空")
	}
	if _, err := s.loadRoomAndParticipant(ctx, payload.RoomCode, userID); err != nil {
		return err
	}
	if err := s.mediaOrchestrator.ResumeConsumer(ctx, payload.ConsumerID); err != nil {
		logs.Warn(ctx, "service.meeting_signal_service.OnConsumeResume", "恢复 Consumer 失败",
			zap.String("room_code", payload.RoomCode),
			zap.Int64("user_id", userID),
			zap.String("consumer_id", payload.ConsumerID),
			zap.Error(err))
		return err
	}
	return nil
}

// ProducerClosePayload meeting.producer.close 请求载荷
type ProducerClosePayload struct {
	RoomCode   string `json:"room_code"`
	ProducerID string `json:"producer_id"`
}

// OnProducerClose 处理 meeting.producer.close 事件
// 成功后广播给房间内其他成员（与 mediasoup 的 producerclose 级联动作平级）
func (s *MeetingSignalService) OnProducerClose(ctx context.Context, userID int64, payload *ProducerClosePayload) error {
	if payload.ProducerID == "" {
		return fmt.Errorf("producer_id 不能为空")
	}
	room, err := s.loadRoomAndParticipant(ctx, payload.RoomCode, userID)
	if err != nil {
		return err
	}
	if err := s.mediaOrchestrator.CloseProducer(ctx, payload.ProducerID); err != nil {
		logs.Warn(ctx, "service.meeting_signal_service.OnProducerClose", "关闭 Producer 失败",
			zap.String("producer_id", payload.ProducerID), zap.Error(err))
	}
	s.untrackResource(ctx, payload.RoomCode, userID, "producer", payload.ProducerID)

	go s.broadcaster.BroadcastToMeeting(context.Background(), room.ID, constants.MeetingWSEventMemberProducerNew, map[string]interface{}{
		"room_code":   payload.RoomCode,
		"user_id":     userID,
		"producer_id": payload.ProducerID,
		"closed":      true,
	}, userID)
	return nil
}

// ========== 资源清理 ==========

// cleanupUserResources 批量关闭指定用户在某会议的所有媒体资源
// WS 断开、主动离会、被踢时使用；依赖 Redis 集合中追踪的资源 ID
func (s *MeetingSignalService) cleanupUserResources(ctx context.Context, roomCode string, userID int64) {
	funcName := "service.meeting_signal_service.cleanupUserResources"

	key := resourceTrackKey(roomCode, userID)
	members, err := s.redis.SMembers(ctx, key).Result()
	if err != nil {
		logs.Warn(ctx, funcName, "读取资源追踪集合失败", zap.String("key", key), zap.Error(err))
		return
	}
	for _, m := range members {
		// 格式："kind:id"
		idx := -1
		for i, c := range m {
			if c == ':' {
				idx = i
				break
			}
		}
		if idx < 0 {
			continue
		}
		kind, id := m[:idx], m[idx+1:]
		switch kind {
		case "producer":
			_ = s.mediaOrchestrator.CloseProducer(ctx, id)
		case "consumer":
			_ = s.mediaOrchestrator.CloseConsumer(ctx, id)
			// transport 关闭一般由 Router 级联；这里不单独处理
		}
	}
	_ = s.redis.Del(ctx, key).Err()

	if len(members) > 0 {
		logs.Info(ctx, funcName, "清理用户媒体资源",
			zap.String("room_code", roomCode),
			zap.Int64("user_id", userID),
			zap.Int("resource_count", len(members)))
	}
}
