// Package dao 提供 admin 模块的数据库访问操作
package dao

import (
	"context"
	"fmt"

	"github.com/echochat/backend/app/auth/model"
	"github.com/echochat/backend/app/dto"
	"github.com/echochat/backend/pkg/logs"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// UserManageDAO 管理端用户数据访问对象
// 提供面向管理后台的用户查询操作（分页、搜索、筛选）
type UserManageDAO struct {
	db *gorm.DB
}

// NewUserManageDAO 创建 UserManageDAO 实例
func NewUserManageDAO(db *gorm.DB) *UserManageDAO {
	return &UserManageDAO{db: db}
}

// ListUsers 分页查询用户列表
// 支持按用户名/邮箱关键词搜索 + 按状态筛选
// SQL 执行两次查询：COUNT 总数 + LIMIT/OFFSET 分页数据
func (d *UserManageDAO) ListUsers(ctx context.Context, req *dto.UserListRequest) ([]model.User, int64, error) {
	funcName := "dao.user_manage_dao.ListUsers"
	logs.Info(ctx, funcName, "查询用户列表",
		zap.Int("page", req.Page),
		zap.Int("page_size", req.PageSize),
		zap.String("keyword", req.Keyword),
	)

	var users []model.User
	var total int64

	query := d.db.WithContext(ctx).Model(&model.User{})

	// 关键词搜索：模糊匹配用户名或邮箱
	if req.Keyword != "" {
		like := fmt.Sprintf("%%%s%%", req.Keyword)
		query = query.Where("username LIKE ? OR email LIKE ?", like, like)
	}

	// 状态筛选
	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
	}

	// 先查询总数
	if err := query.Count(&total).Error; err != nil {
		logs.Error(ctx, funcName, "查询用户总数失败", zap.Error(err))
		return nil, 0, err
	}

	// 分页查询数据，按创建时间倒序
	offset := (req.Page - 1) * req.PageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(req.PageSize).Find(&users).Error; err != nil {
		logs.Error(ctx, funcName, "查询用户列表失败", zap.Error(err))
		return nil, 0, err
	}

	logs.Info(ctx, funcName, "查询用户列表成功",
		zap.Int64("total", total),
		zap.Int("count", len(users)),
	)
	return users, total, nil
}

// CountUsers 统计用户总数（不含筛选条件）
func (d *UserManageDAO) CountUsers(ctx context.Context) (int64, error) {
	funcName := "dao.user_manage_dao.CountUsers"

	var count int64
	err := d.db.WithContext(ctx).Model(&model.User{}).Count(&count).Error
	if err != nil {
		logs.Error(ctx, funcName, "统计用户总数失败", zap.Error(err))
	}
	return count, err
}

// UpdateUserStatus 更新用户账号状态
// 仅更新 status 字段，不触发其他字段的更新
func (d *UserManageDAO) UpdateUserStatus(ctx context.Context, userID int64, status int) error {
	funcName := "dao.user_manage_dao.UpdateUserStatus"
	logs.Info(ctx, funcName, "更新用户状态",
		zap.Int64("user_id", userID),
		zap.Int("status", status),
	)

	err := d.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", userID).
		Update("status", status).Error

	if err != nil {
		logs.Error(ctx, funcName, "更新用户状态失败",
			zap.Int64("user_id", userID),
			zap.Error(err),
		)
	}
	return err
}
