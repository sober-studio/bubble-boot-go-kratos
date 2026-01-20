package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/transport/http"
	pb "github.com/sober-studio/bubble-boot-go-kratos/api/upload/v1"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/biz"
)

type UploadService struct {
	pb.UnimplementedUploadServer
	uc *biz.UploadUseCase
}

func NewUploadService(uc *biz.UploadUseCase) *UploadService {
	return &UploadService{uc: uc}
}

func (s *UploadService) Upload(ctx context.Context, req *pb.UploadFileRequest) (*pb.UploadFileReply, error) {
	// 1. 获取 HTTP Request
	ht, ok := http.RequestFromServerContext(ctx)
	if !ok {
		return nil, errors.BadRequest("TRANSPORT_ERROR", "only support http")
	}

	// 2. 从表单中提取文件流
	// "file" 需与前端 FormData 中的 key 保持一致
	file, header, err := ht.FormFile("file")
	if err != nil {
		return nil, errors.BadRequest("FILE_MISSING", "file is required")
	}
	defer file.Close()

	// 3. 构造 biz 层所需的对象
	// 注意：req.Category 和 req.Remark 已经被之前的 CustomRequestDecoder 自动填充了
	fileInfo := &biz.UploadFileInput{
		Name:        header.Filename,
		ContentType: header.Header.Get("Content-Type"),
		Size:        header.Size,
		Content:     file,               // 直接传递句柄，实现流式上传
		Scene:       req.Scene.String(), // 来自 Proto 定义的字段
	}

	// 4. 调用 biz 层逻辑执行 OSS 上传
	url, err := s.uc.UploadFile(ctx, fileInfo)
	if err != nil {
		return nil, err
	}

	// 5. 返回结果
	return &pb.UploadFileReply{
		FileKey:    url.FileKey,
		FileUrl:    url.FileURL,
		FileSize:   url.FileSize,
		IsPrivate:  url.IsPrivate,
		UploadedAt: url.UploadedAt.Unix(),
	}, nil
}
