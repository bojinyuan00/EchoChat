package middleware

import (
	"time"

	"github.com/echochat/backend/pkg/logs"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Logger 请求日志中间件
// 记录每个请求的完整信息：方法、路径、状态码、耗时、IP、User-Agent
// 自动携带 trace_id，慢请求（>500ms）记录 WARN
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		latency := time.Since(start)
		ctx := c.Request.Context()
		funcName := "middleware.Logger"

		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", latency),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		}

		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("error", c.Errors.String()))
			logs.Error(ctx, funcName, "请求处理异常", fields...)
			return
		}

		if latency > 500*time.Millisecond {
			logs.Warn(ctx, funcName, "慢请求告警", fields...)
			return
		}

		logs.Info(ctx, funcName, "请求完成", fields...)
	}
}
