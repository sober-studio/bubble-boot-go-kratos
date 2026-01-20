package biz

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/pkg/oss"
)

var (
	// ErrorUploadSceneNotFound 上传场景错误
	ErrorUploadSceneNotFound = kerrors.BadRequest("UPLOAD_SCENE_NOT_FOUND", "上传场景错误")
	// ErrorUploadFileSizeExceeded 文件大小超出限制
	ErrorUploadFileSizeExceeded = kerrors.BadRequest("UPLOAD_FILE_SIZE_EXCEEDED", "文件大小超出限制")
	// ErrorUploadFileFailed 文件上传失败
	ErrorUploadFileFailed = kerrors.InternalServer("UPLOAD_FILE_FAILED", "文件上传失败")
)

// UploadUseCase 文件上传用例
type UploadUseCase struct {
	oss    oss.Storage
	config *conf.App_Upload
	log    *log.Helper
}

// NewUploadUseCase 创建文件上传用例
func NewUploadUseCase(oss oss.Storage, c *conf.App, logger log.Logger) *UploadUseCase {
	return &UploadUseCase{
		oss:    oss,
		config: c.Upload,
		log:    log.NewHelper(logger),
	}
}

// UploadFileInput 文件上传输入
type UploadFileInput struct {
	Name        string
	ContentType string
	Size        int64
	Content     io.Reader // 文件流
	Scene       string    // 业务分类
}

// UploadFileResult 文件上传结果
type UploadFileResult struct {
	FileKey    string    // 文件存储路径
	FileURL    string    // 文件访问 URL
	FileSize   int64     // 文件大小
	IsPrivate  bool      // 是否私有
	UploadedAt time.Time // 上传时间
}

// UploadFile 上传文件
func (uc *UploadUseCase) UploadFile(ctx context.Context, input *UploadFileInput) (*UploadFileResult, error) {
	// 1. 获取场景配置
	sceneConfig, ok := uc.config.Scenes[input.Scene]
	if !ok {
		uc.log.Errorf("Upload scene not configured: %s", input.Scene)
		return nil, ErrorUploadSceneNotFound
	}

	// 2. 验证文件大小
	fileSize := input.Size
	if sceneConfig.MaxSize > 0 && fileSize > sceneConfig.MaxSize {
		return nil, ErrorUploadFileSizeExceeded
	}

	// 3. 验证文件类型
	// 注意：verifyFileType 会更新 input.ContentType 为检测到的真实类型
	if err := uc.verifyFileType(input, sceneConfig.AllowedTypes); err != nil {
		return nil, err
	}

	// 4. 生成文件存储路径
	fileKey := uc.generateFileKey(sceneConfig.PathPrefix, input.Name)

	// 5. 上传到对象存储
	_, err := uc.oss.Upload(ctx, fileKey, input.Content, input.Size, input.ContentType, sceneConfig.IsPrivate)
	if err != nil {
		uc.log.Errorf("Failed to upload file to OSS: %v", err)
		return nil, ErrorUploadFileFailed
	}

	// 6. 生成访问URL
	var expires time.Duration
	if uc.config.PrivateUrlExpires != nil {
		expires = uc.config.PrivateUrlExpires.AsDuration()
	} else {
		expires = time.Hour // 默认1小时
	}
	fileURL := uc.oss.GenerateURL(ctx, fileKey, sceneConfig.IsPrivate, expires)

	return &UploadFileResult{
		FileKey:    fileKey,
		FileURL:    fileURL,
		FileSize:   fileSize,
		IsPrivate:  sceneConfig.IsPrivate,
		UploadedAt: time.Now(),
	}, nil
}

// verifyFileType 验证文件类型
func (uc *UploadUseCase) verifyFileType(input *UploadFileInput, allowedTypes []string) error {
	if len(allowedTypes) == 0 {
		return nil
	}

	// 读取前512字节用于检测类型
	head := make([]byte, 512)
	n, err := input.Content.Read(head)
	if err != nil && err != io.EOF {
		return ErrorUploadFileFailed
	}

	// 截取实际读取的字节
	head = head[:n]
	fileType := http.DetectContentType(head)

	// 更新 input.ContentType 为真实检测到的类型，以便后续 OSS 上传使用正确的 MIME
	input.ContentType = fileType

	// 检查是否允许
	isAllowed := false
	for _, t := range allowedTypes {
		// 支持 MIME 类型匹配 (如 image/jpeg)
		if strings.EqualFold(t, fileType) {
			isAllowed = true
			break
		}
		// 支持通配符 (如 image/*)
		if strings.HasSuffix(t, "/*") {
			prefix := strings.TrimSuffix(t, "/*")
			if strings.HasPrefix(fileType, prefix) {
				isAllowed = true
				break
			}
		}
	}

	if !isAllowed {
		return kerrors.BadRequest("UPLOAD_FILE_TYPE_NOT_ALLOWED", fmt.Sprintf("文件类型不允许: %s", fileType))
	}

	// 恢复 Reader
	if seeker, ok := input.Content.(io.Seeker); ok {
		// 尝试 Seek 回到开头
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			// Seek 失败，使用 MultiReader 拼接
			input.Content = io.MultiReader(bytes.NewReader(head), input.Content)
		}
	} else {
		// 无法 Seek，使用 MultiReader 拼接
		input.Content = io.MultiReader(bytes.NewReader(head), input.Content)
	}

	return nil
}

// generateFileKey 生成文件存储路径
func (uc *UploadUseCase) generateFileKey(pathPrefix, filename string) string {
	// 生成唯一文件名：时间戳 + UUID + 原始扩展名
	ext := filepath.Ext(filename)
	uniqueName := fmt.Sprintf("%s_%s%s",
		time.Now().Format("20060102150405"),
		uuid.New().String()[:8],
		ext,
	)

	// 组合路径：prefix + 日期目录 + 文件名
	dateDir := time.Now().Format("2006/01/02")
	return strings.TrimSuffix(pathPrefix, "/") + "/" + dateDir + "/" + uniqueName
}
