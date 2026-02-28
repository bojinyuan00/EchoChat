// Package utils 提供通用工具函数
package utils

import (
	"net/http"
	"time"

	"github.com/echochat/backend/pkg/logs"
	"github.com/gin-gonic/gin"
)

// Response 统一 API 响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	TraceID string      `json:"trace_id,omitempty"`
	Time    string      `json:"time"`
}

// ResponseOK 成功响应 200
func ResponseOK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
		TraceID: logs.GetTraceID(c.Request.Context()),
		Time:    time.Now().Format("2006-01-02 15:04:05"),
	})
}

// ResponseCreated 创建成功响应 201
func ResponseCreated(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Code:    0,
		Message: "created",
		Data:    data,
		TraceID: logs.GetTraceID(c.Request.Context()),
		Time:    time.Now().Format("2006-01-02 15:04:05"),
	})
}

// ResponseBadRequest 参数错误响应 400
func ResponseBadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, Response{
		Code:    400,
		Message: message,
		TraceID: logs.GetTraceID(c.Request.Context()),
		Time:    time.Now().Format("2006-01-02 15:04:05"),
	})
}

// ResponseUnauthorized 未认证响应 401
func ResponseUnauthorized(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, Response{
		Code:    401,
		Message: message,
		TraceID: logs.GetTraceID(c.Request.Context()),
		Time:    time.Now().Format("2006-01-02 15:04:05"),
	})
}

// ResponseForbidden 无权限响应 403
func ResponseForbidden(c *gin.Context, message string) {
	c.JSON(http.StatusForbidden, Response{
		Code:    403,
		Message: message,
		TraceID: logs.GetTraceID(c.Request.Context()),
		Time:    time.Now().Format("2006-01-02 15:04:05"),
	})
}

// ResponseNotFound 资源不存在响应 404
func ResponseNotFound(c *gin.Context, message string) {
	c.JSON(http.StatusNotFound, Response{
		Code:    404,
		Message: message,
		TraceID: logs.GetTraceID(c.Request.Context()),
		Time:    time.Now().Format("2006-01-02 15:04:05"),
	})
}

// ResponseError 服务器内部错误响应 500
func ResponseError(c *gin.Context, message string) {
	c.JSON(http.StatusInternalServerError, Response{
		Code:    500,
		Message: message,
		TraceID: logs.GetTraceID(c.Request.Context()),
		Time:    time.Now().Format("2006-01-02 15:04:05"),
	})
}
