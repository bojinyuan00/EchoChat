package service

import (
	"context"
	"errors"

	"github.com/echochat/backend/app/meeting/dao"
	"github.com/echochat/backend/app/meeting/model"
	"github.com/echochat/backend/pkg/ws"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Task 5 会继续填充具体业务逻辑时使用的哨兵错误，此处先集中定义占位
// 命名与 group/notify 模块的 Err* 风格一致，控制器可直接 errors.Is 识别
var (
	ErrMeetingNotFound      = errors.New("会议不存在")
	ErrMeetingEnded         = errors.New("会议已结束")
	ErrMeetingFull          = errors.New("会议已满员")
	ErrMeetingPasswordWrong = errors.New("会议密码错误")
	ErrNotInMeeting         = errors.New("你不在此会议中")
	ErrAlreadyInOtherMeeting = errors.New("你当前已在其他会议中")
	ErrNotMeetingHost       = errors.New("仅主持人可执行此操作")
	ErrInviteTokenInvalid   = errors.New("邀请链接已失效")
	ErrRoomCodeConflict     = errors.New("会议号生成冲突，请重试")
	ErrNotImplemented       = errors.New("功能尚未实现")
)

// MeetingService 会议业务服务
// Task 4 骨架阶段：仅完成依赖组装和方法签名占位，具体业务逻辑留待 Task 5/6/7 填充
// 方法返回 ErrNotImplemented，Controller 将其映射为 501 响应，便于 Postman 与前端联调前观察路由完整性
type MeetingService struct {
	roomDAO        *dao.MeetingRoomDAO
	participantDAO *dao.MeetingParticipantDAO
	chatDAO        *dao.MeetingChatDAO

	db     *gorm.DB
	redis  *redis.Client
	pubsub *ws.PubSub

	notifyPusher  NotifyPusher
	userResolver  UserInfoResolver
	onlineChecker OnlineChecker
}

// NewMeetingService 创建 MeetingService 实例
// 依赖通过构造函数注入，接口依赖由上游 Wire 绑定到具体实现
func NewMeetingService(
	roomDAO *dao.MeetingRoomDAO,
	participantDAO *dao.MeetingParticipantDAO,
	chatDAO *dao.MeetingChatDAO,
	db *gorm.DB,
	redis *redis.Client,
	pubsub *ws.PubSub,
	notifyPusher NotifyPusher,
	userResolver UserInfoResolver,
	onlineChecker OnlineChecker,
) *MeetingService {
	return &MeetingService{
		roomDAO:        roomDAO,
		participantDAO: participantDAO,
		chatDAO:        chatDAO,
		db:             db,
		redis:          redis,
		pubsub:         pubsub,
		notifyPusher:   notifyPusher,
		userResolver:   userResolver,
		onlineChecker:  onlineChecker,
	}
}

// ====== 会议生命周期（Task 5 实现） ======

// CreateRoom 创建会议房间
// Task 5：生成 room_code、bcrypt 密码、持久化 + 写 Redis room 快照
func (s *MeetingService) CreateRoom(ctx context.Context, hostID int64, title, password string, maxMembers int, settings map[string]interface{}) (*model.MeetingRoom, error) {
	return nil, ErrNotImplemented
}

// GetRoomByCode 通过会议号查询房间详情（含当前成员列表）
func (s *MeetingService) GetRoomByCode(ctx context.Context, code string) (*model.MeetingRoom, error) {
	return nil, ErrNotImplemented
}

// JoinRoom 用户加入会议（校验密码 / 容量 / 单点参会）
func (s *MeetingService) JoinRoom(ctx context.Context, userID int64, code, password string) (*model.MeetingParticipant, error) {
	return nil, ErrNotImplemented
}

// LeaveRoom 用户主动离会
// 若离开者是 host 且房间内还有其他成员，触发主持人转让（最早加入者接任）
// 若离开后房间空，设置 Redis room TTL=300s 等待重入复用
func (s *MeetingService) LeaveRoom(ctx context.Context, userID int64, code string) error {
	return ErrNotImplemented
}

// EndRoom 主持人主动结束会议（status=ended + 全员 left_at + 触发 mediasoup 清理）
func (s *MeetingService) EndRoom(ctx context.Context, userID int64, code string) error {
	return ErrNotImplemented
}

// TransferHost 主持人转让（仅当前 host 可调用）
func (s *MeetingService) TransferHost(ctx context.Context, operatorID int64, code string, newHostID int64) error {
	return ErrNotImplemented
}

// KickMember 主持人踢出成员
func (s *MeetingService) KickMember(ctx context.Context, operatorID int64, code string, targetUserID int64) error {
	return ErrNotImplemented
}

// ListMyMeetings 我主持的 / 我参与的会议列表（含分页）
func (s *MeetingService) ListMyMeetings(ctx context.Context, userID int64, role string, offset, limit int) ([]model.MeetingRoom, int64, error) {
	return nil, 0, ErrNotImplemented
}

// ====== 邀请链接（Task 5） ======

// CreateInviteToken 生成邀请链接 Token 并写 Redis（TTL 600s）
func (s *MeetingService) CreateInviteToken(ctx context.Context, inviterID int64, code string, inviteeID int64) (string, error) {
	return "", ErrNotImplemented
}

// RedeemInviteToken 点击邀请链接时兑换 Token（校验 + 删除）
func (s *MeetingService) RedeemInviteToken(ctx context.Context, userID int64, token string) (string, error) {
	return "", ErrNotImplemented
}

// InviteUsers 主持人邀请用户（批量，走 notify.Push 发送 meeting_invite 通知）
func (s *MeetingService) InviteUsers(ctx context.Context, inviterID int64, code string, inviteeIDs []int64, groupIDs []int64) error {
	return ErrNotImplemented
}

// ====== 会议内聊天（Task 6） ======

// SendChatMessage 写入会议聊天 + 向房间内所有成员广播 WS meeting.chat.message
func (s *MeetingService) SendChatMessage(ctx context.Context, userID int64, code, content string) (*model.MeetingChat, error) {
	return nil, ErrNotImplemented
}

// ListChatMessages 加载会议聊天历史（游标分页）
func (s *MeetingService) ListChatMessages(ctx context.Context, userID int64, code string, afterID int64, limit int) ([]model.MeetingChat, error) {
	return nil, ErrNotImplemented
}
