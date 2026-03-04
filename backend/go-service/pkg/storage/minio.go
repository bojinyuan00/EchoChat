// Package storage 提供对象存储基础设施
// 封装 MinIO 客户端初始化和存储桶自动创建逻辑
package storage

import (
	"context"
	"fmt"

	"github.com/echochat/backend/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// NewMinioClient 初始化 MinIO 客户端并确保存储桶存在
// 启动时自动创建配置中指定的 bucket（如不存在）
func NewMinioClient(cfg *config.MinioConfig) (*minio.Client, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("创建 MinIO 客户端失败: %w", err)
	}

	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("检查存储桶 %s 失败: %w", cfg.Bucket, err)
	}

	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("创建存储桶 %s 失败: %w", cfg.Bucket, err)
		}
	}

	return client, nil
}
