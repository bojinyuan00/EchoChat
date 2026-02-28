package middleware

import (
	"bytes"
	"io"
	"strings"
	"time"

	"github.com/echochat/backend/pkg/logs"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	maxBodyLogSize     = 4096 // 请求/响应 Body 记录的最大字节数
	maxResponseLogSize = 2048 // 响应 Body 记录的最大字节数
)

// 敏感路径：这些路径的请求 Body 中密码字段需要脱敏
var sensitivePathPrefixes = []string{
	"/api/v1/auth/login",
	"/api/v1/auth/register",
	"/api/v1/admin/auth/login",
	"/api/v1/auth/password",
}

// responseBodyWriter 包装 gin.ResponseWriter 以捕获响应 Body
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

// Logger 请求日志中间件
// 记录每个请求的完整信息：方法、路径、请求参数、状态码、耗时、IP、User-Agent
// 请求参数（Query + Body）在 INFO 级别记录（所有环境生效）
// 响应 Body 在 DEBUG 级别记录，错误响应(4xx/5xx)在 INFO 级别也记录
// 自动携带 trace_id，慢请求（>500ms）记录 WARN
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// --- 请求阶段：捕获请求参数 ---
		query := c.Request.URL.RawQuery
		reqBody := readRequestBody(c)

		// 包装 ResponseWriter 以捕获响应
		rbw := &responseBodyWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBufferString(""),
		}
		c.Writer = rbw

		c.Next()

		// --- 响应阶段：记录日志 ---
		latency := time.Since(start)
		ctx := c.Request.Context()
		funcName := "middleware.Logger"
		status := rbw.Status()
		path := c.Request.URL.Path

		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		}

		// Query 参数（始终记录）
		if query != "" {
			fields = append(fields, zap.String("query", query))
		}

		// Request Body（始终记录，敏感路径脱敏）
		if reqBody != "" {
			if isSensitivePath(path) {
				reqBody = maskSensitiveBody(reqBody)
			}
			fields = append(fields, zap.String("req_body", reqBody))
		}

		// Response Body：错误响应(4xx/5xx)始终记录，正常响应仅 DEBUG
		respBody := rbw.body.String()
		if respBody != "" {
			if status >= 400 {
				fields = append(fields, zap.String("resp_body", truncate(respBody, maxResponseLogSize)))
			} else {
				logs.Debug(ctx, funcName, "响应数据",
					zap.String("path", path),
					zap.String("resp_body", truncate(respBody, maxResponseLogSize)),
				)
			}
		}

		// 按状态分级输出
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

// readRequestBody 读取请求 Body（读完后重新填回，不影响后续 Handler）
func readRequestBody(c *gin.Context) string {
	if c.Request.Body == nil {
		return ""
	}

	// 文件上传不记录 Body
	contentType := c.GetHeader("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		return "[file upload]"
	}

	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxBodyLogSize+1))
	if err != nil {
		return "[read error]"
	}
	// 将 Body 重新填回，供后续 Handler 使用
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	if len(body) > maxBodyLogSize {
		return string(body[:maxBodyLogSize]) + "...[truncated]"
	}
	return string(body)
}

// isSensitivePath 判断是否为敏感路径（包含密码等字段的接口）
func isSensitivePath(path string) bool {
	for _, prefix := range sensitivePathPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// maskSensitiveBody 对敏感 Body 中的密码字段进行脱敏
// 简单策略：将 "password":"xxx" 替换为 "password":"***"
func maskSensitiveBody(body string) string {
	// 处理 JSON 中的 password 字段
	for _, field := range []string{"password", "old_password", "new_password", "confirm_password"} {
		for {
			key := `"` + field + `":"`
			idx := strings.Index(body, key)
			if idx < 0 {
				break
			}
			start := idx + len(key)
			end := strings.Index(body[start:], `"`)
			if end < 0 {
				break
			}
			body = body[:start] + "***" + body[start+end:]
		}
	}
	return body
}

// truncate 截断字符串到指定长度
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "...[truncated]"
}
