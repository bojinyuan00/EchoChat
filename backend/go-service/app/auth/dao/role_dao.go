package dao

import (
	"context"
	"math"

	"github.com/echochat/backend/app/auth/model"
	"github.com/echochat/backend/pkg/logs"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RoleDAO 角色数据访问对象
type RoleDAO struct {
	db *gorm.DB
}

// NewRoleDAO 创建 RoleDAO 实例
func NewRoleDAO(db *gorm.DB) *RoleDAO {
	return &RoleDAO{db: db}
}

// FindByCode 按角色代码查询角色
// 利用 code 唯一索引，精确匹配
func (d *RoleDAO) FindByCode(ctx context.Context, code string) (*model.Role, error) {
	funcName := "dao.role_dao.FindByCode"
	logs.Debug(ctx, funcName, "按代码查询角色", zap.String("code", code))

	var role model.Role
	err := d.db.WithContext(ctx).Where("code = ?", code).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// AssignRole 为用户分配角色
// 插入 user_id + role_id 关联记录
func (d *RoleDAO) AssignRole(ctx context.Context, userID int64, roleID int) error {
	funcName := "dao.role_dao.AssignRole"
	logs.Info(ctx, funcName, "分配角色",
		zap.Int64("user_id", userID),
		zap.Int("role_id", roleID),
	)

	var err error
	defer func() {
		if err != nil {
			logs.Error(ctx, funcName, "分配角色失败", zap.Error(err))
		}
	}()

	userRole := model.UserRole{
		UserID: userID,
		RoleID: roleID,
	}
	err = d.db.WithContext(ctx).Create(&userRole).Error
	return err
}

// GetUserRoles 获取用户的所有角色列表
// 通过 JOIN auth_roles 表获取角色完整信息
func (d *RoleDAO) GetUserRoles(ctx context.Context, userID int64) ([]model.Role, error) {
	funcName := "dao.role_dao.GetUserRoles"
	logs.Debug(ctx, funcName, "获取用户角色", zap.Int64("user_id", userID))

	var roles []model.Role
	err := d.db.WithContext(ctx).
		Joins("JOIN auth_user_roles ON auth_user_roles.role_id = auth_roles.id").
		Where("auth_user_roles.user_id = ?", userID).
		Find(&roles).Error
	if err != nil {
		return nil, err
	}
	return roles, nil
}

// GetUserRoleCodes 获取用户的角色代码列表（便于权限判断）
func (d *RoleDAO) GetUserRoleCodes(ctx context.Context, userID int64) ([]string, error) {
	roles, err := d.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, err
	}

	codes := make([]string, 0, len(roles))
	for _, r := range roles {
		codes = append(codes, r.Code)
	}
	return codes, nil
}

// HasRole 检查用户是否拥有指定角色
func (d *RoleDAO) HasRole(ctx context.Context, userID int64, roleCode string) (bool, error) {
	funcName := "dao.role_dao.HasRole"
	logs.Debug(ctx, funcName, "检查用户角色",
		zap.Int64("user_id", userID),
		zap.String("role_code", roleCode),
	)

	var count int64
	err := d.db.WithContext(ctx).
		Model(&model.UserRole{}).
		Joins("JOIN auth_roles ON auth_roles.id = auth_user_roles.role_id").
		Where("auth_user_roles.user_id = ? AND auth_roles.code = ?", userID, roleCode).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// RemoveUserRoles 移除用户的所有角色（用于重新分配）
func (d *RoleDAO) RemoveUserRoles(ctx context.Context, userID int64) error {
	funcName := "dao.role_dao.RemoveUserRoles"
	logs.Info(ctx, funcName, "移除用户所有角色", zap.Int64("user_id", userID))

	return d.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&model.UserRole{}).Error
}

// GetAllRoles 获取所有角色列表（按 level 升序排列）
func (d *RoleDAO) GetAllRoles(ctx context.Context) ([]model.Role, error) {
	funcName := "dao.role_dao.GetAllRoles"
	logs.Debug(ctx, funcName, "获取所有角色列表")

	var roles []model.Role
	err := d.db.WithContext(ctx).Order("level ASC").Find(&roles).Error
	if err != nil {
		logs.Error(ctx, funcName, "获取所有角色失败", zap.Error(err))
		return nil, err
	}
	return roles, nil
}

// GetUserMaxLevel 获取用户的最高权限等级（最小 level 值）
// 如果用户无角色，返回 math.MaxInt32 表示无权限
func (d *RoleDAO) GetUserMaxLevel(ctx context.Context, userID int64) (int, error) {
	funcName := "dao.role_dao.GetUserMaxLevel"
	logs.Debug(ctx, funcName, "获取用户最高权限等级", zap.Int64("user_id", userID))

	var minLevel *int
	err := d.db.WithContext(ctx).
		Model(&model.UserRole{}).
		Joins("JOIN auth_roles ON auth_roles.id = auth_user_roles.role_id").
		Where("auth_user_roles.user_id = ?", userID).
		Select("MIN(auth_roles.level)").
		Scan(&minLevel).Error
	if err != nil {
		logs.Error(ctx, funcName, "获取用户权限等级失败", zap.Error(err))
		return 0, err
	}
	if minLevel == nil {
		return math.MaxInt32, nil
	}
	return *minLevel, nil
}

// SetUserRoles 事务内先清除再批量设置用户角色
func (d *RoleDAO) SetUserRoles(ctx context.Context, userID int64, roleIDs []int) error {
	funcName := "dao.role_dao.SetUserRoles"
	logs.Info(ctx, funcName, "设置用户角色",
		zap.Int64("user_id", userID),
		zap.Ints("role_ids", roleIDs),
	)

	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&model.UserRole{}).Error; err != nil {
			logs.Error(ctx, funcName, "清除用户角色失败", zap.Error(err))
			return err
		}

		if len(roleIDs) == 0 {
			return nil
		}

		userRoles := make([]model.UserRole, 0, len(roleIDs))
		for _, rid := range roleIDs {
			userRoles = append(userRoles, model.UserRole{
				UserID: userID,
				RoleID: rid,
			})
		}
		if err := tx.Create(&userRoles).Error; err != nil {
			logs.Error(ctx, funcName, "批量设置用户角色失败", zap.Error(err))
			return err
		}
		return nil
	})
}

// FindByCodeList 按角色代码列表批量查询角色
func (d *RoleDAO) FindByCodeList(ctx context.Context, codes []string) ([]model.Role, error) {
	funcName := "dao.role_dao.FindByCodeList"
	logs.Debug(ctx, funcName, "批量查询角色", zap.Strings("codes", codes))

	var roles []model.Role
	err := d.db.WithContext(ctx).Where("code IN ?", codes).Find(&roles).Error
	if err != nil {
		logs.Error(ctx, funcName, "批量查询角色失败", zap.Error(err))
		return nil, err
	}
	return roles, nil
}
