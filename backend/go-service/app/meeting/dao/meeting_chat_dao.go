package dao

import (
	"context"

	"github.com/echochat/backend/app/meeting/model"
	"github.com/echochat/backend/pkg/logs"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MeetingChatDAO 会议内文字聊天数据访问对象
type MeetingChatDAO struct {
	db *gorm.DB
}

// NewMeetingChatDAO 创建实例
func NewMeetingChatDAO(db *gorm.DB) *MeetingChatDAO {
	return &MeetingChatDAO{db: db}
}

// Create 写入一条会议聊天消息
// ID / CreatedAt 由数据库自增 + autoCreateTime 填充
func (d *MeetingChatDAO) Create(ctx context.Context, chat *model.MeetingChat) error {
	funcName := "dao.meeting_chat_dao.Create"
	err := d.db.WithContext(ctx).Create(chat).Error
	if err != nil {
		logs.Error(ctx, funcName, "写入会议聊天消息失败",
			zap.Int64("room_id", chat.RoomID),
			zap.Int64("user_id", chat.UserID),
			zap.Error(err))
	}
	return err
}

// ListByRoom 按房间正序返回聊天历史（供会议室打开时加载）
// afterID 为游标（id > afterID 的记录），0 表示从头开始
// limit 默认 50，上限 200
func (d *MeetingChatDAO) ListByRoom(ctx context.Context, roomID int64, afterID int64, limit int) ([]model.MeetingChat, error) {
	funcName := "dao.meeting_chat_dao.ListByRoom"
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	q := d.db.WithContext(ctx).Where("room_id = ?", roomID)
	if afterID > 0 {
		q = q.Where("id > ?", afterID)
	}

	var list []model.MeetingChat
	err := q.Order("id ASC").Limit(limit).Find(&list).Error
	if err != nil {
		logs.Error(ctx, funcName, "查询会议聊天失败",
			zap.Int64("room_id", roomID), zap.Error(err))
	}
	return list, err
}

// ListByRoomBefore 按房间反向游标分页查询聊天历史（id < beforeID），按 created_at DESC, id DESC 排序
// Task 16 P1-2：ListChatMessages 回归 DAO（原 service 层直接拼 SQL 破坏分层）
// beforeID 为 0 表示从最新的一条开始；limit <= 0 时返回空列表
// 由调用方传 limit+1 自行判断 has_more
func (d *MeetingChatDAO) ListByRoomBefore(ctx context.Context, roomID int64, beforeID int64, limit int) ([]model.MeetingChat, error) {
	if limit <= 0 {
		return []model.MeetingChat{}, nil
	}
	funcName := "dao.meeting_chat_dao.ListByRoomBefore"
	q := d.db.WithContext(ctx).
		Model(&model.MeetingChat{}).
		Where("room_id = ?", roomID)
	if beforeID > 0 {
		q = q.Where("id < ?", beforeID)
	}
	var list []model.MeetingChat
	if err := q.Order("created_at DESC, id DESC").Limit(limit).Find(&list).Error; err != nil {
		logs.Error(ctx, funcName, "反向游标查询会议聊天失败",
			zap.Int64("room_id", roomID), zap.Int64("before_id", beforeID), zap.Error(err))
		return nil, err
	}
	return list, nil
}

// DeleteByRoomIDs 按房间 ID 批量删除聊天消息（会议结束 24 小时后清理）
// 外层已有 meeting_rooms ON DELETE CASCADE，此方法用于主动定时清理（不删除 room 本身）
func (d *MeetingChatDAO) DeleteByRoomIDs(ctx context.Context, roomIDs []int64) (int64, error) {
	funcName := "dao.meeting_chat_dao.DeleteByRoomIDs"
	if len(roomIDs) == 0 {
		return 0, nil
	}
	res := d.db.WithContext(ctx).
		Where("room_id IN ?", roomIDs).
		Delete(&model.MeetingChat{})
	if res.Error != nil {
		logs.Error(ctx, funcName, "批量清理聊天消息失败",
			zap.Int("room_count", len(roomIDs)), zap.Error(res.Error))
	}
	return res.RowsAffected, res.Error
}

// CountByRoom 统计房间内聊天条数（供管理端观测）
func (d *MeetingChatDAO) CountByRoom(ctx context.Context, roomID int64) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).
		Model(&model.MeetingChat{}).
		Where("room_id = ?", roomID).
		Count(&count).Error
	return count, err
}
