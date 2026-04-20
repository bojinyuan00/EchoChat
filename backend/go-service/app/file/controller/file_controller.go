// Package controller 提供文件上传模块的 HTTP 接口处理
package controller

import (
	"strconv"

	"github.com/echochat/backend/app/file/service"
	"github.com/echochat/backend/pkg/middleware"
	"github.com/echochat/backend/pkg/utils"
	"github.com/gin-gonic/gin"
)

const (
	maxUploadSize      = 50 << 20 // 50 MB（通用上传）
	maxImageUploadSize = 20 << 20 // 20 MB（图片上传）
	maxVoiceUploadSize = 5 << 20  // 5 MB（语音上传）
)

// FileController 文件上传控制器
type FileController struct {
	fileService *service.FileService
}

// NewFileController 创建文件上传控制器
func NewFileController(fileService *service.FileService) *FileController {
	return &FileController{fileService: fileService}
}

// Upload 处理通用文件上传请求
// POST /api/v1/upload
// 支持 multipart/form-data，字段名为 "file"，最大 50MB
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
		utils.ResponseBadRequest(c, "文件大小不能超过 50MB")
		return
	}

	result, err := ctl.fileService.Upload(ctx, fileHeader)
	if err != nil {
		ctl.handleError(c, err, "文件上传失败")
		return
	}

	utils.ResponseOK(c, result)
}

// UploadImage 处理图片上传请求，自动生成缩略图
// POST /api/v1/upload/image
// 支持 multipart/form-data，字段名为 "file"，最大 20MB
// 返回原图 URL + 缩略图 URL + 宽高信息
func (ctl *FileController) UploadImage(c *gin.Context) {
	ctx := c.Request.Context()
	_, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		utils.ResponseBadRequest(c, "请选择要上传的图片")
		return
	}

	if fileHeader.Size > maxImageUploadSize {
		utils.ResponseBadRequest(c, "图片大小不能超过 20MB")
		return
	}

	result, err := ctl.fileService.UploadImage(ctx, fileHeader)
	if err != nil {
		ctl.handleError(c, err, "图片上传失败")
		return
	}

	utils.ResponseOK(c, result)
}

// UploadVoice 处理语音上传请求，校验格式和时长
// POST /api/v1/upload/voice
// 支持 multipart/form-data，字段名为 "file"，另需 "duration" 字段（秒）
// 最大 5MB，时长不超过 60 秒
func (ctl *FileController) UploadVoice(c *gin.Context) {
	ctx := c.Request.Context()
	_, ok := middleware.GetCurrentUserID(c)
	if !ok {
		utils.ResponseUnauthorized(c, "无法获取当前用户信息")
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		utils.ResponseBadRequest(c, "请选择要上传的语音文件")
		return
	}

	if fileHeader.Size > maxVoiceUploadSize {
		utils.ResponseBadRequest(c, "语音文件大小不能超过 5MB")
		return
	}

	durationStr := c.PostForm("duration")
	duration := 0
	if durationStr != "" {
		duration, err = strconv.Atoi(durationStr)
		if err != nil || duration < 0 {
			utils.ResponseBadRequest(c, "语音时长参数无效")
			return
		}
	}

	result, err := ctl.fileService.UploadVoice(ctx, fileHeader, duration)
	if err != nil {
		ctl.handleError(c, err, "语音上传失败")
		return
	}

	utils.ResponseOK(c, result)
}

// handleError 统一业务错误映射
func (ctl *FileController) handleError(c *gin.Context, err error, fallbackMsg ...string) {
	switch err {
	case service.ErrFileOpen:
		utils.ResponseBadRequest(c, err.Error())
	case service.ErrInvalidImage:
		utils.ResponseBadRequest(c, err.Error())
	case service.ErrInvalidVoice:
		utils.ResponseBadRequest(c, err.Error())
	case service.ErrVoiceTooLong:
		utils.ResponseBadRequest(c, err.Error())
	case service.ErrFileTooLarge:
		utils.ResponseBadRequest(c, err.Error())
	case service.ErrUploadFailed, service.ErrThumbnailGen:
		utils.ResponseError(c, err.Error())
	default:
		msg := "服务器内部错误"
		if len(fallbackMsg) > 0 && fallbackMsg[0] != "" {
			msg = fallbackMsg[0]
		}
		utils.ResponseError(c, msg)
	}
}
