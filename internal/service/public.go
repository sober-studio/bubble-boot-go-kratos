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
	captcha  *biz.CaptchaUseCase
	otp      *biz.OtpUseCase
	passport *biz.PassportUseCase
	log      *log.Helper
}

func NewPublicService(captcha *biz.CaptchaUseCase, otp *biz.OtpUseCase, passport *biz.PassportUseCase, logger log.Logger) *PublicService {
	return &PublicService{captcha: captcha, otp: otp, passport: passport, log: log.NewHelper(logger)}
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

	scene := strings.ToLower(req.Scene.String())

	// 如果是重置密码场景，检查手机号是否已注册
	if scene == string(biz.Reset) {
		if err := s.passport.CheckPhoneRegistered(ctx, req.Mobile); err != nil {
			// 如果是为了安全，这里可以模糊错误，但需求要求直接报错
			// 为了用户体验，直接提示未注册
			return nil, err
		}
	}

	expireTime, err := s.otp.SendPhoneOtp(ctx, req.Mobile, scene)
	if err != nil {
		return nil, err
	}
	return &pb.SendSmsOtpReply{
		ExpireAt: expireTime,
	}, nil
}
