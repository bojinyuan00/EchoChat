// Package db 提供数据库连接管理
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/echochat/backend/config"
	"github.com/echochat/backend/pkg/logs"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// GormDB 全局 GORM 数据库实例
var GormDB *gorm.DB

// zapGormLogger 将 GORM 日志适配到 zap
type zapGormLogger struct{}

func (l *zapGormLogger) LogMode(logger.LogLevel) logger.Interface { return l }

func (l *zapGormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	logs.Info(ctx, "gorm", fmt.Sprintf(msg, data...))
}

func (l *zapGormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	logs.Warn(ctx, "gorm", fmt.Sprintf(msg, data...))
}

func (l *zapGormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	logs.Error(ctx, "gorm", fmt.Sprintf(msg, data...))
}

func (l *zapGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()
	fields := []zap.Field{
		zap.Duration("latency", elapsed),
		zap.Int64("rows", rows),
		zap.String("sql", sql),
	}
	if err != nil {
		fields = append(fields, zap.Error(err))
		logs.Error(ctx, "gorm.trace", "SQL执行失败", fields...)
		return
	}
	if elapsed > 200*time.Millisecond {
		logs.Warn(ctx, "gorm.trace", "慢SQL告警", fields...)
		return
	}
	logs.Debug(ctx, "gorm.trace", "SQL执行", fields...)
}

// NewPostgres 初始化 PostgreSQL 数据库连接
func NewPostgres(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	funcName := "db.NewPostgres"
	ctx := context.Background()

	logs.Info(ctx, funcName, "正在连接 PostgreSQL",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("dbname", cfg.DBName),
	)

	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger: &zapGormLogger{},
	})
	if err != nil {
		logs.Error(ctx, funcName, "PostgreSQL 连接失败", zap.Error(err))
		return nil, fmt.Errorf("连接 PostgreSQL 失败: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取底层 sql.DB 失败: %w", err)
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := sqlDB.Ping(); err != nil {
		logs.Error(ctx, funcName, "PostgreSQL Ping 失败", zap.Error(err))
		return nil, fmt.Errorf("ping PostgreSQL 失败: %w", err)
	}

	GormDB = db
	logs.Info(ctx, funcName, "PostgreSQL 连接成功")
	return db, nil
}
