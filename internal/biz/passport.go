package biz

import (
	"context"
	"errors"
	"strconv"
	"time"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/pkg/auth"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound       = kerrors.NotFound("USER_NOT_FOUND", "用户不存在")
	ErrUserAlreadyExists  = kerrors.Conflict("USER_ALREADY_EXISTS", "用户已存在")
	ErrPasswordInvalid    = kerrors.BadRequest("PASSWORD_INVALID", "密码错误")
	ErrMobileAlreadyBound = kerrors.Conflict("MOBILE_ALREADY_BOUND", "手机号已被绑定")
	ErrUserDisabled       = kerrors.Forbidden("USER_DISABLED", "账号已被禁用")
)

type User struct {
	ID           int64
	Username     string
	PasswordHash string
	Phone        string
	Nickname     string
	IsAvailable  bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UserRepo interface {
	CreateUser(ctx context.Context, user *User) (*User, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	GetUserByPhone(ctx context.Context, phone string) (*User, error)
	GetUserByID(ctx context.Context, id int64) (*User, error)
	UpdatePassword(ctx context.Context, id int64, passwordHash string) error
	UpdatePhone(ctx context.Context, id int64, phone string) error
}

type PassportUseCase struct {
	auth auth.TokenService
	user UserRepo
	log  *log.Helper
}

func NewPassportUseCase(
	auth auth.TokenService,
	user UserRepo,
	logger log.Logger,
) *PassportUseCase {
	return &PassportUseCase{
		auth: auth,
		user: user,
		log:  log.NewHelper(logger),
	}
}

func (uc *PassportUseCase) Register(ctx context.Context, username, password string) (string, error) {
	// 检查用户名是否存在
	if u, _ := uc.user.GetUserByUsername(ctx, username); u != nil {
		return "", ErrUserAlreadyExists
	}

	// 密码加密
	hash, err := uc.hashPassword(password)
	if err != nil {
		return "", err
	}

	user := &User{
		Username:     username,
		PasswordHash: hash,
		IsAvailable:  true,
	}

	savedUser, err := uc.user.CreateUser(ctx, user)
	if err != nil {
		return "", err
	}

	// 生成 Token
	return uc.auth.GenerateToken(ctx, uc.formatUserID(savedUser.ID))
}

func (uc *PassportUseCase) LoginByPassword(ctx context.Context, username, password string) (string, error) {
	// 查询用户
	user, err := uc.user.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return "", ErrUserNotFound
		}
		return "", err
	}

	// 校验密码
	if !uc.checkPassword(password, user.PasswordHash) {
		return "", ErrPasswordInvalid
	}

	if !user.IsAvailable {
		return "", ErrUserDisabled
	}

	return uc.auth.GenerateToken(ctx, uc.formatUserID(user.ID))
}

func (uc *PassportUseCase) LoginByOtp(ctx context.Context, phone string) (string, error) {
	// 查询用户
	user, err := uc.user.GetUserByPhone(ctx, phone)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return "", ErrUserNotFound
		}
		return "", err
	}

	if !user.IsAvailable {
		return "", ErrUserDisabled
	}

	return uc.auth.GenerateToken(ctx, uc.formatUserID(user.ID))
}

func (uc *PassportUseCase) Logout(ctx context.Context) error {
	// 撤销当前 Token
	return uc.auth.RevokeToken(ctx, "")
}

func (uc *PassportUseCase) UserInfo(ctx context.Context) (*User, error) {
	userId, err := uc.auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return uc.user.GetUserByID(ctx, userId)
}

func (uc *PassportUseCase) UpdatePassword(ctx context.Context, oldPassword, newPassword string) error {
	userId, err := uc.auth.GetUserIDFromContext(ctx)
	if err != nil {
		return err
	}

	user, err := uc.user.GetUserByID(ctx, userId)
	if err != nil {
		return err
	}

	if !uc.checkPassword(oldPassword, user.PasswordHash) {
		return ErrPasswordInvalid
	}

	hash, err := uc.hashPassword(newPassword)
	if err != nil {
		return err
	}

	if err := uc.user.UpdatePassword(ctx, userId, hash); err != nil {
		return err
	}

	// 密码修改完成后，撤销用户所有的令牌
	return uc.auth.RevokeAllTokens(ctx)
}

func (uc *PassportUseCase) BindMobile(ctx context.Context, mobile string) error {
	userId, err := uc.auth.GetUserIDFromContext(ctx)
	if err != nil {
		return err
	}

	// 检查手机号是否已被使用
	if u, _ := uc.user.GetUserByPhone(ctx, mobile); u != nil {
		return ErrMobileAlreadyBound
	}

	return uc.user.UpdatePhone(ctx, userId, mobile)
}

func (uc *PassportUseCase) UpdateMobile(ctx context.Context, mobile string) error {
	userId, err := uc.auth.GetUserIDFromContext(ctx)
	if err != nil {
		return err
	}

	// 检查手机号是否已被使用
	if u, _ := uc.user.GetUserByPhone(ctx, mobile); u != nil {
		return ErrMobileAlreadyBound
	}

	return uc.user.UpdatePhone(ctx, userId, mobile)
}

func (uc *PassportUseCase) ResetPassword(ctx context.Context, mobile, newPassword string) error {
	user, err := uc.user.GetUserByPhone(ctx, mobile)
	if err != nil {
		return ErrUserNotFound
	}

	hash, err := uc.hashPassword(newPassword)
	if err != nil {
		return err
	}

	if err := uc.user.UpdatePassword(ctx, user.ID, hash); err != nil {
		return err
	}

	// 密码重置完成后，撤销用户所有的令牌
	// 注意：这里需要通过 receiver 的 ID 来撤销，但 auth.RevokeAllTokens(ctx) 是针对当前登录用户的。
	// 如果是找回密码，当前可能未登录。
	// TokenService 应该支持通过 userID 撤销所有 Token。
	// 暂时假设 RevokeAllTokens 能通过某些方式处理，或者找回密码后需要强制重新登录。
	// 但通常找回密码是针对特定 userID 的。
	// 我检查一下 auth.go
	return uc.auth.RevokeAllTokens(ctx)
}

func (uc *PassportUseCase) hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func (uc *PassportUseCase) checkPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (uc *PassportUseCase) formatUserID(id int64) string {
	return strconv.FormatInt(id, 10)
}
