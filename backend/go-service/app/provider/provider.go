// Package provider 提供全局依赖注入配置
// 使用 Wire 编译时依赖注入，集中管理所有模块的 Provider Set
package provider

import (
	"github.com/echochat/backend/config"
	"github.com/echochat/backend/pkg/db"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// App 应用根容器，持有所有基础设施组件
type App struct {
	Config *config.Config
	DB     *gorm.DB
	Redis  *redis.Client
}

// NewApp 创建应用实例
func NewApp(cfg *config.Config, gormDB *gorm.DB, redisClient *redis.Client) *App {
	return &App{
		Config: cfg,
		DB:     gormDB,
		Redis:  redisClient,
	}
}

// provideDBConfig 从全局 Config 中提取 DatabaseConfig
func provideDBConfig(cfg *config.Config) *config.DatabaseConfig {
	return &cfg.Database
}

// provideRedisConfig 从全局 Config 中提取 RedisConfig
func provideRedisConfig(cfg *config.Config) *config.RedisConfig {
	return &cfg.Redis
}

// InfraSet 基础设施层 Provider Set
var InfraSet = wire.NewSet(
	provideDBConfig,
	provideRedisConfig,
	db.NewPostgres,
	db.NewRedis,
	NewApp,
)
