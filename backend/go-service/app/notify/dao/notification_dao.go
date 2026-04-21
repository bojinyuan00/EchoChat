// Package dao 提供 notify 模块的数据库访问操作
package dao

import (
	"context"
	"time"

	"github.com/echochat/backend/app/constants"
	"github.com/echochat/backend/app/notify/model"
	"github.com/echochat/backend/pkg/logs"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// NotificationDAO 通知记录数据访问对象
type NotificationDAO struct {
	db *gorm.DB
}

// NewNotificationDAO 创建 NotificationDAO 实例
func NewNotificationDAO(db *gorm.DB) *NotificationDAO {
	return &NotificationDAO{db: db}
}

// Create 新增一条通知（单用户）
func (d *NotificationDAO) Create(ctx context.Context, n *model.Notification) error {
	funcName := "dao.notification_dao.Create"
	err := d.db.WithContext(ctx).Create(n).Error
	if err != nil {
		logs.Error(ctx, funcName, "写入通知失败",
			zap.Int64("user_id", n.UserID), zap.String("type", n.Type), zap.Error(err))
	}
	return err
}

// BatchCreate 批量新增通知（管理员广播场景）
// 使用 GORM 的 CreateInBatches 拆分为 500/批次，避免单次参数过多
func (d *NotificationDAO) BatchCreate(ctx context.Context, list []*model.Notification) error {
	funcName := "dao.notification_dao.BatchCreate"
	if len(list) == 0 {
		return nil
	}
	err := d.db.WithContext(ctx).CreateInBatches(list, 500).Error
	if err != nil {
		logs.Error(ctx, funcName, "批量写入通知失败",
			zap.Int("count", len(list)), zap.Error(err))
	}
	return err
}

// GetByID 根据 ID 获取单条通知
func (d *NotificationDAO) GetByID(ctx context.Context, id int64) (*model.Notification, error) {
	var n model.Notification
	err := d.db.WithContext(ctx).First(&n, id).Error
	if err != nil {
		return nil, err
	}
	return &n, nil
}

// ListRequest 列表查询条件
type ListRequest struct {
	UserID   int64    // 接收者用户 ID（必填）
	Types    []string // type 过滤（空则不限）
	IsRead   *bool    // 已读状态过滤（nil 则不限）
	BeforeID int64    // 游标（小于此 ID 的记录），0 表示不限
	Limit    int      // 单页条数
}

// List 按条件分页查询通知（按 ID DESC 返回）
// 使用游标分页（id < beforeID）避免 OFFSET 随数据增长变慢
func (d *NotificationDAO) List(ctx context.Context, req *ListRequest) ([]model.Notification, error) {
	funcName := "dao.notification_dao.List"
	q := d.db.WithContext(ctx).Model(&model.Notification{}).
		Where("user_id = ?", req.UserID)
	if len(req.Types) > 0 {
		q = q.Where("type IN ?", req.Types)
	}
	if req.IsRead != nil {
		q = q.Where("is_read = ?", *req.IsRead)
	}
	if req.BeforeID > 0 {
		q = q.Where("id < ?", req.BeforeID)
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	var list []model.Notification
	err := q.Order("id DESC").Limit(limit).Find(&list).Error
	if err != nil {
		logs.Error(ctx, funcName, "查询通知列表失败",
			zap.Int64("user_id", req.UserID), zap.Error(err))
	}
	return list, err
}

// CountUnread 统计未读总数
func (d *NotificationDAO) CountUnread(ctx context.Context, userID int64) (int64, error) {
	var total int64
	err := d.db.WithContext(ctx).Model(&model.Notification{}).
		Where("user_id = ? AND is_read = false", userID).
		Count(&total).Error
	return total, err
}

// CountUnreadByType 按 type 统计未读数（用于按分类汇总）
// 返回 type -> count
func (d *NotificationDAO) CountUnreadByType(ctx context.Context, userID int64) (map[string]int64, error) {
	type row struct {
		Type string
		Cnt  int64
	}
	var rows []row
	err := d.db.WithContext(ctx).Model(&model.Notification{}).
		Select("type, COUNT(*) as cnt").
		Where("user_id = ? AND is_read = false", userID).
		Group("type").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[string]int64, len(rows))
	for _, r := range rows {
		result[r.Type] = r.Cnt
	}
	return result, nil
}

// MarkRead 将指定通知标记为已读（仅限本人）
// 返回受影响行数，0 表示无效 ID 或非本人
func (d *NotificationDAO) MarkRead(ctx context.Context, id, userID int64) (int64, error) {
	funcName := "dao.notification_dao.MarkRead"
	now := time.Now()
	res := d.db.WithContext(ctx).Model(&model.Notification{}).
		Where("id = ? AND user_id = ? AND is_read = false", id, userID).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": now,
		})
	if res.Error != nil {
		logs.Error(ctx, funcName, "标记已读失败",
			zap.Int64("id", id), zap.Int64("user_id", userID), zap.Error(res.Error))
	}
	return res.RowsAffected, res.Error
}

// MarkAllRead 将用户所有未读通知标记为已读
// 可选按 types 过滤（为空则全部类型）
func (d *NotificationDAO) MarkAllRead(ctx context.Context, userID int64, types []string) (int64, error) {
	funcName := "dao.notification_dao.MarkAllRead"
	now := time.Now()
	q := d.db.WithContext(ctx).Model(&model.Notification{}).
		Where("user_id = ? AND is_read = false", userID)
	if len(types) > 0 {
		q = q.Where("type IN ?", types)
	}
	res := q.Updates(map[string]interface{}{
		"is_read": true,
		"read_at": now,
	})
	if res.Error != nil {
		logs.Error(ctx, funcName, "批量标记已读失败",
			zap.Int64("user_id", userID), zap.Error(res.Error))
	}
	return res.RowsAffected, res.Error
}

// DeleteExpired 清理已读过期通知（保留时长由 constants.NotifyRetentionDays 决定）
// 未读通知无论多久都不清理，避免用户漏看
// 返回清理的行数
func (d *NotificationDAO) DeleteExpired(ctx context.Context) (int64, error) {
	funcName := "dao.notification_dao.DeleteExpired"
	cutoff := time.Now().AddDate(0, 0, -constants.NotifyRetentionDays)
	res := d.db.WithContext(ctx).
		Where("is_read = true AND created_at < ?", cutoff).
		Delete(&model.Notification{})
	if res.Error != nil {
		logs.Error(ctx, funcName, "清理过期通知失败", zap.Error(res.Error))
	}
	return res.RowsAffected, res.Error
}

// ListAllActiveUserIDs 查询系统中所有正常状态的用户 ID（用于管理员广播）
// 此处直接查 auth_users 表，避免广播时循环依赖 user_dao
func (d *NotificationDAO) ListAllActiveUserIDs(ctx context.Context) ([]int64, error) {
	funcName := "dao.notification_dao.ListAllActiveUserIDs"
	var ids []int64
	err := d.db.WithContext(ctx).
		Table("auth_users").
		Where("status = ?", constants.UserStatusActive).
		Pluck("id", &ids).Error
	if err != nil {
		logs.Error(ctx, funcName, "查询全量用户 ID 失败", zap.Error(err))
	}
	return ids, err
}
