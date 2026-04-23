// Package logs 提供基于 zap 的结构化日志封装
// 支持从 context 中提取 trace_id 自动附加到每条日志
// 开发环境输出彩色可读文本，生产环境输出 JSON 结构化格式
// 支持日志文件轮转（按大小切割、自动归档、过期清理）
package logs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/echochat/backend/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var globalLogger *zap.Logger

// Init 初始化全局日志实例
// 根据 LogConfig 配置决定输出目标：控制台、文件、或两者同时输出
func Init(cfg *config.LogConfig) error {
	lvl := parseLevel(cfg.Level)

	// 控制台 encoder（始终保留控制台输出）
	consoleEncoder := buildEncoder(cfg.Format)

	var cores []zapcore.Core

	// 控制台输出（始终启用）
	consoleSyncer := zapcore.AddSync(os.Stdout)
	cores = append(cores, zapcore.NewCore(consoleEncoder, consoleSyncer, lvl))

	// 文件输出（按配置启用）
	if cfg.File.Enable && cfg.File.Dir != "" {
		if err := os.MkdirAll(cfg.File.Dir, 0755); err != nil {
			return fmt.Errorf("创建日志目录失败 [%s]: %w", cfg.File.Dir, err)
		}

		// 应用日志文件（全量日志）
		allLogWriter := &lumberjack.Logger{
			Filename:   filepath.Join(cfg.File.Dir, "app.log"),
			MaxSize:    cfg.File.MaxSize,
			MaxBackups: cfg.File.MaxBackups,
			MaxAge:     cfg.File.MaxAge,
			Compress:   cfg.File.Compress,
			LocalTime:  true,
		}

		// 错误日志文件（仅 WARN 及以上），便于快速定位问题
		errorLogWriter := &lumberjack.Logger{
			Filename:   filepath.Join(cfg.File.Dir, "error.log"),
			MaxSize:    cfg.File.MaxSize,
			MaxBackups: cfg.File.MaxBackups,
			MaxAge:     cfg.File.MaxAge,
			Compress:   cfg.File.Compress,
			LocalTime:  true,
		}

		// 文件始终用 JSON 格式，便于后期接入 ELK/Loki
		fileEncoder := buildEncoder("json")

		cores = append(cores,
			zapcore.NewCore(fileEncoder, zapcore.AddSync(allLogWriter), lvl),
			zapcore.NewCore(fileEncoder, zapcore.AddSync(errorLogWriter), zap.WarnLevel),
		)
	}

	core := zapcore.NewTee(cores...)
	globalLogger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	return nil
}

func buildEncoder(format string) zapcore.Encoder {
	if format == "json" {
		encoderConfig := zap.NewProductionEncoderConfig()
		encoderConfig.TimeKey = "ts"
		encoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format("2006-01-02 15:04:05"))
		}
		encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		return zapcore.NewJSONEncoder(encoderConfig)
	}

	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02 15:04:05"))
	}
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	return zapcore.NewConsoleEncoder(encoderConfig)
}

// Debug 输出 DEBUG 级别日志
func Debug(ctx context.Context, funcName, msg string, fields ...zap.Field) {
	globalLogger.Debug(msg, withContext(ctx, funcName, fields)...)
}

// Info 输出 INFO 级别日志
func Info(ctx context.Context, funcName, msg string, fields ...zap.Field) {
	globalLogger.Info(msg, withContext(ctx, funcName, fields)...)
}

// Warn 输出 WARN 级别日志
func Warn(ctx context.Context, funcName, msg string, fields ...zap.Field) {
	globalLogger.Warn(msg, withContext(ctx, funcName, fields)...)
}

// Error 输出 ERROR 级别日志
func Error(ctx context.Context, funcName, msg string, fields ...zap.Field) {
	globalLogger.Error(msg, withContext(ctx, funcName, fields)...)
}

// Fatal 输出 FATAL 级别日志并退出进程
func Fatal(ctx context.Context, funcName, msg string, fields ...zap.Field) {
	globalLogger.Fatal(msg, withContext(ctx, funcName, fields)...)
}

// Sync 刷新日志缓冲区，应在程序退出时调用
func Sync() {
	if globalLogger != nil {
		_ = globalLogger.Sync()
	}
}

// withContext 从 context 提取 trace_id 和函数名，合并到日志字段中
func withContext(ctx context.Context, funcName string, fields []zap.Field) []zap.Field {
	traceID := GetTraceID(ctx)
	result := make([]zap.Field, 0, len(fields)+2)
	if traceID != "" {
		result = append(result, zap.String("trace_id", traceID))
	}
	if funcName != "" {
		result = append(result, zap.String("func", funcName))
	}
	result = append(result, fields...)
	return result
}

func parseLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// MaskEmail 邮箱脱敏：zh***@example.com
func MaskEmail(email string) string {
	at := strings.Index(email, "@")
	if at <= 0 {
		return "***"
	}
	prefix := email[:at]
	if len(prefix) <= 2 {
		return prefix[:1] + "***" + email[at:]
	}
	return prefix[:2] + "***" + email[at:]
}

// MaskPhone 手机号脱敏：138****8000
func MaskPhone(phone string) string {
	if len(phone) < 7 {
		return "***"
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
}

// MaskToken Token 脱敏：只显示前后各 4 位
func MaskToken(token string) string {
	if len(token) <= 8 {
		return "***"
	}
	return token[:4] + "..." + token[len(token)-4:]
}
