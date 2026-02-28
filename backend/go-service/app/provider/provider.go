// Package provider 提供全局依赖注入配置
// 使用 Wire 编译时依赖注入，集中管理所有模块的 Provider Set
package provider

import (
	"github.com/echochat/backend/app/auth/controller"
	"github.com/echochat/backend/app/auth/service"
	"github.com/echochat/backend/config"
	"github.com/echochat/backend/pkg/db"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// App 应用根容器，持有基础设施组件和各模块的 Controller/Service
type App struct {
	Config             *config.Config
	DB                 *gorm.DB
	Redis              *redis.Client
	AuthService        *service.AuthService              // Auth 认证服务
	AuthController     *controller.AuthController        // 前台认证控制器
	AdminAuthController *controller.AdminAuthController  // 后台认证控制器
}

// NewApp 创建应用实例
func NewApp(
	cfg *config.Config,
	gormDB *gorm.DB,
	redisClient *redis.Client,
	authService *service.AuthService,
	authCtrl *controller.AuthController,
	adminAuthCtrl *controller.AdminAuthController,
) *App {
	return &App{
		Config:             cfg,
		DB:                 gormDB,
		Redis:              redisClient,
		AuthService:        authService,
		AuthController:     authCtrl,
		AdminAuthController: adminAuthCtrl,
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

// provideJWTConfig 从全局 Config 中提取 JWTConfig
func provideJWTConfig(cfg *config.Config) *config.JWTConfig {
	return &cfg.JWT
}

// InfraSet 基础设施层 Provider Set
var InfraSet = wire.NewSet(
	provideDBConfig,
	provideRedisConfig,
	provideJWTConfig,
	db.NewPostgres,
	db.NewRedis,
	NewApp,
)
