package service

import (
	"context"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	pb "github.com/sober-studio/bubble-boot-go-kratos/api/public/v1"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/biz"
)

type PublicService struct {
	pb.UnimplementedPublicServer
	captcha *biz.CaptchaUseCase
	otp     *biz.OtpUseCase
	log     *log.Helper
}

func NewPublicService(captcha *biz.CaptchaUseCase, otp *biz.OtpUseCase, logger log.Logger) *PublicService {
	return &PublicService{captcha: captcha, otp: otp, log: log.NewHelper(logger)}
}

func (s *PublicService) GetCaptcha(ctx context.Context, req *pb.GetCaptchaRequest) (*pb.GetCaptchaReply, error) {
	id, b64, err := s.captcha.Generate(ctx)
	if err != nil {
		return nil, err
	}
	return &pb.GetCaptchaReply{
		CaptchaId: id,
		ImageB64:  b64,
	}, nil
}
func (s *PublicService) SendSmsOtp(ctx context.Context, req *pb.SendSmsOtpRequest) (*pb.SendSmsOtpReply, error) {
	if err := s.captcha.Verify(ctx, req.CaptchaId, req.Captcha); err != nil {
		return nil, err
	}
	expireTime, err := s.otp.SendPhoneOtp(ctx, req.Mobile, strings.ToLower(req.Scene.String()))
	if err != nil {
		return nil, err
	}
	return &pb.SendSmsOtpReply{
		ExpireAt: expireTime,
	}, nil
}
