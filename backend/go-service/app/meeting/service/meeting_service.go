// Package service 提供 meeting 模块的业务逻辑
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/echochat/backend/app/constants"
	"github.com/echochat/backend/app/dto"
	"github.com/echochat/backend/app/meeting/dao"
	"github.com/echochat/backend/app/meeting/model"
	notifyService "github.com/echochat/backend/app/notify/service"
	"github.com/echochat/backend/pkg/logs"
	"github.com/echochat/backend/pkg/utils"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Meeting 模块领域错误
// 命名与 group/notify 模块的 Err* 风格一致，Controller 通过 errors.Is 识别后映射 HTTP 状态码
var (
	ErrMeetingNotFound       = errors.New("会议不存在")
	ErrMeetingEnded          = errors.New("会议已结束")
	ErrMeetingFull           = errors.New("会议已满员")
	ErrMeetingPasswordWrong  = errors.New("会议密码错误")
	ErrMeetingPasswordLocked = errors.New("密码连续错误次数过多，请稍后重试")
	ErrMeetingPasswordReq    = errors.New("此会议需要密码")
	ErrNotInMeeting          = errors.New("你不在此会议中")
	ErrAlreadyInMeeting      = errors.New("你已在此会议中")
	ErrAlreadyInOtherMeeting = errors.New("你当前已在其他会议中")
	ErrNotMeetingHost        = errors.New("仅主持人可执行此操作")
	ErrInviteTokenInvalid    = errors.New("邀请链接已失效")
	ErrRoomCodeConflict      = errors.New("会议号生成冲突，请稍后重试")
	ErrKickSelfForbidden     = errors.New("不能踢出自己")
	ErrTransferToSelf        = errors.New("不能将主持人转让给自己")
	ErrTransferTargetInvalid = errors.New("目标用户不在会议中")
	// ErrResourceNotOwned 媒体资源归属校验失败：用户试图操作不属于自己的 transport/producer/consumer
	// 发生场景：Web 端用户抓到他人 producerID 后调 meeting.producer.close、或跨用户挂 consumer 等横向越权尝试
	ErrResourceNotOwned = errors.New("媒体资源归属校验失败，禁止操作他人资源")
	// ErrMediaServiceUnavailable 媒体服务当前不可用（Router 创建失败 / Node 宕机等）
	// 用于 CreateRoom / JoinRoom 的补偿路径，将前台错误与"用户输入错误"区分开
	ErrMediaServiceUnavailable = errors.New("媒体服务暂时不可用，请稍后重试")
)

// Redis key 前缀（设计文档 §5.4 - Redis 数据结构）
const (
	redisKeyInvitePrefix       = "echo:meeting:invite:"       // 邀请 Token
	redisKeyPasswordLockPrefix = "echo:meeting:lock:"         // 密码错误锁（code:user_id）
	redisPasswordAttemptPrefix = "echo:meeting:pwd_attempt:"  // 密码错误计数
)

// MeetingService 会议业务服务
// Task 5 完成：会议生命周期、主持人管理、邀请、会议内聊天 12 个 REST API 全部落地
// Task 6 新增：广播能力抽离到 MeetingBroadcaster；后续 WS 信令事件由 MeetingSignalService 处理
// Task 7 会把 mediaOrchestrator 的 Noop 实现替换为真实 Node HTTP Client
type MeetingService struct {
	roomDAO        *dao.MeetingRoomDAO
	participantDAO *dao.MeetingParticipantDAO
	chatDAO        *dao.MeetingChatDAO

	db    *gorm.DB
	redis *redis.Client

	broadcaster       *MeetingBroadcaster
	notifyPusher      NotifyPusher
	userResolver      UserInfoResolver
	onlineChecker     OnlineChecker
	mediaOrchestrator MediaOrchestrator
	lifecycleSvc      *MeetingLifecycleService
}

// NewMeetingService 创建 MeetingService 实例
// 依赖通过构造函数注入；接口依赖由上游 Wire 绑定到具体实现
// Task 7 (2026-04-21) 起 mediaOrchestrator 默认绑定 HTTPMediaOrchestrator
// Task 8 (2026-04-21) 起新增 lifecycleSvc 依赖：JoinRoom/LeaveRoom 的空房/复活场景由 lifecycleSvc 托管
func NewMeetingService(
	roomDAO *dao.MeetingRoomDAO,
	participantDAO *dao.MeetingParticipantDAO,
	chatDAO *dao.MeetingChatDAO,
	db *gorm.DB,
	redis *redis.Client,
	broadcaster *MeetingBroadcaster,
	notifyPusher NotifyPusher,
	userResolver UserInfoResolver,
	onlineChecker OnlineChecker,
	mediaOrchestrator MediaOrchestrator,
	lifecycleSvc *MeetingLifecycleService,
) *MeetingService {
	return &MeetingService{
		roomDAO:           roomDAO,
		participantDAO:    participantDAO,
		chatDAO:           chatDAO,
		db:                db,
		redis:             redis,
		broadcaster:       broadcaster,
		notifyPusher:      notifyPusher,
		userResolver:      userResolver,
		onlineChecker:     onlineChecker,
		mediaOrchestrator: mediaOrchestrator,
		lifecycleSvc:      lifecycleSvc,
	}
}

// ====== 辅助：鉴权 / 会议号 / 广播 ======

// assertIsActiveParticipant 确认用户是会议的活跃参会者；返回其 participant 记录供调用方复用
func (s *MeetingService) assertIsActiveParticipant(ctx context.Context, roomID, userID int64) (*model.MeetingParticipant, error) {
	p, err := s.participantDAO.GetByRoomAndUser(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}
	if p == nil || !p.IsActive() {
		return nil, ErrNotInMeeting
	}
	return p, nil
}

// assertIsHost 确认用户是会议当前主持人
func (s *MeetingService) assertIsHost(ctx context.Context, room *model.MeetingRoom, userID int64) error {
	if room == nil {
		return ErrMeetingNotFound
	}
	if room.HostID != userID {
		return ErrNotMeetingHost
	}
	return nil
}

// generateUniqueRoomCode 生成唯一的 XXX-XXX-XXX 会议号，冲突最多重试 MeetingRoomCodeRetryMax 次
func (s *MeetingService) generateUniqueRoomCode(ctx context.Context) (string, error) {
	for i := 0; i < constants.MeetingRoomCodeRetryMax; i++ {
		code, err := utils.GenerateMeetingRoomCode()
		if err != nil {
			return "", err
		}
		exists, err := s.roomDAO.ExistsCode(ctx, code)
		if err != nil {
			return "", err
		}
		if !exists {
			return code, nil
		}
	}
	return "", ErrRoomCodeConflict
}

// broadcastToActiveParticipants 向房间内所有活跃参会者广播 WS 事件
// Task 6 起实现已迁移至 MeetingBroadcaster.BroadcastToMeeting，本方法作为兼容壳保留
// 以减少调用侧改动；未来可逐步替换为直接调用 s.broadcaster.BroadcastToMeeting
func (s *MeetingService) broadcastToActiveParticipants(ctx context.Context, roomID int64, event string, data interface{}, excludeUserIDs ...int64) {
	s.broadcaster.BroadcastToMeeting(ctx, roomID, event, data, excludeUserIDs...)
}

// UserDisplayInfo 用户展示信息（昵称优先，降级为用户名；头像可空）
type UserDisplayInfo struct {
	Name   string
	Avatar string
}

// resolveUserDisplay 查单个用户的展示信息；查询失败/未命中时返回空字符串
// 仅服务层内部使用，调用方失败时降级为 user_name="" 即可
func (s *MeetingService) resolveUserDisplay(ctx context.Context, userID int64) (name string, avatar string) {
	if s.userResolver == nil || userID <= 0 {
		return "", ""
	}
	users, err := s.userResolver.GetUsersByIDs(ctx, []int64{userID})
	if err != nil || len(users) == 0 {
		return "", ""
	}
	u := users[0]
	if u.Nickname != "" {
		return u.Nickname, u.Avatar
	}
	return u.Username, u.Avatar
}

// ResolveUsersDisplay 批量查询用户展示信息，controller 端用于填充 DTO
// 返回 map[userID]UserDisplayInfo，查询失败时返回空 map（调用方按降级处理）
func (s *MeetingService) ResolveUsersDisplay(ctx context.Context, userIDs []int64) map[int64]UserDisplayInfo {
	out := make(map[int64]UserDisplayInfo, len(userIDs))
	if s.userResolver == nil || len(userIDs) == 0 {
		return out
	}
	// 去重
	seen := make(map[int64]struct{}, len(userIDs))
	unique := make([]int64, 0, len(userIDs))
	for _, id := range userIDs {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	users, err := s.userResolver.GetUsersByIDs(ctx, unique)
	if err != nil {
		return out
	}
	for _, u := range users {
		name := u.Nickname
		if name == "" {
			name = u.Username
		}
		out[u.ID] = UserDisplayInfo{Name: name, Avatar: u.Avatar}
	}
	return out
}

// ====== 会议生命周期 ======

// CreateRoom 创建即时会议
// 流程：生成唯一会议号 → bcrypt 密码 → 写入 meeting_rooms（status=Active + started_at=now）→ 主持人落 participant 表 → 驱动 mediasoup Router 创建
func (s *MeetingService) CreateRoom(ctx context.Context, hostID int64, req *dto.CreateMeetingRoomRequest) (*model.MeetingRoom, *model.MeetingParticipant, string, error) {
	funcName := "service.meeting_service.CreateRoom"

	var err error
	defer func() {
		if err != nil {
			logs.Warn(ctx, funcName, "创建会议失败", zap.Int64("host_id", hostID), zap.Error(err))
		}
	}()

	active, err := s.participantDAO.FindActiveByUser(ctx, hostID)
	if err != nil {
		return nil, nil, "", err
	}
	if active != nil {
		err = ErrAlreadyInOtherMeeting
		return nil, nil, "", err
	}

	code, err := s.generateUniqueRoomCode(ctx)
	if err != nil {
		return nil, nil, "", err
	}

	var passwordHash *string
	if req.Password != "" {
		hash, hErr := utils.HashPassword(req.Password)
		if hErr != nil {
			err = fmt.Errorf("密码哈希失败: %w", hErr)
			return nil, nil, "", err
		}
		passwordHash = &hash
	}

	maxMembers := req.MaxMembers
	if maxMembers <= 0 || maxMembers > constants.MeetingMVPMaxMembers {
		maxMembers = constants.MeetingMVPMaxMembers
	}

	now := time.Now()
	room := &model.MeetingRoom{
		RoomCode:     code,
		Title:        req.Title,
		HostID:       hostID,
		Type:         constants.MeetingTypeInstant,
		PasswordHash: passwordHash,
		MaxMembers:   maxMembers,
		Status:       constants.MeetingStatusActive,
		StartedAt:    &now,
		Settings:     "{}",
	}
	if err = s.roomDAO.Create(ctx, room); err != nil {
		return nil, nil, "", err
	}

	participant, err := s.participantDAO.JoinRoom(ctx, room.ID, hostID, constants.MeetingRoleHost)
	if err != nil {
		return nil, nil, "", err
	}

	// Task 7 起 CreateRouter 对接真实 mediasoup；Task 8 起仅在会议首次创建时调一次（房间级资源）
	// P0-3 修复（审计报告）：
	// 旧版在 Router 创建失败时仅 `logs.Warn` 继续返回成功，导致 DB 存在 active 房间但 mediasoup 端无 Router，
	// 所有入会者后续 transport.create 必失败；同时占用"一人一会议"名额，用户无法新建。
	// 现改为 fail-closed：Router 失败 → 补偿 LeaveRoom + MarkEnded(system_error) → 返回 ErrMediaServiceUnavailable 让前端提示重试。
	routerID, mediaErr := s.mediaOrchestrator.CreateRouter(ctx, code)
	if mediaErr != nil {
		logs.Warn(ctx, funcName, "mediasoup Router 创建失败，执行补偿回滚",
			zap.String("room_code", code), zap.Int64("host_id", hostID), zap.Error(mediaErr))

		if _, leaveErr := s.participantDAO.LeaveRoom(ctx, room.ID, hostID, constants.MeetingLeftReasonSelf); leaveErr != nil {
			logs.Warn(ctx, funcName, "补偿阶段 LeaveRoom 失败（已记录，清理任务会兜底）",
				zap.Int64("room_id", room.ID), zap.Int64("host_id", hostID), zap.Error(leaveErr))
		}
		if _, markErr := s.roomDAO.MarkEnded(ctx, room.ID, constants.MeetingEndedReasonSystemError, time.Now()); markErr != nil {
			logs.Warn(ctx, funcName, "补偿阶段 MarkEnded 失败（已记录，清理任务会兜底）",
				zap.Int64("room_id", room.ID), zap.Error(markErr))
		}

		err = ErrMediaServiceUnavailable
		return nil, nil, "", err
	}

	logs.Info(ctx, funcName, "会议创建成功",
		zap.String("room_code", code), zap.Int64("host_id", hostID), zap.String("router_id", routerID))
	return room, participant, routerID, nil
}

// GetRoomByCode 获取会议详情（当前用户必须为活跃参会者）
func (s *MeetingService) GetRoomByCode(ctx context.Context, userID int64, code string) (*model.MeetingRoom, []model.MeetingParticipant, int64, error) {
	room, err := s.roomDAO.GetByCode(ctx, code)
	if err != nil {
		return nil, nil, 0, err
	}
	if room == nil {
		return nil, nil, 0, ErrMeetingNotFound
	}
	if _, err := s.assertIsActiveParticipant(ctx, room.ID, userID); err != nil {
		return nil, nil, 0, err
	}
	participants, err := s.participantDAO.ListByRoom(ctx, room.ID)
	if err != nil {
		return nil, nil, 0, err
	}
	activeCount, err := s.participantDAO.CountActiveByRoom(ctx, room.ID)
	if err != nil {
		return nil, nil, 0, err
	}
	return room, participants, activeCount, nil
}

// JoinRoom 加入会议
// 校验顺序：房间存在 → 未结束 → 单点参会 → 密码锁定 → 密码校验 → 容量 → 写 participant → 广播 meeting.member.joined
func (s *MeetingService) JoinRoom(ctx context.Context, userID int64, code, password string) (*model.MeetingRoom, *model.MeetingParticipant, string, error) {
	funcName := "service.meeting_service.JoinRoom"

	room, err := s.roomDAO.GetByCode(ctx, code)
	if err != nil {
		return nil, nil, "", err
	}
	if room == nil {
		return nil, nil, "", ErrMeetingNotFound
	}
	if room.Status == constants.MeetingStatusEnded {
		return nil, nil, "", ErrMeetingEnded
	}

	if existing, pErr := s.participantDAO.GetByRoomAndUser(ctx, room.ID, userID); pErr != nil {
		return nil, nil, "", pErr
	} else if existing != nil && existing.IsActive() {
		return nil, nil, "", ErrAlreadyInMeeting
	}

	if active, aErr := s.participantDAO.FindActiveByUser(ctx, userID); aErr != nil {
		return nil, nil, "", aErr
	} else if active != nil && active.RoomID != room.ID {
		return nil, nil, "", ErrAlreadyInOtherMeeting
	}

	if room.PasswordHash != nil && *room.PasswordHash != "" {
		lockKey := redisKeyPasswordLockPrefix + code + ":" + strconv.FormatInt(userID, 10)
		if locked, _ := s.redis.Exists(ctx, lockKey).Result(); locked > 0 {
			return nil, nil, "", ErrMeetingPasswordLocked
		}
		if password == "" {
			return nil, nil, "", ErrMeetingPasswordReq
		}
		if !utils.CheckPassword(password, *room.PasswordHash) {
			attemptKey := redisPasswordAttemptPrefix + code + ":" + strconv.FormatInt(userID, 10)
			attempts, _ := s.redis.Incr(ctx, attemptKey).Result()
			if attempts == 1 {
				s.redis.Expire(ctx, attemptKey, time.Duration(constants.MeetingPasswordLockSeconds)*time.Second)
			}
			if attempts >= int64(constants.MeetingPasswordMaxAttempts) {
				s.redis.Set(ctx, lockKey, 1, time.Duration(constants.MeetingPasswordLockSeconds)*time.Second)
				s.redis.Del(ctx, attemptKey)
			}
			return nil, nil, "", ErrMeetingPasswordWrong
		}
		s.redis.Del(ctx, redisPasswordAttemptPrefix+code+":"+strconv.FormatInt(userID, 10))
	}

	activeCount, err := s.participantDAO.CountActiveByRoom(ctx, room.ID)
	if err != nil {
		return nil, nil, "", err
	}
	if int(activeCount) >= room.MaxMembers {
		return nil, nil, "", ErrMeetingFull
	}

	participant, err := s.participantDAO.JoinRoom(ctx, room.ID, userID, constants.MeetingRoleParticipant)
	if err != nil {
		return nil, nil, "", err
	}

	// Task 8：空房 TTL 复活
	// 若该房间正处于 empty_ttl 阶段（全员离开后的 5 分钟窗口），新成员加入立即取消销毁
	if s.lifecycleSvc != nil {
		s.lifecycleSvc.CancelEmptyTTL(ctx, code)
	}

	// Task 8：JoinRoom 不再主动调 CreateRouter（Router 在 CreateRoom 时创建、由 HTTPMediaOrchestrator 本地缓存）
	// 从缓存读取 routerID；缺失时（极少见：服务重启后未重建缓存）保持为空，不阻塞加入流程
	routerID, _ := s.mediaOrchestrator.ResolveRouterID(code)
	if routerID == "" {
		logs.Debug(ctx, funcName, "RouterID 缓存缺失（非致命，可能服务重启）",
			zap.String("room_code", code))
	}

	// 广播 payload 附带 user_name / user_avatar，前端 _onMemberJoined 直接落库，无需二次拉取
	name, avatar := s.resolveUserDisplay(ctx, userID)
	go s.broadcastToActiveParticipants(logs.DetachContext(ctx), room.ID, constants.MeetingWSEventMemberJoined, map[string]interface{}{
		"room_code":   code,
		"user_id":     userID,
		"user_name":   name,
		"user_avatar": avatar,
		"joined_at":   participant.JoinedAt.Format("2006-01-02 15:04:05"),
	}, userID)

	logs.Info(ctx, funcName, "用户加入会议成功", zap.String("room_code", code), zap.Int64("user_id", userID))
	return room, participant, routerID, nil
}

// LeaveRoom 用户主动离会
// 若离开者为 host 且房间内还有其他活跃成员：事务内转让给最早加入者；若为空房：关闭房间（status=Ended, reason=empty_ttl）
func (s *MeetingService) LeaveRoom(ctx context.Context, userID int64, code string) (int, error) {
	funcName := "service.meeting_service.LeaveRoom"

	room, err := s.roomDAO.GetByCode(ctx, code)
	if err != nil {
		return 0, err
	}
	if room == nil {
		return 0, ErrMeetingNotFound
	}
	participant, err := s.assertIsActiveParticipant(ctx, room.ID, userID)
	if err != nil {
		return 0, err
	}

	affected, err := s.participantDAO.LeaveRoom(ctx, room.ID, userID, constants.MeetingLeftReasonSelf)
	if err != nil {
		return 0, err
	}
	if affected == 0 {
		return 0, ErrNotInMeeting
	}

	updated, err := s.participantDAO.GetByRoomAndUser(ctx, room.ID, userID)
	duration := 0
	if err == nil && updated != nil {
		duration = updated.Duration
	}

	actives, err := s.participantDAO.ListActiveByRoom(ctx, room.ID)
	if err != nil {
		return duration, err
	}

	if participant.Role == constants.MeetingRoleHost && len(actives) > 0 {
		newHost := actives[0]
		// P1-1：TransferHost 内部事务已同时更新 meeting_rooms.host_id，不再需要单独 UpdateHost
		if txErr := s.participantDAO.TransferHost(ctx, room.ID, userID, newHost.UserID); txErr != nil {
			logs.Warn(ctx, funcName, "主持人自动转让失败", zap.Error(txErr))
		} else {
			go s.broadcastToActiveParticipants(logs.DetachContext(ctx), room.ID, constants.MeetingWSEventHostChanged, map[string]interface{}{
				"room_code":    code,
				"old_host_id":  userID,
				"new_host_id":  newHost.UserID,
				"auto_reason":  "host_left",
			})
		}
	}

	// Task 8：全员离会改走生命周期状态机（空房 TTL），不再立即销毁房间
	// 在 TTL 窗口内如有新成员加入，房间会被 CancelEmptyTTL 复活；TTL 到期由 HandleEmptyRoomExpired 兜底销毁
	if len(actives) == 0 && s.lifecycleSvc != nil {
		s.lifecycleSvc.OnAllMembersLeft(ctx, code)
	}

	go s.broadcastToActiveParticipants(logs.DetachContext(ctx), room.ID, constants.MeetingWSEventMemberLeft, map[string]interface{}{
		"room_code": code,
		"user_id":   userID,
		"reason":    constants.MeetingLeftReasonSelf,
	}, userID)

	logs.Info(ctx, funcName, "用户离会成功",
		zap.String("room_code", code), zap.Int64("user_id", userID), zap.Int("duration", duration))
	return duration, nil
}

// EndRoom 主持人结束会议
// 将房间 status=Ended + reason=host_ended + 所有活跃成员 LeaveAllActive + 广播 meeting.room.ended + 关闭 mediasoup Router
// EndRoom 主持人主动结束会议
// Task 16 P1-4 强化为行锁 + 单事务：
//  1. 事务内 SELECT meeting_rooms FOR UPDATE 行锁，避免与并发 JoinRoom 的 status 竞争
//     （旧版 JoinRoom 在乐观读 status=active 之后才写 participant 行，可能出现
//     "JoinRoom 刚读 status=active → EndRoom 改 status=ended + LeaveAll → JoinRoom 写入活跃 participant"
//     的僵尸参与者现象，导致用户被卡在"我在会议中"的状态但房间已结束）
//  2. 事务内依序执行 状态校验 / 活跃快照 / MarkEnded / LeaveAllActive，全部成功才提交
//  3. 事务成功后再做 mediasoup CloseRouter 与 WS 广播（这些是外部 IO，不应拉长锁持有时间）
func (s *MeetingService) EndRoom(ctx context.Context, userID int64, code string) error {
	funcName := "service.meeting_service.EndRoom"

	room, err := s.roomDAO.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	if room == nil {
		return ErrMeetingNotFound
	}
	if room.Status == constants.MeetingStatusEnded {
		return ErrMeetingEnded
	}
	if err := s.assertIsHost(ctx, room, userID); err != nil {
		return err
	}

	var (
		activesBefore []model.MeetingParticipant
		endedAt       = time.Now()
	)

	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		roomDAO := s.roomDAO.WithTx(tx)
		participantDAO := s.participantDAO.WithTx(tx)

		// 行锁：事务内持有 meeting_rooms 行级 X 锁，直到事务结束才释放
		locked, err := roomDAO.GetByIDForUpdate(ctx, room.ID)
		if err != nil {
			return err
		}
		if locked == nil {
			return ErrMeetingNotFound
		}
		// 行锁之后复检 status，避免"同时多个 host 点结束"时第二请求重复执行
		if locked.Status == constants.MeetingStatusEnded {
			return ErrMeetingEnded
		}

		snapshot, err := participantDAO.ListActiveByRoom(ctx, room.ID)
		if err != nil {
			return err
		}
		activesBefore = snapshot

		if _, err := roomDAO.MarkEnded(ctx, room.ID, constants.MeetingEndedReasonHostEnded, endedAt); err != nil {
			return err
		}
		if _, err := participantDAO.LeaveAllActive(ctx, room.ID, constants.MeetingLeftReasonHostEnd); err != nil {
			return err
		}
		return nil
	})
	if txErr != nil {
		if errors.Is(txErr, ErrMeetingEnded) || errors.Is(txErr, ErrMeetingNotFound) {
			return txErr
		}
		logs.Error(ctx, funcName, "结束会议事务失败",
			zap.String("room_code", code), zap.Int64("room_id", room.ID), zap.Error(txErr))
		return txErr
	}

	payload := map[string]interface{}{
		"room_code":    code,
		"ended_reason": constants.MeetingEndedReasonHostEnded,
		"ended_at":     endedAt.Format("2006-01-02 15:04:05"),
	}
	// activesBefore 已经是结束前的活跃成员快照，此时 participantDAO.ListActiveByRoom 查会返回空集合
	// 因此使用 broadcaster.PublishToUser 逐人定向推送（而非 BroadcastToMeeting 基于当前状态查库）
	for _, p := range activesBefore {
		_ = s.broadcaster.PublishToUser(ctx, p.UserID, constants.MeetingWSEventRoomEnded, payload)
	}

	if err := s.mediaOrchestrator.CloseRouter(ctx, code); err != nil {
		logs.Warn(ctx, funcName, "关闭 mediasoup Router 失败", zap.Error(err))
	}

	logs.Info(ctx, funcName, "会议已结束", zap.String("room_code", code), zap.Int64("host_id", userID))
	return nil
}

// ====== 主持人管理 ======

// TransferHost 主持人主动转让
func (s *MeetingService) TransferHost(ctx context.Context, operatorID int64, code string, targetUserID int64) error {
	funcName := "service.meeting_service.TransferHost"

	if operatorID == targetUserID {
		return ErrTransferToSelf
	}

	room, err := s.roomDAO.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	if room == nil {
		return ErrMeetingNotFound
	}
	if room.Status == constants.MeetingStatusEnded {
		return ErrMeetingEnded
	}
	if err := s.assertIsHost(ctx, room, operatorID); err != nil {
		return err
	}
	target, err := s.assertIsActiveParticipant(ctx, room.ID, targetUserID)
	if err != nil {
		if errors.Is(err, ErrNotInMeeting) {
			return ErrTransferTargetInvalid
		}
		return err
	}

	// P1-1：TransferHost 内部事务同时更新 meeting_rooms.host_id，无需单独 UpdateHost
	if err := s.participantDAO.TransferHost(ctx, room.ID, operatorID, target.UserID); err != nil {
		return err
	}

	go s.broadcastToActiveParticipants(logs.DetachContext(ctx), room.ID, constants.MeetingWSEventHostChanged, map[string]interface{}{
		"room_code":   code,
		"old_host_id": operatorID,
		"new_host_id": target.UserID,
		"auto_reason": "manual",
	})

	logs.Info(ctx, funcName, "主持人转让成功",
		zap.String("room_code", code), zap.Int64("old_host", operatorID), zap.Int64("new_host", target.UserID))
	return nil
}

// KickMember 主持人踢出成员
func (s *MeetingService) KickMember(ctx context.Context, operatorID int64, code string, targetUserID int64) error {
	funcName := "service.meeting_service.KickMember"

	if operatorID == targetUserID {
		return ErrKickSelfForbidden
	}
	room, err := s.roomDAO.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	if room == nil {
		return ErrMeetingNotFound
	}
	if room.Status == constants.MeetingStatusEnded {
		return ErrMeetingEnded
	}
	if err := s.assertIsHost(ctx, room, operatorID); err != nil {
		return err
	}
	if _, err := s.assertIsActiveParticipant(ctx, room.ID, targetUserID); err != nil {
		return err
	}

	affected, err := s.participantDAO.LeaveRoom(ctx, room.ID, targetUserID, constants.MeetingLeftReasonKicked)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotInMeeting
	}

	_ = s.broadcaster.PublishToUser(ctx, targetUserID, constants.MeetingWSEventMemberKicked, map[string]interface{}{
		"room_code": code,
		"user_id":   targetUserID,
		"by":        operatorID,
	})
	go s.broadcastToActiveParticipants(logs.DetachContext(ctx), room.ID, constants.MeetingWSEventMemberLeft, map[string]interface{}{
		"room_code": code,
		"user_id":   targetUserID,
		"reason":    constants.MeetingLeftReasonKicked,
	}, targetUserID)

	logs.Info(ctx, funcName, "踢出成员成功",
		zap.String("room_code", code), zap.Int64("target_user_id", targetUserID))
	return nil
}

// ListMyMeetings 我参与过的会议列表（包含主持）
// 基于 MeetingParticipantDAO.ListByUser 获取参会记录后批量查询对应 Room
// MVP 场景下数据量小（单用户 30 天内会议通常 <50 条），内存合并与状态过滤可接受
// 后续 Phase 2f 观察量级后若有必要再落 DAO 层 JOIN 优化
func (s *MeetingService) ListMyMeetings(ctx context.Context, userID int64, statusFilter *int, beforeID int64, limit int) ([]model.MeetingRoom, bool, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	// Task 16 P1-3：改走 participantDAO.ListJoinedRoomsByUser 的 JOIN 查询
	// 旧版两轮 (ListByUser O(N) → ListByIDs O(M))、N 放大 6 倍 → 单次 JOIN DISTINCT O(limit+1)
	rooms, err := s.participantDAO.ListJoinedRoomsByUser(ctx, userID, statusFilter, beforeID, limit+1)
	if err != nil {
		return nil, false, err
	}

	hasMore := false
	if len(rooms) > limit {
		hasMore = true
		rooms = rooms[:limit]
	}
	return rooms, hasMore, nil
}

// ====== 邀请链接与邀请推送 ======

// invitePayload 存入 Redis 的邀请 Token 载荷
type invitePayload struct {
	RoomCode  string `json:"room_code"`
	InviterID int64  `json:"inviter_id"`
	InviteeID int64  `json:"invitee_id"` // 0 表示通用链接（当前 MVP 不用）
	CreatedAt int64  `json:"created_at"`
}

// InviteUsers 主持人或参会者邀请用户
// 对每个 invitee 生成独立 Token 写 Redis（TTL 600s），并通过 NotifyPusher 推送 meeting_invite 通知
// 离线用户走通知入库（NotifyService 内部负责 WS 推送或未读补偿）
func (s *MeetingService) InviteUsers(ctx context.Context, inviterID int64, code string, inviteeIDs []int64) (int, int, error) {
	funcName := "service.meeting_service.InviteUsers"

	room, err := s.roomDAO.GetByCode(ctx, code)
	if err != nil {
		return 0, 0, err
	}
	if room == nil {
		return 0, 0, ErrMeetingNotFound
	}
	if room.Status == constants.MeetingStatusEnded {
		return 0, 0, ErrMeetingEnded
	}
	if _, err := s.assertIsActiveParticipant(ctx, room.ID, inviterID); err != nil {
		return 0, 0, err
	}

	// 邀请人展示信息：extra 里一并带上，前端卡片不必再查 actor_* 字段就能渲染
	inviterName, inviterAvatar := s.resolveUserDisplay(ctx, inviterID)

	pushed := 0
	skipped := 0
	seen := make(map[int64]struct{}, len(inviteeIDs))
	payloads := make([]*notifyService.PushPayload, 0, len(inviteeIDs))

	for _, invitee := range inviteeIDs {
		if invitee <= 0 || invitee == inviterID {
			skipped++
			continue
		}
		if _, dup := seen[invitee]; dup {
			skipped++
			continue
		}
		seen[invitee] = struct{}{}

		if p, _ := s.participantDAO.GetByRoomAndUser(ctx, room.ID, invitee); p != nil && p.IsActive() {
			skipped++
			continue
		}

		token, err := utils.GenerateMeetingInviteToken()
		if err != nil {
			logs.Warn(ctx, funcName, "生成邀请 Token 失败", zap.Error(err))
			skipped++
			continue
		}
		now := time.Now()
		expiredAt := now.Add(time.Duration(constants.MeetingInviteTokenTTL) * time.Second).Unix()
		payload := invitePayload{
			RoomCode:  code,
			InviterID: inviterID,
			InviteeID: invitee,
			CreatedAt: now.Unix(),
		}
		buf, _ := json.Marshal(payload)
		if err := s.redis.Set(ctx, redisKeyInvitePrefix+token, string(buf),
			time.Duration(constants.MeetingInviteTokenTTL)*time.Second).Err(); err != nil {
			logs.Warn(ctx, funcName, "写入邀请 Token 到 Redis 失败", zap.Error(err))
			skipped++
			continue
		}

		roomID := room.ID
		actor := inviterID
		// extra 字段约定（Phase 2e-2 Task 13 / design §10.1）：
		//   room_code / invite_token / room_title / has_password：进会所需参数
		//   inviter_id / inviter_name / inviter_avatar：前端卡片渲染"XX 邀请你加入..."
		//   expired_at：Unix 秒，与 Redis TTL 同步，前端据此把按钮灰显
		extra := map[string]interface{}{
			"room_code":       code,
			"invite_token":    token,
			"room_title":      room.Title,
			"has_password":    room.PasswordHash != nil,
			"inviter_id":      inviterID,
			"inviter_name":    inviterName,
			"inviter_avatar":  inviterAvatar,
			"expired_at":      expiredAt,
		}
		payloads = append(payloads, &notifyService.PushPayload{
			UserID:     invitee,
			Type:       constants.NotifyTypeMeetingInvite,
			Title:      "会议邀请",
			Content:    fmt.Sprintf("邀请你加入会议：%s", room.Title),
			ActorID:    &actor,
			TargetType: "meeting",
			TargetID:   &roomID,
			Extra:      extra,
		})
		pushed++
	}

	if len(payloads) > 0 {
		s.notifyPusher.PushBatch(ctx, payloads)
	}

	logs.Info(ctx, funcName, "会议邀请推送完成",
		zap.String("room_code", code), zap.Int("pushed", pushed), zap.Int("skipped", skipped))
	return pushed, skipped, nil
}

// RedeemInviteToken 点击邀请链接时兑换 Token
// 成功：返回会议号 + 邀请人 ID + 是否有密码，前端据此决定弹出密码输入框并调 JoinRoom
// Token 兑换后不立即删除，保留 60 秒冗余（用户可能刷新页面）；过期走 Redis 原生 TTL
func (s *MeetingService) RedeemInviteToken(ctx context.Context, userID int64, token string) (*dto.RedeemInviteTokenResponse, error) {
	raw, err := s.redis.Get(ctx, redisKeyInvitePrefix+token).Result()
	if errors.Is(err, redis.Nil) {
		return nil, ErrInviteTokenInvalid
	}
	if err != nil {
		return nil, err
	}

	var payload invitePayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, ErrInviteTokenInvalid
	}

	// InviteeID > 0 表示定向邀请：仅允许该用户兑换（防止链接转发给非预期收件人）
	if payload.InviteeID > 0 && payload.InviteeID != userID {
		return nil, ErrInviteTokenInvalid
	}

	room, err := s.roomDAO.GetByCode(ctx, payload.RoomCode)
	if err != nil {
		return nil, err
	}
	if room == nil || room.Status == constants.MeetingStatusEnded {
		return nil, ErrInviteTokenInvalid
	}

	return &dto.RedeemInviteTokenResponse{
		RoomCode:    payload.RoomCode,
		InviterID:   payload.InviterID,
		HasPassword: room.PasswordHash != nil && *room.PasswordHash != "",
	}, nil
}

// ====== 会议内聊天 ======

// SendChatMessage 会议内发送文本消息
func (s *MeetingService) SendChatMessage(ctx context.Context, userID int64, code, content string) (*model.MeetingChat, error) {
	funcName := "service.meeting_service.SendChatMessage"

	room, err := s.roomDAO.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, ErrMeetingNotFound
	}
	if room.Status == constants.MeetingStatusEnded {
		return nil, ErrMeetingEnded
	}
	if _, err := s.assertIsActiveParticipant(ctx, room.ID, userID); err != nil {
		return nil, err
	}

	chat := &model.MeetingChat{
		RoomID:  room.ID,
		UserID:  userID,
		Content: content,
	}
	if err := s.chatDAO.Create(ctx, chat); err != nil {
		return nil, err
	}

	// 附带 user_name / user_avatar，前端聊天面板无需额外拉取
	userName, userAvatar := s.resolveUserDisplay(ctx, userID)

	go s.broadcastToActiveParticipants(logs.DetachContext(ctx), room.ID, constants.MeetingWSEventChatMessage, map[string]interface{}{
		"room_code":   code,
		"message_id":  chat.ID,
		"user_id":     userID,
		"user_name":   userName,
		"user_avatar": userAvatar,
		"content":     content,
		"created_at":  chat.CreatedAt.Format("2006-01-02 15:04:05"),
	}, userID)

	logs.Debug(ctx, funcName, "会议聊天已发送",
		zap.String("room_code", code), zap.Int64("user_id", userID), zap.Int64("message_id", chat.ID))
	return chat, nil
}

// ListChatMessages 加载会议聊天历史（游标分页，按 created_at ASC 升序返回）
func (s *MeetingService) ListChatMessages(ctx context.Context, userID int64, code string, beforeID int64, limit int) ([]model.MeetingChat, bool, error) {
	room, err := s.roomDAO.GetByCode(ctx, code)
	if err != nil {
		return nil, false, err
	}
	if room == nil {
		return nil, false, ErrMeetingNotFound
	}
	if _, err := s.assertIsActiveParticipant(ctx, room.ID, userID); err != nil {
		return nil, false, err
	}

	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}

	// Task 16 P1-2：改走 chatDAO.ListByRoomBefore（反向游标），避免 service 层直接拼 SQL 破坏分层
	chats, err := s.chatDAO.ListByRoomBefore(ctx, room.ID, beforeID, limit+1)
	if err != nil {
		return nil, false, err
	}

	hasMore := false
	if len(chats) > limit {
		hasMore = true
		chats = chats[:limit]
	}
	return chats, hasMore, nil
}

// ResolveRouterInfo 代理 mediaOrchestrator.ResolveRouterInfo
// Task 9 引入：JoinRoom 响应需把 rtpCapabilities 一并回给前端，避免前端再拉一次
// 返回 (routerID, rtpCapabilities, ok)；未命中缓存返回 ("", nil, false)
func (s *MeetingService) ResolveRouterInfo(roomCode string) (string, json.RawMessage, bool) {
	if s.mediaOrchestrator == nil {
		return "", nil, false
	}
	return s.mediaOrchestrator.ResolveRouterInfo(roomCode)
}
