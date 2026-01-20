package oss

import (
	"context"
	"time"
)

type Storage interface {
	// Upload 上传字节数据
	Upload(ctx context.Context, key string, data []byte, isPrivate bool) (string, error)
	// Delete 删除文件
	Delete(ctx context.Context, key string) error
	// GenerateURL 获取可访问的 URL（处理公开/私有逻辑）
	GenerateURL(ctx context.Context, key string, isPrivate bool, expires time.Duration) string
}
