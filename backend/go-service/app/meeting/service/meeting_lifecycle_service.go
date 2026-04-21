// Package service 提供 meeting 模块的业务逻辑
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/echochat/backend/app/constants"
	"github.com/echochat/backend/app/meeting/dao"
	"github.com/echochat/backend/config"
	"github.com/echochat/backend/pkg/logs"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Redis key 前缀（设计文档 §5.4 生命周期 keys；Task 8 落地）
const (
	redisKeyHostGracePrefix = "echo:meeting:host_grace:"
	redisKeyEmptyTTLPrefix  = "echo:meeting:empty_ttl:"
)

// hostGracePayload host 宽限期 Redis value 结构
type hostGracePayload struct {
	HostID     int64 `json:"host_id"`
	StartedAt  int64 `json:"started_at"`  // 秒级 unix
	GraceUntil int64 `json:"grace_until"` // 秒级 unix
}

// MeetingLifecycleService 会议生命周期状态机
// 负责设计 §6.5 的 5 类状态跃迁副作用（host 宽限期、自动转让、空房 TTL、TTL 过期销毁）
// 采用 time.AfterFunc 本地 timer + Redis key TTL 双保险：
//   - 本地 timer 负责低延迟触发；服务重启由 RescheduleFromRedis 按 Redis 剩余 PTTL 重新装载
//   - 后台定时任务 MeetingCleanupTask 每 N 秒扫描 Redis 兜底防御（timer 丢失 / 多实例部署重复触发场景）
//
// 并发保护：HandleHostGraceExpired / HandleEmptyRoomExpired 入口先 DEL Redis key
// 通过 DEL 返回值确认唯一处理权（0 = 已被其他 goroutine/节点清走，直接 return 避免重复操作）
type MeetingLifecycleService struct {
	roomDAO        *dao.MeetingRoomDAO
	participantDAO *dao.MeetingParticipantDAO
	db             dbExecutor
	redis          *redis.Client
	broadcaster    *MeetingBroadcaster
	media          MediaOrchestrator
	cfg            config.MeetingConfig

	graceTimers    sync.Map // roomCode -> *time.Timer（host 宽限期）
	emptyTTLTimers sync.Map // roomCode -> *time.Timer（空房 TTL）
}

// dbExecutor MeetingLifecycleService 对事务 DB 的最小依赖抽象
// 仅用于 HandleEmptyRoomExpired 的 MarkEnded 调用链路，避免强绑 *gorm.DB
type dbExecutor interface{}

// NewMeetingLifecycleService 构造生命周期服务
// media 由上游通过 wire 注入 HTTPMediaOrchestrator
func NewMeetingLifecycleService(
	roomDAO *dao.MeetingRoomDAO,
	participantDAO *dao.MeetingParticipantDAO,
	redis *redis.Client,
	broadcaster *MeetingBroadcaster,
	media MediaOrchestrator,
	cfg *config.Config,
) *MeetingLifecycleService {
	meetingCfg := cfg.Meeting
	// 零值兜底：防止 yaml 未配置时走 0 秒 TTL
	if meetingCfg.HostGraceSeconds <= 0 {
		meetingCfg.HostGraceSeconds = constants.MeetingHostGraceSeconds
	}
	if meetingCfg.EmptyRoomTTLSeconds <= 0 {
		meetingCfg.EmptyRoomTTLSeconds = constants.MeetingEmptyRoomTTLSeconds
	}
	if meetingCfg.CleanupIntervalSeconds <= 0 {
		meetingCfg.CleanupIntervalSeconds = 30
	}
	if meetingCfg.StaleRoomHours <= 0 {
		meetingCfg.StaleRoomHours = 4
	}
	return &MeetingLifecycleService{
		roomDAO:        roomDAO,
		participantDAO: participantDAO,
		redis:          redis,
		broadcaster:    broadcaster,
		media:          media,
		cfg:            meetingCfg,
	}
}

// Config 返回生效的生命周期配置（供定时任务读取扫描周期等）
func (s *MeetingLifecycleService) Config() config.MeetingConfig {
	return s.cfg
}

// HostGraceKey 返回 host 宽限期 Redis key
func HostGraceKey(roomCode string) string {
	return redisKeyHostGracePrefix + roomCode
}

// EmptyTTLKey 返回空房 TTL Redis key
func EmptyTTLKey(roomCode string) string {
	return redisKeyEmptyTTLPrefix + roomCode
}

// OnHostDisconnect host WS 断线钩子
// 写入 host_grace key（TTL HostGraceSeconds）并启动本地 AfterFunc
// 若已有同 roomCode 的 grace 记录（重复触发），幂等刷新 TTL + 替换 timer
func (s *MeetingLifecycleService) OnHostDisconnect(ctx context.Context, roomCode string, hostID int64) {
	funcName := "service.meeting_lifecycle_service.OnHostDisconnect"

	now := time.Now()
	payload := hostGracePayload{
		HostID:     hostID,
		StartedAt:  now.Unix(),
		GraceUntil: now.Add(time.Duration(s.cfg.HostGraceSeconds) * time.Second).Unix(),
	}
	raw, _ := json.Marshal(payload)

	graceDur := time.Duration(s.cfg.HostGraceSeconds) * time.Second
	// Redis TTL 比本地 timer 多留一段 buffer（max(cleanup*2, 30s)）
	// 保证本地 timer 先触发 → DEL 命中；即便本地 timer 丢失，cleanup 也能在 buffer 内兜底扫到
	redisTTL := graceDur + s.ttlBuffer()
	if err := s.redis.Set(ctx, HostGraceKey(roomCode), string(raw), redisTTL).Err(); err != nil {
		logs.Warn(ctx, funcName, "写入 host_grace key 失败",
			zap.String("room_code", roomCode), zap.Int64("host_id", hostID), zap.Error(err))
		return
	}

	s.scheduleHostGraceTimer(roomCode, graceDur)
	logs.Info(ctx, funcName, "host 进入宽限期",
		zap.String("room_code", roomCode),
		zap.Int64("host_id", hostID),
		zap.Int("grace_seconds", s.cfg.HostGraceSeconds))
}

// OnHostReconnect host 重连钩子
// 清除 host_grace key 并取消本地 timer；用于 host 掉线后 TTL 内重新出现在 WS 层
// 幂等：key 不存在也不报错
func (s *MeetingLifecycleService) OnHostReconnect(ctx context.Context, roomCode string, hostID int64) {
	funcName := "service.meeting_lifecycle_service.OnHostReconnect"

	deleted, err := s.redis.Del(ctx, HostGraceKey(roomCode)).Result()
	if err != nil {
		logs.Warn(ctx, funcName, "清除 host_grace key 失败",
			zap.String("room_code", roomCode), zap.Error(err))
	}
	s.cancelTimer(&s.graceTimers, roomCode)
	if deleted > 0 {
		logs.Info(ctx, funcName, "host 宽限期内重连，撤销转让计划",
			zap.String("room_code", roomCode), zap.Int64("host_id", hostID))
	}
}

// HandleHostGraceExpired host 宽限期过期处理
// 触发时机：本地 timer 到期 或 cleanup task 扫到 key TTL <=0
// 流程：
//  1. DEL host_grace key → 返回 0 表示已被其他路径处理，直接 return 防止重复转让
//  2. 加载 room + 活跃成员；若 room 已 Ended / host 已变更，视为幂等完成
//  3. 活跃成员存在：选最早加入者转 host（事务），广播 host.changed，auto_reason=host_grace_expired
//  4. 无活跃成员：走 OnAllMembersLeft 路径（置 empty_ttl）
func (s *MeetingLifecycleService) HandleHostGraceExpired(ctx context.Context, roomCode string) {
	funcName := "service.meeting_lifecycle_service.HandleHostGraceExpired"

	deleted, err := s.redis.Del(ctx, HostGraceKey(roomCode)).Result()
	if err != nil {
		logs.Warn(ctx, funcName, "DEL host_grace key 失败（跳过本次处理）",
			zap.String("room_code", roomCode), zap.Error(err))
		return
	}
	s.cancelTimer(&s.graceTimers, roomCode)
	if deleted == 0 {
		logs.Debug(ctx, funcName, "host_grace key 不存在（已被重连/其他节点清走）",
			zap.String("room_code", roomCode))
		return
	}

	room, err := s.roomDAO.GetByCode(ctx, roomCode)
	if err != nil || room == nil {
		logs.Warn(ctx, funcName, "加载会议失败或已不存在",
			zap.String("room_code", roomCode), zap.Error(err))
		return
	}
	if room.Status == constants.MeetingStatusEnded {
		logs.Info(ctx, funcName, "会议已结束，跳过 host 宽限期过期处理",
			zap.String("room_code", roomCode))
		return
	}

	actives, err := s.participantDAO.ListActiveByRoom(ctx, room.ID)
	if err != nil {
		logs.Warn(ctx, funcName, "拉取活跃成员失败",
			zap.String("room_code", roomCode), zap.Error(err))
		return
	}
	// 排除原 host（若 left_at IS NULL 但 WS 掉线未落库）
	candidates := make([]int64, 0, len(actives))
	for _, p := range actives {
		if p.UserID == room.HostID {
			continue
		}
		candidates = append(candidates, p.UserID)
	}

	if len(candidates) == 0 {
		logs.Info(ctx, funcName, "host 宽限期过期且无其他活跃成员，转空房 TTL",
			zap.String("room_code", roomCode), zap.Int64("old_host_id", room.HostID))
		// 清掉 host 自己的记录（若 WS 掉线未落库，由兜底 LeaveRoom 处理）
		_, _ = s.participantDAO.LeaveRoom(ctx, room.ID, room.HostID, constants.MeetingLeftReasonDisconnect)
		s.OnAllMembersLeft(ctx, roomCode)
		return
	}

	newHostID := candidates[0]
	if err := s.participantDAO.TransferHost(ctx, room.ID, room.HostID, newHostID); err != nil {
		logs.Error(ctx, funcName, "host 自动转让 TransferHost 失败",
			zap.String("room_code", roomCode), zap.Error(err))
		return
	}
	if err := s.roomDAO.UpdateHost(ctx, room.ID, newHostID); err != nil {
		logs.Warn(ctx, funcName, "UpdateHost 失败（已发生参与者 role 变更）",
			zap.String("room_code", roomCode), zap.Error(err))
	}
	// 老 host 彻底离会（宽限期内未重连视为网络断开）
	_, _ = s.participantDAO.LeaveRoom(ctx, room.ID, room.HostID, constants.MeetingLeftReasonDisconnect)

	s.broadcaster.BroadcastToMeeting(ctx, room.ID, constants.MeetingWSEventHostChanged, map[string]interface{}{
		"room_code":   roomCode,
		"old_host_id": room.HostID,
		"new_host_id": newHostID,
		"auto_reason": "host_grace_expired",
	})
	logs.Info(ctx, funcName, "host 宽限期过期自动转让完成",
		zap.String("room_code", roomCode),
		zap.Int64("old_host_id", room.HostID),
		zap.Int64("new_host_id", newHostID))
}

// OnAllMembersLeft 全员退出钩子
// 设置 empty_ttl key + 启动本地 AfterFunc；期间有人重新加入可由 CancelEmptyTTL 复活房间
func (s *MeetingLifecycleService) OnAllMembersLeft(ctx context.Context, roomCode string) {
	funcName := "service.meeting_lifecycle_service.OnAllMembersLeft"

	ttl := time.Duration(s.cfg.EmptyRoomTTLSeconds) * time.Second
	// Redis TTL 同样加 buffer 避免与本地 timer 同时到期导致 DEL 无法命中
	redisTTL := ttl + s.ttlBuffer()
	if err := s.redis.Set(ctx, EmptyTTLKey(roomCode), "1", redisTTL).Err(); err != nil {
		logs.Warn(ctx, funcName, "写入 empty_ttl key 失败",
			zap.String("room_code", roomCode), zap.Error(err))
		return
	}
	s.scheduleEmptyTTLTimer(roomCode, ttl)
	logs.Info(ctx, funcName, "空房 TTL 已启动",
		zap.String("room_code", roomCode), zap.Int("ttl_seconds", s.cfg.EmptyRoomTTLSeconds))
}

// CancelEmptyTTL 取消空房 TTL（有新成员加入时调用，房间复活）
// 幂等：key 不存在也不报错
func (s *MeetingLifecycleService) CancelEmptyTTL(ctx context.Context, roomCode string) {
	funcName := "service.meeting_lifecycle_service.CancelEmptyTTL"

	deleted, err := s.redis.Del(ctx, EmptyTTLKey(roomCode)).Result()
	if err != nil {
		logs.Warn(ctx, funcName, "清除 empty_ttl key 失败",
			zap.String("room_code", roomCode), zap.Error(err))
	}
	s.cancelTimer(&s.emptyTTLTimers, roomCode)
	if deleted > 0 {
		logs.Info(ctx, funcName, "空房 TTL 已取消（有人加入）",
			zap.String("room_code", roomCode))
	}
}

// HandleEmptyRoomExpired 空房 TTL 过期处理
// 流程：DEL empty_ttl key → 确认没被新成员清走 → MarkEnded(reason=empty_ttl) + CloseRouter 幂等
func (s *MeetingLifecycleService) HandleEmptyRoomExpired(ctx context.Context, roomCode string) {
	funcName := "service.meeting_lifecycle_service.HandleEmptyRoomExpired"

	deleted, err := s.redis.Del(ctx, EmptyTTLKey(roomCode)).Result()
	if err != nil {
		logs.Warn(ctx, funcName, "DEL empty_ttl key 失败",
			zap.String("room_code", roomCode), zap.Error(err))
		return
	}
	s.cancelTimer(&s.emptyTTLTimers, roomCode)
	if deleted == 0 {
		logs.Debug(ctx, funcName, "empty_ttl key 已消失（有人重入 或 已处理过）",
			zap.String("room_code", roomCode))
		return
	}

	room, err := s.roomDAO.GetByCode(ctx, roomCode)
	if err != nil || room == nil {
		logs.Warn(ctx, funcName, "加载会议失败或已不存在",
			zap.String("room_code", roomCode), zap.Error(err))
		return
	}
	if room.Status == constants.MeetingStatusEnded {
		return
	}

	// 二次校验：DEL 成功与 DB 校验之间可能有新成员加入（并发场景）
	activeCount, _ := s.participantDAO.CountActiveByRoom(ctx, room.ID)
	if activeCount > 0 {
		logs.Info(ctx, funcName, "检测到活跃成员，放弃销毁",
			zap.String("room_code", roomCode), zap.Int64("active_count", activeCount))
		return
	}

	if _, mErr := s.roomDAO.MarkEnded(ctx, room.ID, constants.MeetingEndedReasonEmptyTTL, time.Now()); mErr != nil {
		logs.Error(ctx, funcName, "MarkEnded 失败",
			zap.String("room_code", roomCode), zap.Error(mErr))
		return
	}
	if err := s.media.CloseRouter(ctx, roomCode); err != nil {
		logs.Warn(ctx, funcName, "CloseRouter 失败（不影响 DB 状态）",
			zap.String("room_code", roomCode), zap.Error(err))
	}

	s.broadcaster.BroadcastToMeeting(ctx, room.ID, constants.MeetingWSEventRoomEnded, map[string]interface{}{
		"room_code": roomCode,
		"reason":    constants.MeetingEndedReasonEmptyTTL,
	})
	logs.Info(ctx, funcName, "空房 TTL 过期，会议已销毁",
		zap.String("room_code", roomCode), zap.Int64("room_id", room.ID))
}

// RescheduleFromRedis 服务启动时扫描 Redis 已存在的 grace / empty_ttl key，按剩余 PTTL 补装本地 timer
// 服务重启后调用一次，避免重启期间 WS 断线事件的本地 timer 丢失导致宽限期/TTL 永远不触发
func (s *MeetingLifecycleService) RescheduleFromRedis(ctx context.Context) {
	funcName := "service.meeting_lifecycle_service.RescheduleFromRedis"

	s.rescheduleByPattern(ctx, funcName, redisKeyHostGracePrefix+"*", func(roomCode string, dur time.Duration) {
		s.scheduleHostGraceTimer(roomCode, dur)
	})
	s.rescheduleByPattern(ctx, funcName, redisKeyEmptyTTLPrefix+"*", func(roomCode string, dur time.Duration) {
		s.scheduleEmptyTTLTimer(roomCode, dur)
	})
}

// ScanExpired 扫描指定 key pattern，返回 TTL <=0 （或即将过期）对应的 roomCode 列表
// 供定时任务兜底调用：若本地 timer 已丢失，此时 Redis key 可能仍存在但 TTL 为负
// 本实现采用 SCAN + PTTL，单次扫描 limit 控制内存
func (s *MeetingLifecycleService) ScanExpired(ctx context.Context, keyPrefix string, limit int64) []string {
	funcName := "service.meeting_lifecycle_service.ScanExpired"

	expired := make([]string, 0, limit)
	var cursor uint64
	for {
		keys, next, err := s.redis.Scan(ctx, cursor, keyPrefix+"*", limit).Result()
		if err != nil {
			logs.Warn(ctx, funcName, "SCAN 失败", zap.String("pattern", keyPrefix), zap.Error(err))
			return expired
		}
		for _, k := range keys {
			pttl, err := s.redis.PTTL(ctx, k).Result()
			if err != nil {
				continue
			}
			// PTTL<=0（redis 过期延迟删除场景）；PTTL==-2 已不存在
			if pttl > 0 {
				continue
			}
			roomCode := keyToRoomCode(k, keyPrefix)
			if roomCode != "" {
				expired = append(expired, roomCode)
			}
		}
		if next == 0 {
			break
		}
		cursor = next
	}
	return expired
}

// HostGracePrefix / EmptyTTLPrefix 暴露 key 前缀供定时任务扫描使用
func (s *MeetingLifecycleService) HostGracePrefix() string { return redisKeyHostGracePrefix }
func (s *MeetingLifecycleService) EmptyTTLPrefix() string  { return redisKeyEmptyTTLPrefix }

// ========== 内部辅助 ==========

func (s *MeetingLifecycleService) scheduleHostGraceTimer(roomCode string, d time.Duration) {
	s.cancelTimer(&s.graceTimers, roomCode)
	timer := time.AfterFunc(d, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		s.HandleHostGraceExpired(ctx, roomCode)
	})
	s.graceTimers.Store(roomCode, timer)
}

func (s *MeetingLifecycleService) scheduleEmptyTTLTimer(roomCode string, d time.Duration) {
	s.cancelTimer(&s.emptyTTLTimers, roomCode)
	timer := time.AfterFunc(d, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		s.HandleEmptyRoomExpired(ctx, roomCode)
	})
	s.emptyTTLTimers.Store(roomCode, timer)
}

// ttlBuffer 返回 Redis key 相对本地 timer 的额外 buffer 时长
// 用于保证本地 timer 先触发（DEL 命中），避免 Redis 自动过期后 DEL 返回 0 被误判为重复触发
// 规则：max(CleanupIntervalSeconds * 2, 30s)，确保单机 cleanup 至少能扫到一轮兜底
func (s *MeetingLifecycleService) ttlBuffer() time.Duration {
	buf := time.Duration(s.cfg.CleanupIntervalSeconds*2) * time.Second
	if buf < 30*time.Second {
		buf = 30 * time.Second
	}
	return buf
}

// cancelTimer 取消本地 timer（幂等，不存在也安全）
func (s *MeetingLifecycleService) cancelTimer(store *sync.Map, roomCode string) {
	if raw, ok := store.LoadAndDelete(roomCode); ok {
		if t, ok := raw.(*time.Timer); ok {
			t.Stop()
		}
	}
}

// rescheduleByPattern 通用"按 pattern 扫描 Redis + 按 PTTL 重建本地 timer"逻辑
func (s *MeetingLifecycleService) rescheduleByPattern(
	ctx context.Context,
	funcName string,
	pattern string,
	schedule func(roomCode string, dur time.Duration),
) {
	var cursor uint64
	total := 0
	for {
		keys, next, err := s.redis.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			logs.Warn(ctx, funcName, "SCAN 失败", zap.String("pattern", pattern), zap.Error(err))
			return
		}
		for _, k := range keys {
			pttl, err := s.redis.PTTL(ctx, k).Result()
			if err != nil || pttl <= 0 {
				continue
			}
			prefix := pattern[:len(pattern)-1] // 去掉末尾 *
			roomCode := keyToRoomCode(k, prefix)
			if roomCode == "" {
				continue
			}
			schedule(roomCode, pttl)
			total++
		}
		if next == 0 {
			break
		}
		cursor = next
	}
	if total > 0 {
		logs.Info(ctx, funcName, "服务启动重建 timer 完成",
			zap.String("pattern", pattern), zap.Int("count", total))
	}
}

// keyToRoomCode 从完整 Redis key 中去掉前缀得到 roomCode
// 返回空字符串表示格式异常
func keyToRoomCode(fullKey, prefix string) string {
	if len(fullKey) <= len(prefix) {
		return ""
	}
	return fullKey[len(prefix):]
}

// formatErr 构造统一带上下文的错误（用于少数需要 return 的场景，避免裸字符串拼接）
// nolint:unused
func formatErr(funcName, msg string, err error) error {
	return fmt.Errorf("%s: %s: %w", funcName, msg, err)
}
