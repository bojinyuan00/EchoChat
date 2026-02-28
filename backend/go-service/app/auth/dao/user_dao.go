// Package dao 提供 auth 模块的数据库访问操作
package dao

import (
	"context"

	"github.com/echochat/backend/app/auth/model"
	"github.com/echochat/backend/pkg/logs"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// UserDAO 用户数据访问对象
type UserDAO struct {
	db *gorm.DB
}

// NewUserDAO 创建 UserDAO 实例
func NewUserDAO(db *gorm.DB) *UserDAO {
	return &UserDAO{db: db}
}

// Create 创建用户
// 插入一条新用户记录，username 和 email 有唯一索引
func (d *UserDAO) Create(ctx context.Context, user *model.User) error {
	funcName := "dao.user_dao.Create"
	logs.Info(ctx, funcName, "创建用户",
		zap.String("username", user.Username),
		zap.String("email", logs.MaskEmail(user.Email)),
	)

	var err error
	defer func() {
		if err != nil {
			logs.Error(ctx, funcName, "创建用户失败", zap.Error(err))
		} else {
			logs.Info(ctx, funcName, "创建用户成功", zap.Int64("user_id", user.ID))
		}
	}()

	err = d.db.WithContext(ctx).Create(user).Error
	return err
}

// FindByEmail 按邮箱查询用户
// 精确匹配邮箱地址，利用 email 唯一索引
func (d *UserDAO) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	funcName := "dao.user_dao.FindByEmail"
	logs.Debug(ctx, funcName, "按邮箱查询用户", zap.String("email", logs.MaskEmail(email)))

	var user model.User
	err := d.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByUsername 按用户名查询用户
// 精确匹配用户名，利用 username 唯一索引
func (d *UserDAO) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	funcName := "dao.user_dao.FindByUsername"
	logs.Debug(ctx, funcName, "按用户名查询用户", zap.String("username", username))

	var user model.User
	err := d.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByID 按 ID 查询用户
func (d *UserDAO) FindByID(ctx context.Context, id int64) (*model.User, error) {
	funcName := "dao.user_dao.FindByID"
	logs.Debug(ctx, funcName, "按ID查询用户", zap.Int64("id", id))

	var user model.User
	err := d.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Update 更新用户信息
// 仅更新非零值字段
func (d *UserDAO) Update(ctx context.Context, user *model.User) error {
	funcName := "dao.user_dao.Update"
	logs.Info(ctx, funcName, "更新用户信息", zap.Int64("user_id", user.ID))

	var err error
	defer func() {
		if err != nil {
			logs.Error(ctx, funcName, "更新用户失败", zap.Int64("user_id", user.ID), zap.Error(err))
		}
	}()

	err = d.db.WithContext(ctx).Save(user).Error
	return err
}

// UpdateLastLogin 更新最后登录信息
func (d *UserDAO) UpdateLastLogin(ctx context.Context, userID int64, ip string) error {
	funcName := "dao.user_dao.UpdateLastLogin"
	logs.Debug(ctx, funcName, "更新登录信息", zap.Int64("user_id", userID))

	err := d.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"last_login_at": gorm.Expr("NOW()"),
			"last_login_ip": ip,
		}).Error
	return err
}

// FindByAccount 按用户名或邮箱查找用户（登录时使用）
func (d *UserDAO) FindByAccount(ctx context.Context, account string) (*model.User, error) {
	funcName := "dao.user_dao.FindByAccount"
	logs.Debug(ctx, funcName, "按账号查询用户", zap.String("account", account))

	var user model.User
	err := d.db.WithContext(ctx).
		Where("username = ? OR email = ?", account, account).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
