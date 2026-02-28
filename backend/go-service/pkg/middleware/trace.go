// Package middleware 提供 Gin HTTP 中间件
package middleware

import (
	"github.com/echochat/backend/pkg/logs"
	"github.com/gin-gonic/gin"
)

// Trace 链路追踪中间件
// 从请求头提取 X-Request-ID，不存在则自动生成
// 注入 context，后续所有日志自动携带 trace_id
// 在响应头中返回 X-Request-ID
func Trace() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader("X-Request-ID")
		if traceID == "" {
			traceID = logs.GenerateTraceID()
		}

		ctx := logs.WithTraceID(c.Request.Context(), traceID)
		c.Request = c.Request.WithContext(ctx)
		c.Header("X-Request-ID", traceID)

		c.Next()
	}
}
