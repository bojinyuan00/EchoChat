package middleware

import (
	"context"
	"strings"

	"github.com/echochat/backend/config"
	"github.com/echochat/backend/pkg/logs"
	"github.com/echochat/backend/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	ContextKeyUserID   = "user_id"   // Context 中存储当前用户 ID 的 Key
	ContextKeyUsername = "username"   // Context 中存储当前用户名的 Key
	ContextKeyRoles    = "roles"     // Context 中存储当前用户角色列表的 Key
)

// TokenValidator Token 有效性校验接口
// 由 AuthService 实现，中间件通过此接口检查 Token 是否在 Redis 中有效
type TokenValidator interface {
	ValidateAccessToken(ctx context.Context, userID int64, token string) bool
}

// JWTAuth JWT 认证中间件（有状态 JWT）
// 验证流程：解析 Token → 检查类型 → 校验 Redis 有效性 → 注入用户信息
func JWTAuth(jwtCfg *config.JWTConfig, validator TokenValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		funcName := "middleware.JWTAuth"
		ctx := c.Request.Context()

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.ResponseUnauthorized(c, "缺少认证信息")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			utils.ResponseUnauthorized(c, "认证格式错误，应为 Bearer {token}")
			c.Abort()
			return
		}

		tokenStr := parts[1]
		claims, err := utils.ParseToken(jwtCfg, tokenStr)
		if err != nil {
			logs.Warn(ctx, funcName, "Token 验证失败",
				zap.Error(err),
				zap.String("ip", c.ClientIP()),
			)
			utils.ResponseUnauthorized(c, "认证已过期或无效，请重新登录")
			c.Abort()
			return
		}

		if claims.Subject != "access" {
			utils.ResponseUnauthorized(c, "无效的 Token 类型")
			c.Abort()
			return
		}

		// 校验 Token 是否在 Redis 中有效（有状态 JWT 核心逻辑）
		if !validator.ValidateAccessToken(ctx, claims.UserID, tokenStr) {
			logs.Warn(ctx, funcName, "Token 已失效（已登出或被覆盖）",
				zap.Int64("user_id", claims.UserID),
				zap.String("ip", c.ClientIP()),
			)
			utils.ResponseUnauthorized(c, "认证已失效，请重新登录")
			c.Abort()
			return
		}

		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyUsername, claims.Username)
		c.Set(ContextKeyRoles, claims.Roles)

		c.Next()
	}
}

// RequireRole 角色权限检查中间件
// 检查当前用户是否拥有指定角色之一（OR 逻辑），需在 JWTAuth 之后使用
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		funcName := "middleware.RequireRole"
		ctx := c.Request.Context()

		userRoles, exists := c.Get(ContextKeyRoles)
		if !exists {
			utils.ResponseForbidden(c, "无权限访问")
			c.Abort()
			return
		}

		roleList, ok := userRoles.([]string)
		if !ok {
			utils.ResponseForbidden(c, "无权限访问")
			c.Abort()
			return
		}

		for _, required := range roles {
			for _, userRole := range roleList {
				if userRole == required {
					c.Next()
					return
				}
			}
		}

		userID, _ := c.Get(ContextKeyUserID)
		logs.Warn(ctx, funcName, "角色权限不足",
			zap.Any("user_id", userID),
			zap.Strings("required", roles),
			zap.Strings("actual", roleList),
		)
		utils.ResponseForbidden(c, "权限不足，需要角色: "+strings.Join(roles, " 或 "))
		c.Abort()
	}
}

// GetCurrentUserID 从 Gin Context 获取当前登录用户 ID
func GetCurrentUserID(c *gin.Context) (int64, bool) {
	val, exists := c.Get(ContextKeyUserID)
	if !exists {
		return 0, false
	}
	userID, ok := val.(int64)
	return userID, ok
}

// GetCurrentUsername 从 Gin Context 获取当前登录用户名
func GetCurrentUsername(c *gin.Context) (string, bool) {
	val, exists := c.Get(ContextKeyUsername)
	if !exists {
		return "", false
	}
	username, ok := val.(string)
	return username, ok
}
