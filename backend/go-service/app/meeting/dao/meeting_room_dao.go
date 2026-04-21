// Package dao 提供 meeting 模块的数据库访问操作
package dao

import (
	"context"
	"errors"
	"time"

	"github.com/echochat/backend/app/constants"
	"github.com/echochat/backend/app/meeting/model"
	"github.com/echochat/backend/pkg/logs"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MeetingRoomDAO 会议房间数据访问对象
type MeetingRoomDAO struct {
	db *gorm.DB
}

// NewMeetingRoomDAO 创建 MeetingRoomDAO 实例
func NewMeetingRoomDAO(db *gorm.DB) *MeetingRoomDAO {
	return &MeetingRoomDAO{db: db}
}

// Create 新建一个会议房间
// 由调用方预先生成 RoomCode 并确认唯一，本方法不做冲突重试
func (d *MeetingRoomDAO) Create(ctx context.Context, room *model.MeetingRoom) error {
	funcName := "dao.meeting_room_dao.Create"
	logs.Info(ctx, funcName, "创建会议房间",
		zap.String("room_code", room.RoomCode), zap.Int64("host_id", room.HostID))

	err := d.db.WithContext(ctx).Create(room).Error
	if err != nil {
		logs.Error(ctx, funcName, "创建会议房间失败",
			zap.String("room_code", room.RoomCode), zap.Error(err))
	}
	return err
}

// GetByID 按主键查询
func (d *MeetingRoomDAO) GetByID(ctx context.Context, id int64) (*model.MeetingRoom, error) {
	var room model.MeetingRoom
	err := d.db.WithContext(ctx).First(&room, id).Error
	if err != nil {
		return nil, err
	}
	return &room, nil
}

// GetByCode 按会议号查询（入会流程的主要入口）
// 返回 gorm.ErrRecordNotFound 时上层应转换为业务错误 ErrMeetingNotFound
func (d *MeetingRoomDAO) GetByCode(ctx context.Context, code string) (*model.MeetingRoom, error) {
	funcName := "dao.meeting_room_dao.GetByCode"

	var room model.MeetingRoom
	err := d.db.WithContext(ctx).Where("room_code = ?", code).First(&room).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			logs.Error(ctx, funcName, "按会议号查询房间失败",
				zap.String("room_code", code), zap.Error(err))
		}
		return nil, err
	}
	return &room, nil
}

// ExistsCode 会议号是否已被占用（用于创建会议时的 code 冲突重试）
func (d *MeetingRoomDAO) ExistsCode(ctx context.Context, code string) (bool, error) {
	var count int64
	err := d.db.WithContext(ctx).
		Model(&model.MeetingRoom{}).
		Where("room_code = ?", code).
		Count(&count).Error
	return count > 0, err
}

// MarkStarted 记录会议实际开始时间（host 首次加入时调用）
// 仅当 status=pending 且 started_at IS NULL 时才写入，避免重复覆盖
func (d *MeetingRoomDAO) MarkStarted(ctx context.Context, id int64, startedAt time.Time) (int64, error) {
	funcName := "dao.meeting_room_dao.MarkStarted"
	res := d.db.WithContext(ctx).
		Model(&model.MeetingRoom{}).
		Where("id = ? AND status = ? AND started_at IS NULL", id, constants.MeetingStatusPending).
		Updates(map[string]interface{}{
			"status":     constants.MeetingStatusActive,
			"started_at": startedAt,
		})
	if res.Error != nil {
		logs.Error(ctx, funcName, "标记会议开始失败",
			zap.Int64("room_id", id), zap.Error(res.Error))
	}
	return res.RowsAffected, res.Error
}

// MarkEnded 记录会议结束（status=2 + ended_at + ended_reason）
// 乐观锁：只对 status != ended 的行生效，避免重复结束覆盖原始 reason
func (d *MeetingRoomDAO) MarkEnded(ctx context.Context, id int64, reason string, endedAt time.Time) (int64, error) {
	funcName := "dao.meeting_room_dao.MarkEnded"
	res := d.db.WithContext(ctx).
		Model(&model.MeetingRoom{}).
		Where("id = ? AND status != ?", id, constants.MeetingStatusEnded).
		Updates(map[string]interface{}{
			"status":        constants.MeetingStatusEnded,
			"ended_at":      endedAt,
			"ended_reason":  reason,
		})
	if res.Error != nil {
		logs.Error(ctx, funcName, "标记会议结束失败",
			zap.Int64("room_id", id), zap.String("reason", reason), zap.Error(res.Error))
	}
	return res.RowsAffected, res.Error
}

// UpdateHost 更新主持人（仅修改 meeting_rooms.host_id 字段，meeting_participants.role 由 ParticipantDAO.TransferHost 在同一事务中处理）
func (d *MeetingRoomDAO) UpdateHost(ctx context.Context, id, newHostID int64) error {
	return d.db.WithContext(ctx).
		Model(&model.MeetingRoom{}).
		Where("id = ?", id).
		Update("host_id", newHostID).Error
}

// UpdateSettings 更新房间级配置（settings 字段整体替换）
// 调用方需保证 settingsJSON 是合法 JSON 字符串
func (d *MeetingRoomDAO) UpdateSettings(ctx context.Context, id int64, settingsJSON string) error {
	return d.db.WithContext(ctx).
		Model(&model.MeetingRoom{}).
		Where("id = ?", id).
		Update("settings", settingsJSON).Error
}

// ListByHost 按主持人查询会议列表（支持 status 过滤 + 分页）
// status 传 -1 表示不限
// 返回按 created_at DESC 排序，走 idx_meeting_rooms_host_status 索引
func (d *MeetingRoomDAO) ListByHost(ctx context.Context, hostID int64, status int, offset, limit int) ([]model.MeetingRoom, int64, error) {
	funcName := "dao.meeting_room_dao.ListByHost"
	q := d.db.WithContext(ctx).
		Model(&model.MeetingRoom{}).
		Where("host_id = ?", hostID)
	if status >= 0 {
		q = q.Where("status = ?", status)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		logs.Error(ctx, funcName, "计数失败", zap.Int64("host_id", hostID), zap.Error(err))
		return nil, 0, err
	}

	var rooms []model.MeetingRoom
	err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&rooms).Error
	if err != nil {
		logs.Error(ctx, funcName, "查询失败", zap.Int64("host_id", hostID), zap.Error(err))
	}
	return rooms, total, err
}

// ListExpiredForCleanup 返回已结束且超过指定小时数的房间 ID 列表
// 供定时任务清理 meeting_chats 使用；room 本身保留归档不删
// 返回 limit 上限用于分批处理，避免一次捞太多
func (d *MeetingRoomDAO) ListExpiredForCleanup(ctx context.Context, hoursAgo int, limit int) ([]int64, error) {
	funcName := "dao.meeting_room_dao.ListExpiredForCleanup"
	cutoff := time.Now().Add(-time.Duration(hoursAgo) * time.Hour)

	var ids []int64
	err := d.db.WithContext(ctx).
		Model(&model.MeetingRoom{}).
		Where("status = ? AND ended_at IS NOT NULL AND ended_at < ?",
			constants.MeetingStatusEnded, cutoff).
		Order("ended_at ASC").
		Limit(limit).
		Pluck("id", &ids).Error
	if err != nil {
		logs.Error(ctx, funcName, "查询过期房间失败", zap.Error(err))
	}
	return ids, err
}
