package oss

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/conf"
)

type localStorage struct {
	baseDir string
	conf    *conf.Data_Oss
	log     *log.Helper
}

func NewLocalStorage(c *conf.Data_Oss, logger log.Logger) Storage {
	// 默认存储在当前运行目录的 uploads 下
	baseDir := "uploads"
	if c.Bucket != "" {
		baseDir = c.Bucket
	}

	// 确保存储目录存在
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		panic(fmt.Sprintf("Failed to create local storage directory: %v", err))
	}

	return &localStorage{
		baseDir: baseDir,
		conf:    c,
		log:     log.NewHelper(logger),
	}
}

func (s *localStorage) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string, isPrivate bool) (string, error) {
	// 拼接完整文件路径
	filePath := filepath.Join(s.baseDir, key)

	// 安全检查：防止路径遍历 (Path Traversal)
	// 确保生成的路径仍在 baseDir 下
	// Clean 路径以处理 .. 等相对路径符号
	cleanPath := filepath.Clean(filePath)
	cleanBase := filepath.Clean(s.baseDir)
	// 注意：这里需要确保 cleanBase 后面带上 separator，否则 /base/dir2 可能会匹配 /base/dir 前缀
	if !filepath.IsAbs(cleanBase) {
		absBase, err := filepath.Abs(cleanBase)
		if err == nil {
			cleanBase = absBase
		}
	}
	if !filepath.IsAbs(cleanPath) {
		absPath, err := filepath.Abs(cleanPath)
		if err == nil {
			cleanPath = absPath
		}
	}
	// 简单的包含检查
	// 注意：filepath.Rel 可能更严谨，但要处理不同平台的差异
	rel, err := filepath.Rel(cleanBase, cleanPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("invalid file path: %s", key)
	}

	// 确保文件所在目录存在
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	// 创建目标文件
	out, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	// 流式写入文件
	if _, err := io.Copy(out, reader); err != nil {
		return "", err
	}

	return key, nil
}

func (s *localStorage) Delete(ctx context.Context, key string) error {
	filePath := filepath.Join(s.baseDir, key)
	return os.Remove(filePath)
}

func (s *localStorage) GenerateURL(ctx context.Context, key string, isPrivate bool, expires time.Duration) string {
	schema := "http"
	if s.conf.UseHttps {
		schema = "https"
	}

	// 如果配置了 Domain，则使用 Domain + Key
	// 例如：http://localhost:8000/uploads/avatar/123.jpg
	if s.conf.Domain != "" {
		return fmt.Sprintf("%s://%s/%s", schema, s.conf.Domain, key)
	}

	// 如果没有配置 Domain，则只能返回相对路径或者提示错误
	// 这里返回相对路径，前端可能需要自己拼接
	return fmt.Sprintf("/%s", key)
}
