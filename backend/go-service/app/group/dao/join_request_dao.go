package dao

import (
	"context"

	"github.com/echochat/backend/app/constants"
	"github.com/echochat/backend/app/group/model"
	"github.com/echochat/backend/pkg/logs"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// JoinRequestDAO 入群申请数据访问对象
type JoinRequestDAO struct {
	db *gorm.DB
}

// NewJoinRequestDAO 创建 JoinRequestDAO 实例
func NewJoinRequestDAO(db *gorm.DB) *JoinRequestDAO {
	return &JoinRequestDAO{db: db}
}

// Create 创建入群申请
func (d *JoinRequestDAO) Create(ctx context.Context, groupID, userID int64, message string) (*model.GroupJoinRequest, error) {
	funcName := "dao.join_request_dao.Create"
	logs.Info(ctx, funcName, "创建入群申请",
		zap.Int64("group_id", groupID), zap.Int64("user_id", userID))

	req := &model.GroupJoinRequest{
		GroupID: groupID,
		UserID:  userID,
		Message: message,
		Status:  constants.JoinRequestStatusPending,
	}
	err := d.db.WithContext(ctx).Create(req).Error
	if err != nil {
		logs.Error(ctx, funcName, "创建入群申请失败", zap.Error(err))
	}
	return req, err
}

// GetByID 根据 ID 获取入群申请
func (d *JoinRequestDAO) GetByID(ctx context.Context, id int64) (*model.GroupJoinRequest, error) {
	var req model.GroupJoinRequest
	err := d.db.WithContext(ctx).First(&req, id).Error
	if err != nil {
		return nil, err
	}
	return &req, nil
}

// GetPendingByGroupAndUser 查找用户对某群的待处理申请
func (d *JoinRequestDAO) GetPendingByGroupAndUser(ctx context.Context, groupID, userID int64) (*model.GroupJoinRequest, error) {
	funcName := "dao.join_request_dao.GetPendingByGroupAndUser"
	logs.Debug(ctx, funcName, "查找待处理申请",
		zap.Int64("group_id", groupID), zap.Int64("user_id", userID))

	var req model.GroupJoinRequest
	err := d.db.WithContext(ctx).
		Where("group_id = ? AND user_id = ? AND status = ?", groupID, userID, constants.JoinRequestStatusPending).
		First(&req).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &req, nil
}

// GetListByGroup 获取群的入群申请列表
func (d *JoinRequestDAO) GetListByGroup(ctx context.Context, groupID int64) ([]model.GroupJoinRequest, error) {
	funcName := "dao.join_request_dao.GetListByGroup"
	logs.Debug(ctx, funcName, "获取入群申请列表", zap.Int64("group_id", groupID))

	var requests []model.GroupJoinRequest
	err := d.db.WithContext(ctx).
		Where("group_id = ?", groupID).
		Order("created_at DESC").
		Find(&requests).Error
	return requests, err
}

// Approve 通过入群申请
func (d *JoinRequestDAO) Approve(ctx context.Context, id, reviewerID int64) error {
	funcName := "dao.join_request_dao.Approve"
	logs.Info(ctx, funcName, "通过入群申请",
		zap.Int64("request_id", id), zap.Int64("reviewer_id", reviewerID))

	return d.db.WithContext(ctx).
		Model(&model.GroupJoinRequest{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":      constants.JoinRequestStatusApproved,
			"reviewer_id": reviewerID,
		}).Error
}

// Reject 拒绝入群申请
func (d *JoinRequestDAO) Reject(ctx context.Context, id, reviewerID int64) error {
	funcName := "dao.join_request_dao.Reject"
	logs.Info(ctx, funcName, "拒绝入群申请",
		zap.Int64("request_id", id), zap.Int64("reviewer_id", reviewerID))

	return d.db.WithContext(ctx).
		Model(&model.GroupJoinRequest{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":      constants.JoinRequestStatusRejected,
			"reviewer_id": reviewerID,
		}).Error
}
