// Package task 提供 notify 模块的后台定时任务
package task

import (
	"context"
	"time"

	"github.com/echochat/backend/app/notify/dao"
	"github.com/echochat/backend/pkg/logs"
	"go.uber.org/zap"
)

// 默认每日凌晨 3 点执行一次，通过固定周期 24h 近似实现
// 如需精确到凌晨可在未来引入 robfig/cron 库替换 Ticker
const defaultInterval = 24 * time.Hour

// CleanupTask 过期通知清理任务
// 周期扫描 notify_notifications，删除已读且超过 constants.NotifyRetentionDays 天的记录
// 未读通知无论多久都不清理
type CleanupTask struct {
	notifyDAO *dao.NotificationDAO
	interval  time.Duration
	stopCh    chan struct{}
}

// NewCleanupTask 创建 CleanupTask 实例
// 默认周期 24 小时
func NewCleanupTask(notifyDAO *dao.NotificationDAO) *CleanupTask {
	return &CleanupTask{
		notifyDAO: notifyDAO,
		interval:  defaultInterval,
		stopCh:    make(chan struct{}),
	}
}

// Start 启动定时任务（非阻塞）
// 应在 main 进程启动后调用，进程退出时调用 Stop 释放 goroutine
func (t *CleanupTask) Start() {
	funcName := "task.cleanup_task.Start"
	logs.Info(nil, funcName, "启动通知清理任务", zap.Duration("interval", t.interval))

	go func() {
		// 启动时延迟 30s 再执行，避免与启动迁移 / 初始化竞争资源
		warmup := time.NewTimer(30 * time.Second)
		select {
		case <-t.stopCh:
			warmup.Stop()
			return
		case <-warmup.C:
		}
		t.runOnce()

		ticker := time.NewTicker(t.interval)
		defer ticker.Stop()
		for {
			select {
			case <-t.stopCh:
				return
			case <-ticker.C:
				t.runOnce()
			}
		}
	}()
}

// Stop 停止定时任务
func (t *CleanupTask) Stop() {
	close(t.stopCh)
}

// runOnce 执行一次清理，带独立超时控制
func (t *CleanupTask) runOnce() {
	funcName := "task.cleanup_task.runOnce"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	affected, err := t.notifyDAO.DeleteExpired(ctx)
	if err != nil {
		logs.Warn(ctx, funcName, "通知清理失败", zap.Error(err))
		return
	}
	logs.Info(ctx, funcName, "通知清理完成", zap.Int64("deleted", affected))
}
