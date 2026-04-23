package logs

import (
	"context"

	"github.com/google/uuid"
)

type contextKey string

const traceIDKey contextKey = "trace_id"

// GenerateTraceID 生成唯一的链路追踪 ID（UUID v4）
func GenerateTraceID() string {
	return uuid.New().String()
}

// WithTraceID 将 trace_id 注入 context
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// GetTraceID 从 context 提取 trace_id，不存在则返回空字符串
func GetTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if traceID, ok := ctx.Value(traceIDKey).(string); ok {
		return traceID
	}
	return ""
}

// DetachContext 剥离父 ctx 的 Deadline/Cancel 但保留 trace_id
// 用于启动后台 goroutine 时避免随 HTTP 请求结束被取消，同时保持日志链路追踪连续性
// 典型使用：go func() { ... }(logs.DetachContext(ctx))
// Task 16 P1-3：后台 goroutine trace_id 保留 新增
func DetachContext(ctx context.Context) context.Context {
	bg := context.Background()
	if traceID := GetTraceID(ctx); traceID != "" {
		return WithTraceID(bg, traceID)
	}
	return bg
}
