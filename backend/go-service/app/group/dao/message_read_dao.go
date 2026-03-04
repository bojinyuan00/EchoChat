package dao

import (
	"context"

	"github.com/echochat/backend/app/group/model"
	"github.com/echochat/backend/pkg/logs"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// MessageReadDAO 消息已读记录数据访问对象
type MessageReadDAO struct {
	db *gorm.DB
}

// NewMessageReadDAO 创建 MessageReadDAO 实例
func NewMessageReadDAO(db *gorm.DB) *MessageReadDAO {
	return &MessageReadDAO{db: db}
}

// BatchCreate 批量创建已读记录（忽略重复冲突）
func (d *MessageReadDAO) BatchCreate(ctx context.Context, reads []model.MessageRead) error {
	funcName := "dao.message_read_dao.BatchCreate"
	logs.Info(ctx, funcName, "批量创建已读记录", zap.Int("count", len(reads)))

	if len(reads) == 0 {
		return nil
	}
	return d.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&reads).Error
}

// GetReadCount 获取消息的已读人数
func (d *MessageReadDAO) GetReadCount(ctx context.Context, messageID int64) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).
		Model(&model.MessageRead{}).
		Where("message_id = ?", messageID).
		Count(&count).Error
	return count, err
}

// GetReadCountBatch 批量获取多条消息的已读人数
func (d *MessageReadDAO) GetReadCountBatch(ctx context.Context, messageIDs []int64) (map[int64]int, error) {
	funcName := "dao.message_read_dao.GetReadCountBatch"
	logs.Debug(ctx, funcName, "批量获取已读计数", zap.Int("count", len(messageIDs)))

	type result struct {
		MessageID int64 `gorm:"column:message_id"`
		ReadCount int   `gorm:"column:read_count"`
	}

	var results []result
	err := d.db.WithContext(ctx).
		Model(&model.MessageRead{}).
		Select("message_id, COUNT(*) as read_count").
		Where("message_id IN ?", messageIDs).
		Group("message_id").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	m := make(map[int64]int, len(results))
	for _, r := range results {
		m[r.MessageID] = r.ReadCount
	}
	return m, nil
}

// GetReadUsers 获取消息的已读用户列表
func (d *MessageReadDAO) GetReadUsers(ctx context.Context, messageID int64) ([]model.MessageRead, error) {
	funcName := "dao.message_read_dao.GetReadUsers"
	logs.Debug(ctx, funcName, "获取已读用户列表", zap.Int64("message_id", messageID))

	var reads []model.MessageRead
	err := d.db.WithContext(ctx).
		Where("message_id = ?", messageID).
		Order("read_at ASC").
		Find(&reads).Error
	return reads, err
}

// BatchCreateReads 批量标记消息已读（满足 im/service.MessageReadRecorder 接口）
func (d *MessageReadDAO) BatchCreateReads(ctx context.Context, messageIDs []int64, userID int64) error {
	funcName := "dao.message_read_dao.BatchCreateReads"
	logs.Info(ctx, funcName, "批量标记已读",
		zap.Int64("user_id", userID), zap.Int("count", len(messageIDs)))

	if len(messageIDs) == 0 {
		return nil
	}

	reads := make([]model.MessageRead, 0, len(messageIDs))
	for _, msgID := range messageIDs {
		reads = append(reads, model.MessageRead{
			MessageID: msgID,
			UserID:    userID,
		})
	}
	return d.BatchCreate(ctx, reads)
}

// GetReadUserIDs 获取消息的已读用户 ID 列表（满足 im/service.MessageReadRecorder 接口）
func (d *MessageReadDAO) GetReadUserIDs(ctx context.Context, messageID int64) ([]int64, error) {
	funcName := "dao.message_read_dao.GetReadUserIDs"
	logs.Debug(ctx, funcName, "获取已读用户 ID 列表", zap.Int64("message_id", messageID))

	var ids []int64
	err := d.db.WithContext(ctx).
		Model(&model.MessageRead{}).
		Where("message_id = ?", messageID).
		Pluck("user_id", &ids).Error
	return ids, err
}

// HasRead 检查用户是否已读某消息
func (d *MessageReadDAO) HasRead(ctx context.Context, messageID, userID int64) (bool, error) {
	var count int64
	err := d.db.WithContext(ctx).
		Model(&model.MessageRead{}).
		Where("message_id = ? AND user_id = ?", messageID, userID).
		Count(&count).Error
	return count > 0, err
}
