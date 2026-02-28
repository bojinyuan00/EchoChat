package db

import (
	"context"
	"fmt"

	"github.com/echochat/backend/config"
	"github.com/echochat/backend/pkg/logs"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Redis 全局 Redis 客户端实例
var Redis *redis.Client

// NewRedis 初始化 Redis 客户端连接
func NewRedis(cfg *config.RedisConfig) (*redis.Client, error) {
	funcName := "db.NewRedis"
	ctx := context.Background()

	logs.Info(ctx, funcName, "正在连接 Redis",
		zap.String("addr", cfg.Addr()),
		zap.Int("db", cfg.DB),
	)

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		logs.Error(ctx, funcName, "Redis 连接失败", zap.Error(err))
		return nil, fmt.Errorf("连接 Redis 失败: %w", err)
	}

	Redis = client
	logs.Info(ctx, funcName, "Redis 连接成功")
	return client, nil
}
