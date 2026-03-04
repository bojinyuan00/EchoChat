// Package service 提供文件上传模块的业务逻辑
// 封装 MinIO 对象存储的上传操作，返回可访问的文件 URL
package service

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/echochat/backend/config"
	"github.com/echochat/backend/pkg/logs"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
)

var (
	ErrFileOpen     = errors.New("打开上传文件失败")
	ErrUploadFailed = errors.New("文件上传失败")
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

// UploadResult 文件上传结果
type UploadResult struct {
	URL      string `json:"url"`
	FileName string `json:"file_name"`
	Size     int64  `json:"size"`
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

	scheme := "http"
	if s.minioCfg.UseSSL {
		scheme = "https"
	}
	url := fmt.Sprintf("%s://%s/%s/%s", scheme, s.minioCfg.Endpoint, s.minioCfg.Bucket, objectName)

	return &UploadResult{
		URL:      url,
		FileName: fileHeader.Filename,
		Size:     fileHeader.Size,
	}, nil
}
