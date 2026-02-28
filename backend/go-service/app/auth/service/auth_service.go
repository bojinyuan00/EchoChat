// Package service 提供 auth 模块的核心业务逻辑
package service

import (
	"context"
	"errors"

	"github.com/echochat/backend/app/auth/dao"
	"github.com/echochat/backend/app/auth/model"
	"github.com/echochat/backend/app/constants"
	"github.com/echochat/backend/app/dto"
	"github.com/echochat/backend/config"
	"github.com/echochat/backend/pkg/logs"
	"github.com/echochat/backend/pkg/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrUserAlreadyExists = errors.New("用户名或邮箱已被注册")
	ErrUserNotFound      = errors.New("用户不存在")
	ErrPasswordWrong     = errors.New("密码错误")
	ErrUserDisabled      = errors.New("账号已被禁用")
	ErrUserDeleted       = errors.New("账号已注销")
	ErrNotAdmin          = errors.New("该账号无管理员权限")
	ErrRefreshTokenType  = errors.New("无效的 Refresh Token 类型")
)

// AuthService 认证服务，处理注册、登录、Token 管理、个人信息等业务逻辑
type AuthService struct {
	userDAO    *dao.UserDAO
	roleDAO    *dao.RoleDAO
	jwtCfg     *config.JWTConfig
	tokenStore *TokenStore
}

// NewAuthService 创建认证服务实例
func NewAuthService(userDAO *dao.UserDAO, roleDAO *dao.RoleDAO, jwtCfg *config.JWTConfig, tokenStore *TokenStore) *AuthService {
	return &AuthService{
		userDAO:    userDAO,
		roleDAO:    roleDAO,
		jwtCfg:     jwtCfg,
		tokenStore: tokenStore,
	}
}

// Register 用户注册
// 流程：检查用户名/邮箱是否重复 → 加密密码 → 创建用户 → 分配默认角色 → 生成 Token
func (s *AuthService) Register(ctx context.Context, req *dto.RegisterRequest) (*dto.LoginResponse, error) {
	funcName := "service.auth_service.Register"
	logs.Info(ctx, funcName, "开始处理注册",
		zap.String("username", req.Username),
		zap.String("email", logs.MaskEmail(req.Email)),
	)

	var err error
	defer func() {
		if err != nil {
			logs.Error(ctx, funcName, "注册处理失败",
				zap.String("username", req.Username),
				zap.Error(err),
			)
		}
	}()

	// 检查用户名是否已存在
	existing, _ := s.userDAO.FindByUsername(ctx, req.Username)
	if existing != nil {
		err = ErrUserAlreadyExists
		return nil, err
	}

	// 检查邮箱是否已存在
	existing, _ = s.userDAO.FindByEmail(ctx, req.Email)
	if existing != nil {
		err = ErrUserAlreadyExists
		return nil, err
	}

	// 加密密码
	hashedPassword, hashErr := utils.HashPassword(req.Password)
	if hashErr != nil {
		err = hashErr
		return nil, err
	}

	// 创建用户
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
	if err = s.userDAO.Create(ctx, user); err != nil {
		return nil, err
	}

	// 分配默认角色（普通用户）
	defaultRole, roleErr := s.roleDAO.FindByCode(ctx, constants.RoleUser)
	if roleErr != nil {
		logs.Warn(ctx, funcName, "查找默认角色失败，跳过角色分配", zap.Error(roleErr))
	} else {
		if assignErr := s.roleDAO.AssignRole(ctx, user.ID, defaultRole.ID); assignErr != nil {
			logs.Warn(ctx, funcName, "分配默认角色失败", zap.Error(assignErr))
		}
	}

	// 获取角色列表并生成 Token
	roles, _ := s.roleDAO.GetUserRoleCodes(ctx, user.ID)
	resp, err := s.buildLoginResponse(ctx, user, roles)
	if err != nil {
		return nil, err
	}

	logs.Info(ctx, funcName, "注册成功", zap.Int64("user_id", user.ID))
	return resp, nil
}

// Login 用户登录
// 流程：查找用户 → 校验密码 → 检查账号状态 → 更新登录信息 → 生成 Token
func (s *AuthService) Login(ctx context.Context, req *dto.LoginRequest, clientIP string) (*dto.LoginResponse, error) {
	funcName := "service.auth_service.Login"
	logs.Info(ctx, funcName, "开始处理登录",
		zap.String("account", req.Account),
	)

	var err error
	defer func() {
		if err != nil {
			logs.Error(ctx, funcName, "登录处理失败",
				zap.String("account", req.Account),
				zap.Error(err),
			)
		}
	}()

	// 按用户名或邮箱查找用户
	user, findErr := s.userDAO.FindByAccount(ctx, req.Account)
	if findErr != nil {
		if errors.Is(findErr, gorm.ErrRecordNotFound) {
			err = ErrUserNotFound
		} else {
			err = findErr
		}
		return nil, err
	}

	// 校验密码
	if !utils.CheckPassword(req.Password, user.PasswordHash) {
		err = ErrPasswordWrong
		return nil, err
	}

	// 检查账号状态
	if err = s.checkUserStatus(user.Status); err != nil {
		return nil, err
	}

	// 更新最后登录信息
	_ = s.userDAO.UpdateLastLogin(ctx, user.ID, clientIP)

	// 获取角色列表并生成 Token（同时存入 Redis）
	roles, _ := s.roleDAO.GetUserRoleCodes(ctx, user.ID)
	resp, err := s.buildLoginResponse(ctx, user, roles)
	if err != nil {
		return nil, err
	}

	logs.Info(ctx, funcName, "登录成功",
		zap.Int64("user_id", user.ID),
		zap.String("username", user.Username),
	)
	return resp, nil
}

// AdminLogin 管理后台登录
// 与普通登录相同，但额外检查用户是否拥有 admin 或 super_admin 角色
func (s *AuthService) AdminLogin(ctx context.Context, req *dto.LoginRequest, clientIP string) (*dto.LoginResponse, error) {
	funcName := "service.auth_service.AdminLogin"
	logs.Info(ctx, funcName, "开始处理管理员登录",
		zap.String("account", req.Account),
	)

	resp, err := s.Login(ctx, req, clientIP)
	if err != nil {
		return nil, err
	}

	// 检查是否拥有管理员角色
	hasAdmin := false
	for _, role := range resp.User.Roles {
		if role == constants.RoleAdmin || role == constants.RoleSuperAdmin {
			hasAdmin = true
			break
		}
	}
	if !hasAdmin {
		logs.Warn(ctx, funcName, "非管理员尝试后台登录",
			zap.String("account", req.Account),
		)
		return nil, ErrNotAdmin
	}

	logs.Info(ctx, funcName, "管理员登录成功",
		zap.String("account", req.Account),
	)
	return resp, nil
}

// RefreshToken 刷新 Access Token
// 验证 Refresh Token 有效性后，重新生成一对新的 Token
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*dto.LoginResponse, error) {
	funcName := "service.auth_service.RefreshToken"

	// 解析 Refresh Token
	claims, err := utils.ParseToken(s.jwtCfg, refreshToken)
	if err != nil {
		logs.Warn(ctx, funcName, "Refresh Token 解析失败", zap.Error(err))
		return nil, err
	}

	// 验证 Token 类型
	if claims.Subject != "refresh" {
		return nil, ErrRefreshTokenType
	}

	// 验证 Refresh Token 是否与 Redis 中存储的一致
	if !s.tokenStore.ValidateRefreshToken(ctx, claims.UserID, refreshToken) {
		logs.Warn(ctx, funcName, "Refresh Token 已失效（不在 Redis 中）",
			zap.Int64("user_id", claims.UserID),
		)
		return nil, ErrRefreshTokenType
	}

	// 查找用户（确保用户仍然有效）
	user, err := s.userDAO.FindByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	if err = s.checkUserStatus(user.Status); err != nil {
		return nil, err
	}

	// 获取角色并生成新 Token（同时存入 Redis 覆盖旧 Token）
	roles, _ := s.roleDAO.GetUserRoleCodes(ctx, user.ID)
	resp, err := s.buildLoginResponse(ctx, user, roles)
	if err != nil {
		return nil, err
	}

	logs.Info(ctx, funcName, "Token 刷新成功", zap.Int64("user_id", user.ID))
	return resp, nil
}

// Logout 用户登出
// 从 Redis 中删除该用户的 Access Token 和 Refresh Token
func (s *AuthService) Logout(ctx context.Context, userID int64) error {
	funcName := "service.auth_service.Logout"
	logs.Info(ctx, funcName, "用户登出", zap.Int64("user_id", userID))
	return s.tokenStore.RemoveTokens(ctx, userID)
}

// ValidateAccessToken 校验 Access Token 是否在 Redis 中有效
// 供 JWT 中间件调用，实现有状态 JWT 验证
func (s *AuthService) ValidateAccessToken(ctx context.Context, userID int64, token string) bool {
	return s.tokenStore.ValidateAccessToken(ctx, userID, token)
}

// GetProfile 获取用户个人信息
func (s *AuthService) GetProfile(ctx context.Context, userID int64) (*dto.UserInfo, error) {
	funcName := "service.auth_service.GetProfile"
	logs.Debug(ctx, funcName, "获取用户信息", zap.Int64("user_id", userID))

	user, err := s.userDAO.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	roles, _ := s.roleDAO.GetUserRoleCodes(ctx, user.ID)

	return s.buildUserInfo(user, roles), nil
}

// UpdateProfile 更新用户个人资料
func (s *AuthService) UpdateProfile(ctx context.Context, userID int64, req *dto.UpdateProfileRequest) (*dto.UserInfo, error) {
	funcName := "service.auth_service.UpdateProfile"
	logs.Info(ctx, funcName, "更新用户资料", zap.Int64("user_id", userID))

	user, err := s.userDAO.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	if req.Nickname != "" {
		user.Nickname = req.Nickname
	}
	if req.Avatar != "" {
		user.Avatar = req.Avatar
	}
	if req.Gender != nil {
		user.Gender = *req.Gender
	}
	if req.Phone != "" {
		user.Phone = &req.Phone
	}

	if err = s.userDAO.Update(ctx, user); err != nil {
		return nil, err
	}

	roles, _ := s.roleDAO.GetUserRoleCodes(ctx, user.ID)
	logs.Info(ctx, funcName, "用户资料更新成功", zap.Int64("user_id", userID))
	return s.buildUserInfo(user, roles), nil
}

// ChangePassword 修改密码
func (s *AuthService) ChangePassword(ctx context.Context, userID int64, req *dto.ChangePasswordRequest) error {
	funcName := "service.auth_service.ChangePassword"
	logs.Info(ctx, funcName, "修改密码", zap.Int64("user_id", userID))

	user, err := s.userDAO.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	// 校验旧密码
	if !utils.CheckPassword(req.OldPassword, user.PasswordHash) {
		return ErrPasswordWrong
	}

	// 加密新密码
	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}

	user.PasswordHash = hashedPassword
	if err = s.userDAO.Update(ctx, user); err != nil {
		return err
	}

	logs.Info(ctx, funcName, "密码修改成功", zap.Int64("user_id", userID))
	return nil
}

// checkUserStatus 检查用户账号状态
func (s *AuthService) checkUserStatus(status int) error {
	switch status {
	case constants.UserStatusDisabled:
		return ErrUserDisabled
	case constants.UserStatusDeleted:
		return ErrUserDeleted
	default:
		return nil
	}
}

// buildLoginResponse 构建登录响应（生成 Token + 存入 Redis + 用户信息）
func (s *AuthService) buildLoginResponse(ctx context.Context, user *model.User, roles []string) (*dto.LoginResponse, error) {
	if roles == nil {
		roles = []string{}
	}

	token, err := utils.GenerateToken(s.jwtCfg, user.ID, user.Username, roles)
	if err != nil {
		return nil, err
	}

	refreshToken, err := utils.GenerateRefreshToken(s.jwtCfg, user.ID)
	if err != nil {
		return nil, err
	}

	// 将 Token 存入 Redis（有状态 JWT，支持主动失效和单设备登录）
	if err = s.tokenStore.SaveTokens(ctx, user.ID, token, refreshToken); err != nil {
		return nil, err
	}

	return &dto.LoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.jwtCfg.AccessExpireMin) * 60,
		User:         *s.buildUserInfo(user, roles),
	}, nil
}

// buildUserInfo 从 model.User 构建 dto.UserInfo
func (s *AuthService) buildUserInfo(user *model.User, roles []string) *dto.UserInfo {
	if roles == nil {
		roles = []string{}
	}
	info := &dto.UserInfo{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Nickname:  user.Nickname,
		Avatar:    user.Avatar,
		Gender:    user.Gender,
		Roles:     roles,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	if user.Phone != nil {
		info.Phone = *user.Phone
	}
	return info
}
