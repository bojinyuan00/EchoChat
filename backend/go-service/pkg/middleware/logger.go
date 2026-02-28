package middleware

import (
	"bytes"
	"strings"
	"time"

	"github.com/echochat/backend/pkg/logs"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	maxResponseLogSize = 2048 // 错误响应 Body 记录的最大字节数
)

// responseBodyWriter 包装 gin.ResponseWriter 以捕获响应 Body（仅用于错误响应记录）
type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseBodyWriter) Write(b []byte) (int, error) {
	if w.body.Len() < maxResponseLogSize {
		remaining := maxResponseLogSize - w.body.Len()
		if len(b) > remaining {
			w.body.Write(b[:remaining])
		} else {
			w.body.Write(b)
		}
	}
	return w.ResponseWriter.Write(b)
}

// Logger 请求日志中间件（Access Log 层）
//
// 职责：记录 HTTP 请求级元信息，不记录请求 Body（由 Controller 层以结构化参数形式记录）
// 记录字段：method / path / handler / status / latency / ip / user_agent / query
// 错误响应（4xx/5xx）额外记录响应 Body，便于排查接口返回内容
//
// 分层日志策略（符合社区最佳实践）：
//   - 中间件层：HTTP 元信息 + 错误响应
//   - Controller 层：结构化请求参数（ShouldBindJSON 后，caller 准确指向业务代码）
//   - Service/DAO 层：业务逻辑关键节点和异常
//   - 通过 trace_id 串联同一请求的所有层级日志
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		query := c.Request.URL.RawQuery

		rbw := &responseBodyWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBufferString(""),
		}
		c.Writer = rbw

		c.Next()

		latency := time.Since(start)
		ctx := c.Request.Context()
		funcName := "middleware.Logger"
		status := rbw.Status()
		path := c.Request.URL.Path

		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("handler", c.HandlerName()),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		}

		if query != "" {
			fields = append(fields, zap.String("query", query))
		}

		// 错误响应(4xx/5xx)：额外记录响应 Body
		if status >= 400 {
			respBody := rbw.body.String()
			if respBody != "" {
				fields = append(fields, zap.String("resp_body", truncate(respBody, maxResponseLogSize)))
			}
		}

		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("error", c.Errors.String()))
			logs.Error(ctx, funcName, "请求处理异常", fields...)
			return
		}

		if status >= 500 {
			logs.Error(ctx, funcName, "服务器错误", fields...)
			return
		}

		if status >= 400 {
			logs.Warn(ctx, funcName, "客户端错误", fields...)
			return
		}

		if latency > 500*time.Millisecond {
			logs.Warn(ctx, funcName, "慢请求告警", fields...)
			return
		}

		logs.Info(ctx, funcName, "请求完成", fields...)
	}
}

// truncate 截断字符串到指定长度
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "...[truncated]"
}

// --- 以下工具函数保留供 Controller 层使用 ---

// IsSensitivePath 判断是否为敏感路径（包含密码等字段的接口）
// Controller 层记录参数前可调用此函数决定是否脱敏
func IsSensitivePath(path string) bool {
	for _, prefix := range sensitivePathPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

var sensitivePathPrefixes = []string{
	"/api/v1/auth/login",
	"/api/v1/auth/register",
	"/api/v1/admin/auth/login",
	"/api/v1/auth/password",
}
