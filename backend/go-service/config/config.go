// Package config 提供应用配置管理功能
// 使用 Viper 读取 YAML 配置文件，支持环境变量覆盖
package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config 应用全局配置结构体
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Log      LogConfig      `mapstructure:"log"`
}

// ServerConfig HTTP 服务配置
type ServerConfig struct {
	Port int    `mapstructure:"port"` // 监听端口
	Mode string `mapstructure:"mode"` // 运行模式: debug/release
}

// DatabaseConfig PostgreSQL 数据库配置
type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	DBName       string `mapstructure:"dbname"`
	SSLMode      string `mapstructure:"sslmode"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"` // 最大空闲连接数
	MaxOpenConns int    `mapstructure:"max_open_conns"` // 最大打开连接数
}

// DSN 生成 PostgreSQL 连接字符串
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
	)
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"` // 数据库编号
}

// Addr 生成 Redis 连接地址
func (r *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// JWTConfig JWT 认证配置
type JWTConfig struct {
	Secret           string `mapstructure:"secret"`             // 签名密钥
	AccessExpireMin  int    `mapstructure:"access_expire_min"`  // Access Token 有效期（分钟）
	RefreshExpireDay int    `mapstructure:"refresh_expire_day"` // Refresh Token 有效期（天）
	Issuer           string `mapstructure:"issuer"`             // 签发者
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string        `mapstructure:"level"`       // debug/info/warn/error
	Format     string        `mapstructure:"format"`      // text(开发)/json(生产)
	OutputPath string        `mapstructure:"output_path"` // stdout 或文件路径
	File       LogFileConfig `mapstructure:"file"`        // 日志文件轮转配置
}

// LogFileConfig 日志文件轮转配置（基于 lumberjack）
type LogFileConfig struct {
	Enable     bool   `mapstructure:"enable"`      // 是否启用文件日志
	Dir        string `mapstructure:"dir"`          // 日志文件目录
	MaxSize    int    `mapstructure:"max_size"`     // 单个文件最大大小（MB），超过后自动切割
	MaxBackups int    `mapstructure:"max_backups"`  // 保留的旧日志文件最大数量
	MaxAge     int    `mapstructure:"max_age"`      // 旧日志文件保留天数
	Compress   bool   `mapstructure:"compress"`     // 是否压缩归档的旧日志文件
}

// Load 加载配置文件并返回 Config 实例
// configPath 为配置文件所在目录，configName 为文件名（不含扩展名）
func Load(configPath, configName string) (*Config, error) {
	v := viper.New()
	v.SetConfigName(configName)
	v.SetConfigType("yaml")
	v.AddConfigPath(configPath)

	// 环境变量覆盖：ECHOCHAT_SERVER_PORT → server.port
	v.SetEnvPrefix("ECHOCHAT")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	return &cfg, nil
}
