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

// MeetingParticipantDAO 参与者数据访问对象
type MeetingParticipantDAO struct {
	db *gorm.DB
}

// NewMeetingParticipantDAO 创建实例
func NewMeetingParticipantDAO(db *gorm.DB) *MeetingParticipantDAO {
	return &MeetingParticipantDAO{db: db}
}

// JoinRoom 记录用户加入会议
// 语义：
//   - 若 (room_id, user_id) 不存在 → 创建新行，role 取参数值，joined_at = NOW()
//   - 若 (room_id, user_id) 已存在且 left_at != NULL（历史离会）→ 重置 left_at/left_reason/duration，
//     joined_at 刷新为当前时间（视为全新一次参会），role 保持原值（角色变更走 UpdateRole）
//   - 若 (room_id, user_id) 已存在且 left_at = NULL（仍在会议中）→ 返回 ErrAlreadyInMeeting
//
// 返回最新持久化的参与者记录
var (
	// ErrAlreadyInMeeting 用户已在会议中（未 left_at）
	ErrAlreadyInMeeting = errors.New("user already in meeting")
)

// JoinRoom 加入会议（幂等 + 重入复用记录）
func (d *MeetingParticipantDAO) JoinRoom(ctx context.Context, roomID, userID int64, role int) (*model.MeetingParticipant, error) {
	funcName := "dao.meeting_participant_dao.JoinRoom"
	logs.Info(ctx, funcName, "加入会议",
		zap.Int64("room_id", roomID), zap.Int64("user_id", userID), zap.Int("role", role))

	var result model.MeetingParticipant
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.MeetingParticipant
		findErr := tx.Where("room_id = ? AND user_id = ?", roomID, userID).First(&existing).Error

		if errors.Is(findErr, gorm.ErrRecordNotFound) {
			now := time.Now()
			result = model.MeetingParticipant{
				RoomID:   roomID,
				UserID:   userID,
				Role:     role,
				JoinedAt: now,
				Duration: 0,
			}
			return tx.Create(&result).Error
		}
		if findErr != nil {
			return findErr
		}

		if existing.LeftAt == nil {
			return ErrAlreadyInMeeting
		}

		now := time.Now()
		updates := map[string]interface{}{
			"joined_at":   now,
			"left_at":     nil,
			"left_reason": nil,
			"duration":    0,
		}
		if err := tx.Model(&model.MeetingParticipant{}).
			Where("id = ?", existing.ID).
			Updates(updates).Error; err != nil {
			return err
		}
		result = existing
		result.JoinedAt = now
		result.LeftAt = nil
		result.LeftReason = nil
		result.Duration = 0
		return nil
	})

	if err != nil && !errors.Is(err, ErrAlreadyInMeeting) {
		logs.Error(ctx, funcName, "加入会议失败",
			zap.Int64("room_id", roomID), zap.Int64("user_id", userID), zap.Error(err))
	}
	return &result, err
}

// LeaveRoom 记录用户离会
// 计算 duration = NOW() - joined_at（秒），写入 left_at / left_reason
// 已离会（left_at != NULL）的记录幂等忽略，返回受影响行数 0
func (d *MeetingParticipantDAO) LeaveRoom(ctx context.Context, roomID, userID int64, reason string) (int64, error) {
	funcName := "dao.meeting_participant_dao.LeaveRoom"
	now := time.Now()

	res := d.db.WithContext(ctx).
		Model(&model.MeetingParticipant{}).
		Where("room_id = ? AND user_id = ? AND left_at IS NULL", roomID, userID).
		Updates(map[string]interface{}{
			"left_at":     now,
			"left_reason": reason,
			// duration 使用 SQL 表达式保证数据库时间一致
			"duration": gorm.Expr("EXTRACT(EPOCH FROM (? - joined_at))::INT", now),
		})
	if res.Error != nil {
		logs.Error(ctx, funcName, "离开会议失败",
			zap.Int64("room_id", roomID), zap.Int64("user_id", userID), zap.Error(res.Error))
	}
	return res.RowsAffected, res.Error
}

// LeaveAllActive 批量将会议内所有仍活跃的参与者置为离会（用于会议结束时兜底）
// reason 建议传 host_end / empty_ttl 等房间级原因
func (d *MeetingParticipantDAO) LeaveAllActive(ctx context.Context, roomID int64, reason string) (int64, error) {
	funcName := "dao.meeting_participant_dao.LeaveAllActive"
	now := time.Now()

	res := d.db.WithContext(ctx).
		Model(&model.MeetingParticipant{}).
		Where("room_id = ? AND left_at IS NULL", roomID).
		Updates(map[string]interface{}{
			"left_at":     now,
			"left_reason": reason,
			"duration":    gorm.Expr("EXTRACT(EPOCH FROM (? - joined_at))::INT", now),
		})
	if res.Error != nil {
		logs.Error(ctx, funcName, "批量离会失败",
			zap.Int64("room_id", roomID), zap.Error(res.Error))
	}
	return res.RowsAffected, res.Error
}

// GetByRoomAndUser 查询单个参与者记录
func (d *MeetingParticipantDAO) GetByRoomAndUser(ctx context.Context, roomID, userID int64) (*model.MeetingParticipant, error) {
	var p model.MeetingParticipant
	err := d.db.WithContext(ctx).
		Where("room_id = ? AND user_id = ?", roomID, userID).
		First(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

// ListActiveByRoom 返回房间内所有仍未离会的参与者（left_at IS NULL）
// 按 joined_at ASC 排序，供"最早加入者接任主持人"使用
func (d *MeetingParticipantDAO) ListActiveByRoom(ctx context.Context, roomID int64) ([]model.MeetingParticipant, error) {
	funcName := "dao.meeting_participant_dao.ListActiveByRoom"
	var list []model.MeetingParticipant
	err := d.db.WithContext(ctx).
		Where("room_id = ? AND left_at IS NULL", roomID).
		Order("joined_at ASC").
		Find(&list).Error
	if err != nil {
		logs.Error(ctx, funcName, "查询在线参与者失败",
			zap.Int64("room_id", roomID), zap.Error(err))
	}
	return list, err
}

// CountActiveByRoom 房间内当前在线人数（不含已离会）
// 用于加入会议时的容量校验
func (d *MeetingParticipantDAO) CountActiveByRoom(ctx context.Context, roomID int64) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).
		Model(&model.MeetingParticipant{}).
		Where("room_id = ? AND left_at IS NULL", roomID).
		Count(&count).Error
	return count, err
}

// ListByRoom 房间内所有历史参与者（含已离会），按 joined_at ASC 排序
func (d *MeetingParticipantDAO) ListByRoom(ctx context.Context, roomID int64) ([]model.MeetingParticipant, error) {
	var list []model.MeetingParticipant
	err := d.db.WithContext(ctx).
		Where("room_id = ?", roomID).
		Order("joined_at ASC").
		Find(&list).Error
	return list, err
}

// FindActiveByUser 查询用户当前仍在进行中的会议（left_at IS NULL，且 room.status != ended）
// 用于"用户不能同时在多个会议中"的业务校验
// 返回 gorm.ErrRecordNotFound 表示当前空闲
func (d *MeetingParticipantDAO) FindActiveByUser(ctx context.Context, userID int64) (*model.MeetingParticipant, error) {
	var p model.MeetingParticipant
	err := d.db.WithContext(ctx).
		Joins("JOIN meeting_rooms ON meeting_rooms.id = meeting_participants.room_id").
		Where("meeting_participants.user_id = ? AND meeting_participants.left_at IS NULL AND meeting_rooms.status != ?",
			userID, constants.MeetingStatusEnded).
		First(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

// TransferHost 事务内完成主持人转让
// 1) 旧 host 的 role 置为 Participant（0）
// 2) 新 host 的 role 置为 Host（1）
// 调用方应在同一上游事务中同时调 MeetingRoomDAO.UpdateHost 修改 meeting_rooms.host_id
// 本方法内部已自带事务，上游无需再包裹
func (d *MeetingParticipantDAO) TransferHost(ctx context.Context, roomID, oldHostID, newHostID int64) error {
	funcName := "dao.meeting_participant_dao.TransferHost"
	logs.Info(ctx, funcName, "转让主持人",
		zap.Int64("room_id", roomID),
		zap.Int64("old_host_id", oldHostID),
		zap.Int64("new_host_id", newHostID))

	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.MeetingParticipant{}).
			Where("room_id = ? AND user_id = ? AND role = ?",
				roomID, oldHostID, constants.MeetingRoleHost).
			Update("role", constants.MeetingRoleParticipant).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.MeetingParticipant{}).
			Where("room_id = ? AND user_id = ? AND left_at IS NULL",
				roomID, newHostID).
			Update("role", constants.MeetingRoleHost).Error; err != nil {
			return err
		}
		return nil
	})
}

// UpdateRole 更新指定参与者角色（非转让场景，例如直接指派联合主持人）
func (d *MeetingParticipantDAO) UpdateRole(ctx context.Context, roomID, userID int64, role int) error {
	return d.db.WithContext(ctx).
		Model(&model.MeetingParticipant{}).
		Where("room_id = ? AND user_id = ?", roomID, userID).
		Update("role", role).Error
}

// ListByUser 用户历史会议列表（分页，按加入时间倒序）
func (d *MeetingParticipantDAO) ListByUser(ctx context.Context, userID int64, offset, limit int) ([]model.MeetingParticipant, int64, error) {
	q := d.db.WithContext(ctx).
		Model(&model.MeetingParticipant{}).
		Where("user_id = ?", userID)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []model.MeetingParticipant
	err := q.Order("joined_at DESC").Offset(offset).Limit(limit).Find(&list).Error
	return list, total, err
}
