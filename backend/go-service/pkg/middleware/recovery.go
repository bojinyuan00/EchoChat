package middleware

import (
	"net/http"
	"runtime/debug"
	"time"

	"github.com/echochat/backend/pkg/logs"
	"github.com/echochat/backend/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Recovery Panic 恢复中间件
// 捕获 panic 后记录 ERROR 日志含堆栈信息，返回 500 响应
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				ctx := c.Request.Context()
				logs.Error(ctx, "middleware.Recovery", "服务发生 panic",
					zap.Any("error", r),
					zap.String("stack", string(debug.Stack())),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
				)

				c.AbortWithStatusJSON(http.StatusInternalServerError, utils.Response{
					Code:    500,
					Message: "服务内部错误",
					TraceID: logs.GetTraceID(ctx),
					Time:    time.Now().Format("2006-01-02 15:04:05"),
				})
			}
		}()
		c.Next()
	}
}
