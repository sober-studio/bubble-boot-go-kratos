package service

import (
	"context"

	pb "github.com/sober-studio/bubble-boot-go-kratos/api/passport/v1"
)

type PassportService struct {
	pb.UnimplementedPassportServer
}

func NewPassportService() *PassportService {
	return &PassportService{}
}

func (s *PassportService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterReply, error) {
	return &pb.RegisterReply{}, nil
}
func (s *PassportService) LoginByPassword(ctx context.Context, req *pb.LoginByPasswordRequest) (*pb.LoginReply, error) {
	return &pb.LoginReply{}, nil
}
func (s *PassportService) LoginByOtp(ctx context.Context, req *pb.LoginByOtpRequest) (*pb.LoginReply, error) {
	return &pb.LoginReply{}, nil
}
func (s *PassportService) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutReply, error) {
	return &pb.LogoutReply{}, nil
}
func (s *PassportService) UserInfo(ctx context.Context, req *pb.UserInfoRequest) (*pb.UserInfoReply, error) {
	return &pb.UserInfoReply{}, nil
}
func (s *PassportService) UpdatePassword(ctx context.Context, req *pb.UpdatePasswordRequest) (*pb.UpdatePasswordReply, error) {
	return &pb.UpdatePasswordReply{}, nil
}
func (s *PassportService) BindMobile(ctx context.Context, req *pb.BindMobileRequest) (*pb.BindMobileReply, error) {
	return &pb.BindMobileReply{}, nil
}
func (s *PassportService) UpdateMobile(ctx context.Context, req *pb.UpdateMobileRequest) (*pb.UpdateMobileReply, error) {
	return &pb.UpdateMobileReply{}, nil
}
func (s *PassportService) ResetPassword(ctx context.Context, req *pb.ResetPasswordRequest) (*pb.ResetPasswordReply, error) {
	return &pb.ResetPasswordReply{}, nil
}
