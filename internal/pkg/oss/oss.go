package oss

import (
	"context"
	"io"
	"time"
)

type Storage interface {
	// Upload 上传数据
	// key: 文件存储路径
	// reader: 文件流
	// size: 文件大小（如果未知传 -1，但某些 SDK 可能要求必须提供）
	// isPrivate: 是否私有访问
	Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string, isPrivate bool) (string, error)
	// Delete 删除文件
	Delete(ctx context.Context, key string) error
	// GenerateURL 获取可访问的 URL（处理公开/私有逻辑）
	GenerateURL(ctx context.Context, key string, isPrivate bool, expires time.Duration) string
}
