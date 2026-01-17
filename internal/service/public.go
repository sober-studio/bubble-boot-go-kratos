package service

import (
	"context"

	pb "github.com/sober-studio/bubble-boot-go-kratos/api/public/v1"
)

type PublicService struct {
	pb.UnimplementedPublicServer
}

func NewPublicService() *PublicService {
	return &PublicService{}
}

func (s *PublicService) GetCaptcha(ctx context.Context, req *pb.GetCaptchaRequest) (*pb.GetCaptchaReply, error) {
	return &pb.GetCaptchaReply{}, nil
}
func (s *PublicService) SendSmsCode(ctx context.Context, req *pb.SendSmsCodeRequest) (*pb.SendSmsCodeReply, error) {
	return &pb.SendSmsCodeReply{}, nil
}
