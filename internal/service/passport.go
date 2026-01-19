package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/errors"
	pb "github.com/sober-studio/bubble-boot-go-kratos/api/passport/v1"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/biz"
)

type PassportService struct {
	pb.UnimplementedPassportServer
	uc      *biz.PassportUseCase
	otp     *biz.OtpUseCase
	captcha *biz.CaptchaUseCase
}

func NewPassportService(uc *biz.PassportUseCase, otp *biz.OtpUseCase, captcha *biz.CaptchaUseCase) *PassportService {
	return &PassportService{
		uc:      uc,
		otp:     otp,
		captcha: captcha,
	}
}

func (s *PassportService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterReply, error) {
	if req.Password != req.ConfirmPassword {
		return nil, errors.BadRequest("PASSWORD_MISMATCH", "两次输入密码不一致")
	}

	token, err := s.uc.Register(ctx, req.Username, req.Password)
	if err != nil {
		return nil, err
	}
	return &pb.RegisterReply{Token: token}, nil
}

func (s *PassportService) LoginByPassword(ctx context.Context, req *pb.LoginByPasswordRequest) (*pb.LoginReply, error) {
	// 校验验证码
	if err := s.captcha.Verify(ctx, req.CaptchaId, req.Captcha); err != nil {
		return nil, err
	}

	token, err := s.uc.LoginByPassword(ctx, req.Username, req.Password)
	if err != nil {
		return nil, err
	}
	return &pb.LoginReply{Token: token}, nil
}

func (s *PassportService) LoginByOtp(ctx context.Context, req *pb.LoginByOtpRequest) (*pb.LoginReply, error) {
	// 校验短信验证码
	if valid, err := s.otp.VerifyPhoneOtp(ctx, req.Mobile, biz.Login, req.Code); err != nil || !valid {
		return nil, biz.ErrorOtpInvalid
	}

	token, err := s.uc.LoginByOtp(ctx, req.Mobile)
	if err != nil {
		return nil, err
	}
	return &pb.LoginReply{Token: token}, nil
}

func (s *PassportService) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutReply, error) {
	if err := s.uc.Logout(ctx); err != nil {
		return nil, err
	}
	return &pb.LogoutReply{}, nil
}

func (s *PassportService) UserInfo(ctx context.Context, req *pb.UserInfoRequest) (*pb.UserInfoReply, error) {
	u, err := s.uc.UserInfo(ctx)
	if err != nil {
		return nil, err
	}
	status := int32(0)
	if u.IsAvailable {
		status = 1
	}
	return &pb.UserInfoReply{
		Username: u.Username,
		Mobile:   u.Phone,
		Status:   status,
	}, nil
}

func (s *PassportService) UpdatePassword(ctx context.Context, req *pb.UpdatePasswordRequest) (*pb.UpdatePasswordReply, error) {
	if req.NewPassword != req.ConfirmPassword {
		return nil, errors.BadRequest("PASSWORD_MISMATCH", "两次输入密码不一致")
	}

	err := s.uc.UpdatePassword(ctx, req.OldPassword, req.NewPassword)
	if err != nil {
		return nil, err
	}
	return &pb.UpdatePasswordReply{}, nil
}

func (s *PassportService) BindMobile(ctx context.Context, req *pb.BindMobileRequest) (*pb.BindMobileReply, error) {
	// 校验短信验证码
	if valid, err := s.otp.VerifyPhoneOtp(ctx, req.Mobile, biz.Bind, req.Code); err != nil || !valid {
		return nil, biz.ErrorOtpInvalid
	}

	err := s.uc.BindMobile(ctx, req.Mobile)
	if err != nil {
		return nil, err
	}
	return &pb.BindMobileReply{}, nil
}

func (s *PassportService) UpdateMobile(ctx context.Context, req *pb.UpdateMobileRequest) (*pb.UpdateMobileReply, error) {
	// 校验短信验证码
	if valid, err := s.otp.VerifyPhoneOtp(ctx, req.Mobile, biz.Bind, req.Code); err != nil || !valid {
		return nil, biz.ErrorOtpInvalid
	}

	err := s.uc.UpdateMobile(ctx, req.Mobile)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateMobileReply{}, nil
}

func (s *PassportService) ResetPassword(ctx context.Context, req *pb.ResetPasswordRequest) (*pb.ResetPasswordReply, error) {
	if req.NewPassword != req.ConfirmPassword {
		return nil, errors.BadRequest("PASSWORD_MISMATCH", "两次输入密码不一致")
	}

	// 校验短信验证码
	if valid, err := s.otp.VerifyPhoneOtp(ctx, req.Mobile, biz.Reset, req.SmsCode); err != nil || !valid {
		return nil, biz.ErrorOtpInvalid
	}

	err := s.uc.ResetPassword(ctx, req.Mobile, req.NewPassword)
	if err != nil {
		return nil, err
	}
	return &pb.ResetPasswordReply{}, nil
}
