// Package service 提供 admin 模块的核心业务逻辑
package service

import (
	"context"
	"errors"
	"math"

	authDAO "github.com/echochat/backend/app/auth/dao"
	"github.com/echochat/backend/app/auth/model"
	adminDAO "github.com/echochat/backend/app/admin/dao"
	"github.com/echochat/backend/app/constants"
	"github.com/echochat/backend/app/dto"
	"github.com/echochat/backend/pkg/logs"
	"github.com/echochat/backend/pkg/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound           = errors.New("用户不存在")
	ErrUserExists             = errors.New("用户名或邮箱已被注册")
	ErrInvalidStatus          = errors.New("无效的用户状态")
	ErrInvalidRole            = errors.New("无效的角色代码")
	ErrCannotDisableSelf      = errors.New("不能禁用自己的账号")
	ErrInsufficientPermission = errors.New("权限不足，无法操作更高等级的用户")
	ErrCannotAssignHigherRole = errors.New("不能分配高于自身等级的角色")
)

// UserManageService 管理端用户管理服务
type UserManageService struct {
	userManageDAO *adminDAO.UserManageDAO
	userDAO       *authDAO.UserDAO
	roleDAO       *authDAO.RoleDAO
}

// NewUserManageService 创建用户管理服务实例
func NewUserManageService(
	userManageDAO *adminDAO.UserManageDAO,
	userDAO *authDAO.UserDAO,
	roleDAO *authDAO.RoleDAO,
) *UserManageService {
	return &UserManageService{
		userManageDAO: userManageDAO,
		userDAO:       userDAO,
		roleDAO:       roleDAO,
	}
}

// GetUserList 获取用户列表（分页 + 搜索 + 筛选）
func (s *UserManageService) GetUserList(ctx context.Context, req *dto.UserListRequest) (*dto.UserListResponse, error) {
	funcName := "service.user_manage_service.GetUserList"
	logs.Info(ctx, funcName, "获取用户列表")

	users, total, err := s.userManageDAO.ListUsers(ctx, req)
	if err != nil {
		return nil, err
	}

	list := make([]dto.AdminUserInfo, 0, len(users))
	for _, user := range users {
		roles, roleErr := s.roleDAO.GetUserRoles(ctx, user.ID)
		if roleErr != nil {
			logs.Warn(ctx, funcName, "获取用户角色失败，降级为空角色列表",
				zap.Int64("user_id", user.ID), zap.Error(roleErr))
			roles = []model.Role{}
		}
		list = append(list, *s.buildAdminUserInfo(&user, roles))
	}

	return &dto.UserListResponse{
		Total: total,
		List:  list,
	}, nil
}

// GetUserDetail 获取单个用户详情（含角色信息）
func (s *UserManageService) GetUserDetail(ctx context.Context, userID int64) (*dto.AdminUserInfo, error) {
	funcName := "service.user_manage_service.GetUserDetail"
	logs.Info(ctx, funcName, "获取用户详情", zap.Int64("user_id", userID))

	user, err := s.userDAO.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	roles, roleErr := s.roleDAO.GetUserRoles(ctx, user.ID)
	if roleErr != nil {
		logs.Warn(ctx, funcName, "获取用户角色失败，降级为空角色列表",
			zap.Int64("user_id", user.ID), zap.Error(roleErr))
		roles = []model.Role{}
	}
	return s.buildAdminUserInfo(user, roles), nil
}

// UpdateUserStatus 启用/禁用用户
// adminUserID 用于防止管理员禁用自己；同时校验操作者权限等级必须高于目标用户
func (s *UserManageService) UpdateUserStatus(ctx context.Context, userID int64, status int, adminUserID int64) error {
	funcName := "service.user_manage_service.UpdateUserStatus"
	logs.Info(ctx, funcName, "更新用户状态",
		zap.Int64("target_user_id", userID),
		zap.Int("status", status),
		zap.Int64("admin_user_id", adminUserID),
	)

	if userID == adminUserID {
		return ErrCannotDisableSelf
	}

	if status != constants.UserStatusActive && status != constants.UserStatusDisabled {
		return ErrInvalidStatus
	}

	_, err := s.userDAO.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	if err := s.checkPermissionLevel(ctx, adminUserID, userID); err != nil {
		return err
	}

	if err := s.userManageDAO.UpdateUserStatus(ctx, userID, status); err != nil {
		return err
	}

	logs.Info(ctx, funcName, "用户状态更新成功",
		zap.Int64("user_id", userID),
		zap.Int("new_status", status),
	)
	return nil
}

// SetUserRoles 批量设置用户角色（事务内先清后设）
// adminUserID 用于校验操作者等级，防止越权分配高等级角色
func (s *UserManageService) SetUserRoles(ctx context.Context, userID int64, roleCodes []string, adminUserID int64) error {
	funcName := "service.user_manage_service.SetUserRoles"
	logs.Info(ctx, funcName, "设置用户角色",
		zap.Int64("target_user_id", userID),
		zap.Strings("role_codes", roleCodes),
		zap.Int64("admin_user_id", adminUserID),
	)

	_, err := s.userDAO.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	if err := s.checkPermissionLevel(ctx, adminUserID, userID); err != nil {
		return err
	}

	roles, err := s.roleDAO.FindByCodeList(ctx, roleCodes)
	if err != nil {
		return err
	}
	if len(roles) != len(roleCodes) {
		return ErrInvalidRole
	}

	adminLevel, err := s.roleDAO.GetUserMaxLevel(ctx, adminUserID)
	if err != nil {
		return err
	}
	for _, r := range roles {
		if r.Level <= adminLevel {
			logs.Warn(ctx, funcName, "尝试分配高于自身等级的角色",
				zap.String("role_code", r.Code),
				zap.Int("role_level", r.Level),
				zap.Int("admin_level", adminLevel),
			)
			return ErrCannotAssignHigherRole
		}
	}

	roleIDs := make([]int, 0, len(roles))
	for _, r := range roles {
		roleIDs = append(roleIDs, r.ID)
	}
	if err := s.roleDAO.SetUserRoles(ctx, userID, roleIDs); err != nil {
		return err
	}

	logs.Info(ctx, funcName, "用户角色设置成功",
		zap.Int64("user_id", userID),
		zap.Strings("role_codes", roleCodes),
	)
	return nil
}

// GetAllRoles 获取所有角色列表
func (s *UserManageService) GetAllRoles(ctx context.Context) ([]dto.RoleInfo, error) {
	funcName := "service.user_manage_service.GetAllRoles"
	logs.Info(ctx, funcName, "获取所有角色列表")

	roles, err := s.roleDAO.GetAllRoles(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]dto.RoleInfo, 0, len(roles))
	for _, r := range roles {
		result = append(result, dto.RoleInfo{
			Code:  r.Code,
			Name:  r.Name,
			Level: r.Level,
		})
	}
	return result, nil
}

// checkPermissionLevel 校验操作者的权限等级是否高于目标用户
// 操作者 level 必须 < 目标用户 level（数值更小 = 权限更高）
func (s *UserManageService) checkPermissionLevel(ctx context.Context, adminUserID, targetUserID int64) error {
	funcName := "service.user_manage_service.checkPermissionLevel"

	adminLevel, err := s.roleDAO.GetUserMaxLevel(ctx, adminUserID)
	if err != nil {
		logs.Error(ctx, funcName, "获取操作者权限等级失败", zap.Error(err))
		return err
	}

	targetLevel, err := s.roleDAO.GetUserMaxLevel(ctx, targetUserID)
	if err != nil {
		logs.Error(ctx, funcName, "获取目标用户权限等级失败", zap.Error(err))
		return err
	}

	if adminLevel >= targetLevel {
		logs.Warn(ctx, funcName, "权限不足",
			zap.Int64("admin_user_id", adminUserID),
			zap.Int("admin_level", adminLevel),
			zap.Int64("target_user_id", targetUserID),
			zap.Int("target_level", targetLevel),
		)
		return ErrInsufficientPermission
	}
	return nil
}

// CreateUser 管理员手动创建用户
// adminUserID 用于校验操作者权限等级，防止越权分配高等级角色
func (s *UserManageService) CreateUser(ctx context.Context, req *dto.AdminCreateUserRequest, adminUserID int64) (*dto.AdminUserInfo, error) {
	funcName := "service.user_manage_service.CreateUser"
	logs.Info(ctx, funcName, "管理员创建用户",
		zap.String("username", req.Username),
		zap.String("email", logs.MaskEmail(req.Email)),
		zap.Int64("admin_user_id", adminUserID),
	)

	existing, findErr := s.userDAO.FindByUsername(ctx, req.Username)
	if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
		return nil, findErr
	}
	if existing != nil {
		return nil, ErrUserExists
	}
	existing, findErr = s.userDAO.FindByEmail(ctx, req.Email)
	if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
		return nil, findErr
	}
	if existing != nil {
		return nil, ErrUserExists
	}

	roleCode := req.RoleCode
	if roleCode == "" {
		roleCode = constants.RoleUser
	}
	role, roleErr := s.roleDAO.FindByCode(ctx, roleCode)
	if roleErr != nil {
		return nil, ErrInvalidRole
	}

	adminLevel, err := s.roleDAO.GetUserMaxLevel(ctx, adminUserID)
	if err != nil {
		return nil, err
	}
	if role.Level <= adminLevel {
		logs.Warn(ctx, funcName, "尝试为新用户分配高于自身等级的角色",
			zap.String("role_code", roleCode),
			zap.Int("role_level", role.Level),
			zap.Int("admin_level", adminLevel),
		)
		return nil, ErrCannotAssignHigherRole
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	nickname := req.Nickname
	if nickname == "" {
		nickname = req.Username
	}

	user := &model.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Nickname:     nickname,
		Status:       constants.UserStatusActive,
	}
	if err := s.userDAO.Create(ctx, user); err != nil {
		return nil, err
	}

	if assignErr := s.roleDAO.AssignRole(ctx, user.ID, role.ID); assignErr != nil {
		logs.Warn(ctx, funcName, "角色分配失败",
			zap.Int64("user_id", user.ID), zap.Error(assignErr))
	}

	userRoles, rolesErr := s.roleDAO.GetUserRoles(ctx, user.ID)
	if rolesErr != nil {
		logs.Warn(ctx, funcName, "获取新建用户角色失败",
			zap.Int64("user_id", user.ID), zap.Error(rolesErr))
		userRoles = []model.Role{}
	}
	logs.Info(ctx, funcName, "管理员创建用户成功", zap.Int64("user_id", user.ID))
	return s.buildAdminUserInfo(user, userRoles), nil
}

// buildAdminUserInfo 从 model.User + []model.Role 构建 dto.AdminUserInfo
func (s *UserManageService) buildAdminUserInfo(user *model.User, roles []model.Role) *dto.AdminUserInfo {
	roleInfos := make([]dto.RoleInfo, 0, len(roles))
	minLevel := math.MaxInt32
	for _, r := range roles {
		roleInfos = append(roleInfos, dto.RoleInfo{
			Code:  r.Code,
			Name:  r.Name,
			Level: r.Level,
		})
		if r.Level < minLevel {
			minLevel = r.Level
		}
	}

	info := &dto.AdminUserInfo{
		ID:         user.ID,
		Username:   user.Username,
		Email:      user.Email,
		Nickname:   user.Nickname,
		Avatar:     user.Avatar,
		Gender:     user.Gender,
		Status:     user.Status,
		StatusText: constants.UserStatusMap[user.Status],
		Roles:      roleInfos,
		MaxLevel:   minLevel,
		CreatedAt:  user.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:  user.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	if user.Phone != nil {
		info.Phone = *user.Phone
	}
	if user.LastLoginAt != nil {
		info.LastLoginAt = user.LastLoginAt.Format("2006-01-02 15:04:05")
	}
	if user.LastLoginIP != nil {
		info.LastLoginIP = *user.LastLoginIP
	}
	return info
}
