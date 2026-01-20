package oss

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/conf"
)

type minioStorage struct {
	client *minio.Client
	conf   *conf.Data_Oss
	log    *log.Helper
}

func NewMinioStorage(c *conf.Data_Oss, logger log.Logger) Storage {
	// MinIO 初始化
	// Endpoint 不包含 http/https 前缀
	client, err := minio.New(c.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(c.AccessKeyId, c.AccessKeySecret, ""),
		Secure: c.UseHttps,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize MinIO client: %v", err))
	}

	return &minioStorage{
		client: client,
		conf:   c,
		log:    log.NewHelper(logger),
	}
}

func (s *minioStorage) Upload(ctx context.Context, key string, data []byte, isPrivate bool) (string, error) {
	reader := bytes.NewReader(data)
	_, err := s.client.PutObject(ctx, s.conf.Bucket, key, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: "application/octet-stream",
		// 如果需要可以在这里设置 UserMetadata 或其他选项
	})
	if err != nil {
		return "", err
	}

	return key, nil
}

func (s *minioStorage) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.conf.Bucket, key, minio.RemoveObjectOptions{})
}

func (s *minioStorage) GenerateURL(ctx context.Context, key string, isPrivate bool, expires time.Duration) string {
	if !isPrivate {
		schema := "http"
		if s.conf.UseHttps {
			schema = "https"
		}
		// MinIO 公开访问通常需要 Bucket Policy 设置为 public
		// 直接返回访问 URL
		return fmt.Sprintf("%s://%s/%s/%s", schema, s.conf.Domain, s.conf.Bucket, key)
	}

	// 生成预签名 URL
	reqParams := make(url.Values)
	presignedURL, err := s.client.PresignedGetObject(ctx, s.conf.Bucket, key, expires, reqParams)
	if err != nil {
		s.log.Errorf("Failed to generate presigned URL: %v", err)
		return ""
	}
	return presignedURL.String()
}
