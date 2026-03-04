// Package controller 提供文件上传模块的 HTTP 接口处理
package controller

import (
	"github.com/echochat/backend/app/file/service"
	"github.com/echochat/backend/pkg/middleware"
	"github.com/echochat/backend/pkg/utils"
	"github.com/gin-gonic/gin"
)

const maxUploadSize = 10 << 20 // 10 MB

// FileController 文件上传控制器
type FileController struct {
	fileService *service.FileService
}

// NewFileController 创建文件上传控制器
func NewFileController(fileService *service.FileService) *FileController {
	return &FileController{fileService: fileService}
}

// Upload 处理文件上传请求
// POST /api/v1/upload
// 支持 multipart/form-data，字段名为 "file"
func (ctl *FileController) Upload(c *gin.Context) {
	ctx := c.Request.Context()
	_, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		utils.ResponseBadRequest(c, "请选择要上传的文件")
		return
	}

	if fileHeader.Size > maxUploadSize {
		utils.ResponseBadRequest(c, "文件大小不能超过 10MB")
		return
	}

	result, err := ctl.fileService.Upload(ctx, fileHeader)
	if err != nil {
		ctl.handleError(c, err, "文件上传失败")
		return
	}

	utils.ResponseOK(c, result)
}

// handleError 统一业务错误映射
// 已知业务错误 → 返回 Service 层定义的具体提示
// 未知错误 → 返回 fallbackMsg（未传则默认"服务器内部错误"）
func (ctl *FileController) handleError(c *gin.Context, err error, fallbackMsg ...string) {
	switch err {
	case service.ErrFileOpen:
		utils.ResponseBadRequest(c, err.Error())
	case service.ErrUploadFailed:
		utils.ResponseError(c, err.Error())
	default:
		msg := "服务器内部错误"
		if len(fallbackMsg) > 0 && fallbackMsg[0] != "" {
			msg = fallbackMsg[0]
		}
		utils.ResponseError(c, msg)
	}
}
