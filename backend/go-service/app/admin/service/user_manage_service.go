// Package service 提供 admin 模块的核心业务逻辑
package service

import (
	"context"
	"errors"

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
	ErrUserNotFound    = errors.New("用户不存在")
	ErrUserExists      = errors.New("用户名或邮箱已被注册")
	ErrInvalidStatus   = errors.New("无效的用户状态")
	ErrInvalidRole     = errors.New("无效的角色代码")
	ErrCannotDisableSelf = errors.New("不能禁用自己的账号")
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
		roles, roleErr := s.roleDAO.GetUserRoleCodes(ctx, user.ID)
		if roleErr != nil {
			logs.Warn(ctx, funcName, "获取用户角色失败，降级为空角色列表",
				zap.Int64("user_id", user.ID), zap.Error(roleErr))
			roles = []string{}
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

	roles, roleErr := s.roleDAO.GetUserRoleCodes(ctx, user.ID)
	if roleErr != nil {
		logs.Warn(ctx, funcName, "获取用户角色失败，降级为空角色列表",
			zap.Int64("user_id", user.ID), zap.Error(roleErr))
		roles = []string{}
	}
	return s.buildAdminUserInfo(user, roles), nil
}

// UpdateUserStatus 启用/禁用用户
// adminUserID 用于防止管理员禁用自己
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

	// 确认用户存在
	_, err := s.userDAO.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
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

// AssignUserRole 分配角色给用户
func (s *UserManageService) AssignUserRole(ctx context.Context, userID int64, roleCode string) error {
	funcName := "service.user_manage_service.AssignUserRole"
	logs.Info(ctx, funcName, "分配角色",
		zap.Int64("user_id", userID),
		zap.String("role_code", roleCode),
	)

	// 验证用户存在
	_, err := s.userDAO.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	// 查找角色
	role, err := s.roleDAO.FindByCode(ctx, roleCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrInvalidRole
		}
		return err
	}

	// 分配角色（RoleDAO.AssignRole 内部会处理重复分配）
	if err := s.roleDAO.AssignRole(ctx, userID, role.ID); err != nil {
		return err
	}

	logs.Info(ctx, funcName, "角色分配成功",
		zap.Int64("user_id", userID),
		zap.String("role_code", roleCode),
	)
	return nil
}

// CreateUser 管理员手动创建用户
func (s *UserManageService) CreateUser(ctx context.Context, req *dto.AdminCreateUserRequest) (*dto.AdminUserInfo, error) {
	funcName := "service.user_manage_service.CreateUser"
	logs.Info(ctx, funcName, "管理员创建用户",
		zap.String("username", req.Username),
		zap.String("email", logs.MaskEmail(req.Email)),
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

	// 加密密码
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

	roleCode := req.RoleCode
	if roleCode == "" {
		roleCode = constants.RoleUser
	}
	role, roleErr := s.roleDAO.FindByCode(ctx, roleCode)
	if roleErr != nil {
		logs.Warn(ctx, funcName, "角色查找失败，将跳过角色分配",
			zap.String("role_code", roleCode), zap.Error(roleErr))
	} else {
		if assignErr := s.roleDAO.AssignRole(ctx, user.ID, role.ID); assignErr != nil {
			logs.Warn(ctx, funcName, "角色分配失败",
				zap.Int64("user_id", user.ID), zap.Error(assignErr))
		}
	}

	roles, rolesErr := s.roleDAO.GetUserRoleCodes(ctx, user.ID)
	if rolesErr != nil {
		logs.Warn(ctx, funcName, "获取新建用户角色失败",
			zap.Int64("user_id", user.ID), zap.Error(rolesErr))
		roles = []string{}
	}
	logs.Info(ctx, funcName, "管理员创建用户成功", zap.Int64("user_id", user.ID))
	return s.buildAdminUserInfo(user, roles), nil
}

// buildAdminUserInfo 从 model.User 构建 dto.AdminUserInfo
func (s *UserManageService) buildAdminUserInfo(user *model.User, roles []string) *dto.AdminUserInfo {
	if roles == nil {
		roles = []string{}
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
		Roles:      roles,
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
