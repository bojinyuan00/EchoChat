package service

import (
	"context"
	"fmt"
	"time"

	"github.com/echochat/backend/config"
	"github.com/echochat/backend/pkg/logs"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Redis Key 前缀，按 client_type 隔离前台和管理端的 Token
// 格式：echo:auth:token:{client_type}:{user_id}
// 例如：echo:auth:token:frontend:1 / echo:auth:token:admin:1
const (
	keyPrefixAccessToken  = "echo:auth:token:"   // echo:auth:token:{client_type}:{user_id}
	keyPrefixRefreshToken = "echo:auth:refresh:"  // echo:auth:refresh:{client_type}:{user_id}
)

// TokenStore 管理 Token 在 Redis 中的存取
// 实现有状态 JWT：登录存入、验证时校验、登出时删除
type TokenStore struct {
	redis  *redis.Client
	jwtCfg *config.JWTConfig
}

// NewTokenStore 创建 TokenStore 实例
func NewTokenStore(redisClient *redis.Client, jwtCfg *config.JWTConfig) *TokenStore {
	return &TokenStore{
		redis:  redisClient,
		jwtCfg: jwtCfg,
	}
}

// SaveTokens 将 Access Token 和 Refresh Token 存入 Redis
// 每次登录/注册时调用，覆盖同一 clientType 下的旧 Token
// clientType 用于隔离前台（frontend）和管理端（admin）的 Token
func (s *TokenStore) SaveTokens(ctx context.Context, userID int64, clientType, accessToken, refreshToken string) error {
	funcName := "service.token_store.SaveTokens"

	accessKey := fmt.Sprintf("%s%s:%d", keyPrefixAccessToken, clientType, userID)
	refreshKey := fmt.Sprintf("%s%s:%d", keyPrefixRefreshToken, clientType, userID)

	accessTTL := time.Duration(s.jwtCfg.AccessExpireMin) * time.Minute
	refreshTTL := time.Duration(s.jwtCfg.RefreshExpireDay) * 24 * time.Hour

	pipe := s.redis.Pipeline()
	pipe.Set(ctx, accessKey, accessToken, accessTTL)
	pipe.Set(ctx, refreshKey, refreshToken, refreshTTL)

	if _, err := pipe.Exec(ctx); err != nil {
		logs.Error(ctx, funcName, "保存 Token 到 Redis 失败",
			zap.Int64("user_id", userID),
			zap.String("client_type", clientType),
			zap.Error(err),
		)
		return err
	}

	logs.Debug(ctx, funcName, "Token 已存入 Redis",
		zap.Int64("user_id", userID),
		zap.String("client_type", clientType),
		zap.Duration("access_ttl", accessTTL),
		zap.Duration("refresh_ttl", refreshTTL),
	)
	return nil
}

// ValidateAccessToken 校验 Access Token 是否与 Redis 中存储的一致
// clientType 用于定位正确的 Redis key（前台 vs 管理端）
func (s *TokenStore) ValidateAccessToken(ctx context.Context, userID int64, clientType, token string) bool {
	funcName := "service.token_store.ValidateAccessToken"

	key := fmt.Sprintf("%s%s:%d", keyPrefixAccessToken, clientType, userID)
	stored, err := s.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		logs.Debug(ctx, funcName, "Token 不存在（已登出或过期）",
			zap.Int64("user_id", userID),
			zap.String("client_type", clientType),
		)
		return false
	}
	if err != nil {
		logs.Error(ctx, funcName, "Redis 查询 Token 失败",
			zap.Int64("user_id", userID),
			zap.String("client_type", clientType),
			zap.Error(err),
		)
		return false
	}

	return stored == token
}

// ValidateRefreshToken 校验 Refresh Token 是否与 Redis 中存储的一致
func (s *TokenStore) ValidateRefreshToken(ctx context.Context, userID int64, clientType, token string) bool {
	key := fmt.Sprintf("%s%s:%d", keyPrefixRefreshToken, clientType, userID)
	stored, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		return false
	}
	return stored == token
}

// RemoveTokens 从 Redis 删除指定 clientType 下用户的 Token（登出时调用）
func (s *TokenStore) RemoveTokens(ctx context.Context, userID int64, clientType string) error {
	funcName := "service.token_store.RemoveTokens"

	accessKey := fmt.Sprintf("%s%s:%d", keyPrefixAccessToken, clientType, userID)
	refreshKey := fmt.Sprintf("%s%s:%d", keyPrefixRefreshToken, clientType, userID)

	if err := s.redis.Del(ctx, accessKey, refreshKey).Err(); err != nil {
		logs.Error(ctx, funcName, "删除 Token 失败",
			zap.Int64("user_id", userID),
			zap.String("client_type", clientType),
			zap.Error(err),
		)
		return err
	}

	logs.Info(ctx, funcName, "Token 已从 Redis 删除",
		zap.Int64("user_id", userID),
		zap.String("client_type", clientType),
	)
	return nil
}
