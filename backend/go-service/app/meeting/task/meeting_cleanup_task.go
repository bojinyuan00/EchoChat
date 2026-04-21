// Package task 提供 meeting 模块的后台定时任务
package task

import (
	"context"
	"time"

	"github.com/echochat/backend/app/constants"
	"github.com/echochat/backend/app/meeting/dao"
	"github.com/echochat/backend/app/meeting/service"
	"github.com/echochat/backend/pkg/logs"
	"go.uber.org/zap"
)

// MeetingCleanupTask 会议生命周期兜底清理任务（Phase 2e-2 Task 8）
// 职责（设计 §11.4 兜底策略 + Task 8 落地）：
//  1. 启动时一次性 RescheduleFromRedis，按 Redis 残留 grace/empty_ttl key 重建本地 timer
//  2. 每 cleanupInterval 秒扫描 host_grace / empty_ttl，命中"Redis 仍在但本地 timer 可能已丢失"的场景
//     兜底触发 HandleHostGraceExpired / HandleEmptyRoomExpired；真正过期判断由 Lifecycle 内部 DEL 互斥
//  3. 每 10 分钟扫一次 stale active rooms（活跃状态但超过 StaleRoomHours 无成员动作）→ 强制 MarkEnded(system_error)
//  4. 每 24 小时扫 Ended 超过 chat 保留时长的房间 → DeleteByRoomIDs 清理 meeting_chats
//
// 所有周期都由独立 ticker 驱动，相互不阻塞；退出走 stopCh 统一关闭
type MeetingCleanupTask struct {
	lifecycleSvc *service.MeetingLifecycleService
	roomDAO      *dao.MeetingRoomDAO
	chatDAO      *dao.MeetingChatDAO

	cleanupInterval time.Duration // 扫 grace/empty_ttl 的周期
	staleInterval   time.Duration // 扫 stale active 的周期
	chatInterval    time.Duration // 扫 chat 清理的周期
	staleHours      int           // 超过此小时的活跃房间视为 stale

	stopCh chan struct{}
}

// NewMeetingCleanupTask 构造定时任务
// 扫描参数从 lifecycleSvc.Config() 读取，避免两处配置分离
func NewMeetingCleanupTask(
	lifecycleSvc *service.MeetingLifecycleService,
	roomDAO *dao.MeetingRoomDAO,
	chatDAO *dao.MeetingChatDAO,
) *MeetingCleanupTask {
	cfg := lifecycleSvc.Config()
	interval := time.Duration(cfg.CleanupIntervalSeconds) * time.Second
	if interval <= 0 {
		interval = 30 * time.Second
	}
	staleHours := cfg.StaleRoomHours
	if staleHours <= 0 {
		staleHours = 4
	}
	return &MeetingCleanupTask{
		lifecycleSvc:    lifecycleSvc,
		roomDAO:         roomDAO,
		chatDAO:         chatDAO,
		cleanupInterval: interval,
		staleInterval:   10 * time.Minute,
		chatInterval:    24 * time.Hour,
		staleHours:      staleHours,
		stopCh:          make(chan struct{}),
	}
}

// Start 启动任务（非阻塞）
// 应在 main 进程启动后调用，进程退出时调用 Stop
func (t *MeetingCleanupTask) Start() {
	funcName := "task.meeting_cleanup_task.Start"
	logs.Info(nil, funcName, "启动会议生命周期兜底任务",
		zap.Duration("cleanup_interval", t.cleanupInterval),
		zap.Duration("stale_interval", t.staleInterval),
		zap.Duration("chat_interval", t.chatInterval),
		zap.Int("stale_hours", t.staleHours))

	go t.runRescheduleOnce()
	go t.runLoop("grace_empty_ttl", t.cleanupInterval, 5*time.Second, t.scanExpiredKeys)
	go t.runLoop("stale_active", t.staleInterval, 30*time.Second, t.scanStaleActive)
	go t.runLoop("chat_cleanup", t.chatInterval, 60*time.Second, t.cleanupExpiredChats)
}

// Stop 停止所有循环
func (t *MeetingCleanupTask) Stop() {
	close(t.stopCh)
}

// runRescheduleOnce 启动后一次性扫描 Redis 残留 key 并补装本地 timer
func (t *MeetingCleanupTask) runRescheduleOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	t.lifecycleSvc.RescheduleFromRedis(ctx)
}

// runLoop 通用"固定间隔 + 预热延迟 + stopCh 退出"循环
// warmup 用于错峰避免启动风暴，单次业务执行由 do 函数自行控制超时
func (t *MeetingCleanupTask) runLoop(name string, interval, warmup time.Duration, do func()) {
	funcName := "task.meeting_cleanup_task.runLoop." + name

	timer := time.NewTimer(warmup)
	select {
	case <-t.stopCh:
		timer.Stop()
		return
	case <-timer.C:
	}

	do()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-t.stopCh:
			logs.Info(nil, funcName, "任务循环退出")
			return
		case <-ticker.C:
			do()
		}
	}
}

// scanExpiredKeys 扫 host_grace 与 empty_ttl 两个 key 集合
// Redis 原生 TTL 过期时 Go 的 PTTL 会返回 -2（不存在），此时无需兜底
// 少数场景（AfterFunc 丢失 / 多实例部署）会出现 Redis 仍有 key 但无本地 timer
// 本扫描对 "TTL≤0 但仍存在" 的 key 主动调 HandleXxxExpired（内部 DEL 互斥）
func (t *MeetingCleanupTask) scanExpiredKeys() {
	funcName := "task.meeting_cleanup_task.scanExpiredKeys"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	hostExpired := t.lifecycleSvc.ScanExpired(ctx, t.lifecycleSvc.HostGracePrefix(), 100)
	for _, code := range hostExpired {
		t.lifecycleSvc.HandleHostGraceExpired(ctx, code)
	}
	emptyExpired := t.lifecycleSvc.ScanExpired(ctx, t.lifecycleSvc.EmptyTTLPrefix(), 100)
	for _, code := range emptyExpired {
		t.lifecycleSvc.HandleEmptyRoomExpired(ctx, code)
	}
	if len(hostExpired)+len(emptyExpired) > 0 {
		logs.Info(ctx, funcName, "兜底处理过期 key 完成",
			zap.Int("host_grace", len(hostExpired)),
			zap.Int("empty_ttl", len(emptyExpired)))
	}
}

// scanStaleActive 扫超 staleHours 无任何成员动作的活跃会议，强制结束
// reason=system_error：明确标记非正常终止，便于运维排查 Node worker 掉链等异常
func (t *MeetingCleanupTask) scanStaleActive() {
	funcName := "task.meeting_cleanup_task.scanStaleActive"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	ids, err := t.roomDAO.ListStaleActive(ctx, t.staleHours, 100)
	if err != nil {
		logs.Warn(ctx, funcName, "扫描 stale active 失败", zap.Error(err))
		return
	}
	if len(ids) == 0 {
		return
	}
	now := time.Now()
	for _, id := range ids {
		if _, err := t.roomDAO.MarkEnded(ctx, id, constants.MeetingEndedReasonSystemError, now); err != nil {
			logs.Warn(ctx, funcName, "兜底 MarkEnded 失败",
				zap.Int64("room_id", id), zap.Error(err))
		}
	}
	logs.Info(ctx, funcName, "stale 会议兜底结束", zap.Int("count", len(ids)))
}

// cleanupExpiredChats 扫超过 MeetingChatRetentionHours 的 Ended 会议，批量删除聊天记录
func (t *MeetingCleanupTask) cleanupExpiredChats() {
	funcName := "task.meeting_cleanup_task.cleanupExpiredChats"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	ids, err := t.roomDAO.ListExpiredForCleanup(ctx, constants.MeetingChatRetentionHours, 500)
	if err != nil {
		logs.Warn(ctx, funcName, "扫描过期会议失败", zap.Error(err))
		return
	}
	if len(ids) == 0 {
		return
	}
	deleted, err := t.chatDAO.DeleteByRoomIDs(ctx, ids)
	if err != nil {
		logs.Warn(ctx, funcName, "批量删除聊天记录失败", zap.Error(err))
		return
	}
	logs.Info(ctx, funcName, "会议聊天清理完成",
		zap.Int("room_count", len(ids)),
		zap.Int64("chat_deleted", deleted))
}
