// Package service 提供文件上传模块的业务逻辑
// 封装 MinIO 对象存储的上传操作，返回可访问的文件 URL
package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/echochat/backend/config"
	"github.com/echochat/backend/pkg/logs"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
)

var (
	ErrFileOpen       = errors.New("打开上传文件失败")
	ErrUploadFailed   = errors.New("文件上传失败")
	ErrFileTooLarge   = errors.New("文件大小超出限制")
	ErrInvalidImage   = errors.New("无效的图片文件")
	ErrInvalidVoice   = errors.New("不支持的语音格式，仅支持 mp3/wav/aac/m4a/webm/ogg")
	ErrVoiceTooLong   = errors.New("语音时长不能超过 60 秒")
	ErrThumbnailGen   = errors.New("缩略图生成失败")
)

// FileService 文件上传服务
type FileService struct {
	minioClient *minio.Client
	minioCfg    *config.MinioConfig
}

// NewFileService 创建文件上传服务实例
func NewFileService(client *minio.Client, cfg *config.MinioConfig) *FileService {
	return &FileService{
		minioClient: client,
		minioCfg:    cfg,
	}
}

// UploadResult 通用文件上传结果
type UploadResult struct {
	URL      string `json:"url"`
	FileName string `json:"file_name"`
	Size     int64  `json:"size"`
}

// ImageUploadResult 图片上传结果（含缩略图和尺寸）
type ImageUploadResult struct {
	URL          string `json:"url"`
	ThumbnailURL string `json:"thumbnail_url"`
	FileName     string `json:"file_name"`
	Size         int64  `json:"size"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	MimeType     string `json:"mime_type"`
}

// VoiceUploadResult 语音上传结果（含时长）
type VoiceUploadResult struct {
	URL      string `json:"url"`
	FileName string `json:"file_name"`
	Size     int64  `json:"size"`
	Duration int    `json:"duration"`
	MimeType string `json:"mime_type"`
}

// allowedVoiceExts 允许的语音文件扩展名
// .webm / .ogg 用于兼容 H5 端浏览器 MediaRecorder 输出（uni-app H5 端不支持 getRecorderManager）
var allowedVoiceExts = map[string]bool{
	".mp3":  true,
	".wav":  true,
	".aac":  true,
	".m4a":  true,
	".webm": true,
	".ogg":  true,
}

// Upload 上传文件到 MinIO，返回文件访问 URL
// 文件按日期目录组织：uploads/2026/03/04/{uuid}.{ext}
func (s *FileService) Upload(ctx context.Context, fileHeader *multipart.FileHeader) (*UploadResult, error) {
	funcName := "service.file_service.Upload"
	logs.Info(ctx, funcName, "上传文件",
		zap.String("file_name", fileHeader.Filename),
		zap.Int64("size", fileHeader.Size))

	file, err := fileHeader.Open()
	if err != nil {
		logs.Error(ctx, funcName, "打开上传文件失败", zap.Error(err))
		return nil, ErrFileOpen
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	now := time.Now()
	objectName := fmt.Sprintf("uploads/%s/%s%s", now.Format("2006/01/02"), uuid.New().String(), ext)

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err = s.minioClient.PutObject(ctx, s.minioCfg.Bucket, objectName, file, fileHeader.Size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		logs.Error(ctx, funcName, "上传文件到 MinIO 失败", zap.Error(err))
		return nil, ErrUploadFailed
	}

	url := s.buildURL(objectName)

	return &UploadResult{
		URL:      url,
		FileName: fileHeader.Filename,
		Size:     fileHeader.Size,
	}, nil
}

// UploadImage 上传图片并生成缩略图
// 原图和缩略图均存储到 MinIO，缩略图宽度 200px 等比缩放，JPEG quality 80
func (s *FileService) UploadImage(ctx context.Context, fileHeader *multipart.FileHeader) (*ImageUploadResult, error) {
	funcName := "service.file_service.UploadImage"
	logs.Info(ctx, funcName, "上传图片",
		zap.String("file_name", fileHeader.Filename),
		zap.Int64("size", fileHeader.Size))

	file, err := fileHeader.Open()
	if err != nil {
		logs.Error(ctx, funcName, "打开图片文件失败", zap.Error(err))
		return nil, ErrFileOpen
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		logs.Error(ctx, funcName, "解码图片失败", zap.Error(err))
		return nil, ErrInvalidImage
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	now := time.Now()
	fileID := uuid.New().String()
	dateDir := now.Format("2006/01/02")

	// 上传原图（回到文件开头）
	if _, err = file.Seek(0, 0); err != nil {
		logs.Error(ctx, funcName, "重置文件指针失败", zap.Error(err))
		return nil, ErrFileOpen
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	objectName := fmt.Sprintf("uploads/%s/%s%s", dateDir, fileID, ext)

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}

	_, err = s.minioClient.PutObject(ctx, s.minioCfg.Bucket, objectName, file, fileHeader.Size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		logs.Error(ctx, funcName, "上传原图到 MinIO 失败", zap.Error(err))
		return nil, ErrUploadFailed
	}

	// 生成缩略图：宽度 200px，等比缩放
	thumb := imaging.Resize(img, 200, 0, imaging.Lanczos)

	var thumbBuf bytes.Buffer
	if err = imaging.Encode(&thumbBuf, thumb, imaging.JPEG, imaging.JPEGQuality(80)); err != nil {
		logs.Error(ctx, funcName, "编码缩略图失败", zap.Error(err))
		return nil, ErrThumbnailGen
	}

	thumbObjectName := fmt.Sprintf("uploads/%s/%s_thumb.jpg", dateDir, fileID)
	_, err = s.minioClient.PutObject(ctx, s.minioCfg.Bucket, thumbObjectName, &thumbBuf, int64(thumbBuf.Len()), minio.PutObjectOptions{
		ContentType: "image/jpeg",
	})
	if err != nil {
		logs.Error(ctx, funcName, "上传缩略图到 MinIO 失败", zap.Error(err))
		return nil, ErrThumbnailGen
	}

	return &ImageUploadResult{
		URL:          s.buildURL(objectName),
		ThumbnailURL: s.buildURL(thumbObjectName),
		FileName:     fileHeader.Filename,
		Size:         fileHeader.Size,
		Width:        width,
		Height:       height,
		MimeType:     contentType,
	}, nil
}

// UploadVoice 上传语音文件，校验格式和前端传入的时长
// duration 由前端录音回调提供（秒），后端仅做范围校验
func (s *FileService) UploadVoice(ctx context.Context, fileHeader *multipart.FileHeader, duration int) (*VoiceUploadResult, error) {
	funcName := "service.file_service.UploadVoice"
	logs.Info(ctx, funcName, "上传语音",
		zap.String("file_name", fileHeader.Filename),
		zap.Int64("size", fileHeader.Size),
		zap.Int("duration", duration))

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if !allowedVoiceExts[ext] {
		return nil, ErrInvalidVoice
	}

	if duration > 60 {
		return nil, ErrVoiceTooLong
	}

	file, err := fileHeader.Open()
	if err != nil {
		logs.Error(ctx, funcName, "打开语音文件失败", zap.Error(err))
		return nil, ErrFileOpen
	}
	defer file.Close()

	now := time.Now()
	objectName := fmt.Sprintf("uploads/%s/%s%s", now.Format("2006/01/02"), uuid.New().String(), ext)

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "audio/mpeg"
	}

	_, err = s.minioClient.PutObject(ctx, s.minioCfg.Bucket, objectName, file, fileHeader.Size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		logs.Error(ctx, funcName, "上传语音到 MinIO 失败", zap.Error(err))
		return nil, ErrUploadFailed
	}

	return &VoiceUploadResult{
		URL:      s.buildURL(objectName),
		FileName: fileHeader.Filename,
		Size:     fileHeader.Size,
		Duration: duration,
		MimeType: contentType,
	}, nil
}

// buildURL 根据 MinIO 配置构建文件访问 URL
func (s *FileService) buildURL(objectName string) string {
	scheme := "http"
	if s.minioCfg.UseSSL {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/%s/%s", scheme, s.minioCfg.Endpoint, s.minioCfg.Bucket, objectName)
}
