package biz

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/conf"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/pkg/debug"
)

var (
	ErrorOtpSendError       = errors.InternalServer("OTP_SEND_ERROR", "发送验证码错误")
	ErrorOtpSendTooFrequent = errors.BadRequest("OTP_SEND_TOO_FAST", "发送过于频繁，请稍后再试")
	ErrorSceneNotFound      = errors.BadRequest("SCENE_NOT_FOUND", "验证码场景错误")
	ErrorOtpExpired         = errors.BadRequest("OTP_EXPIRED", "验证码已过期或未发送")
	ErrorOtpInvalid         = errors.BadRequest("OTP_INVALID", "验证码错误")
)

type SmsSender interface {
	Send(ctx context.Context, phone, templateName string, params map[string]string) error
}

type EmailSender interface {
	Send(ctx context.Context, email, subjectName string, params map[string]string) error
}

type OtpCache interface {
	Set(ctx context.Context, key string, value string, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Del(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}

type OtpUseCase struct {
	sms   SmsSender
	email EmailSender
	cache OtpCache
	conf  *conf.App_Otp
	log   *log.Helper
}

func NewOtpUseCase(s SmsSender, e EmailSender, c OtpCache, conf *conf.App, logger log.Logger) *OtpUseCase {
	return &OtpUseCase{sms: s, email: e, cache: c, conf: conf.Otp, log: log.NewHelper(logger)}
}

// SendPhoneOtp 发送手机验证码
func (uc *OtpUseCase) SendPhoneOtp(ctx context.Context, phone, scene string) error {
	cfg, ok := uc.conf.PhoneScenes[scene]
	if !ok {
		return ErrorSceneNotFound
	}

	return uc.process(ctx, "phone", scene, phone, cfg, func(code string) error {
		return uc.sms.Send(ctx, phone, cfg.TemplateName, map[string]string{"code": code})
	})
}

// SendEmailOtp 发送邮箱验证码
func (uc *OtpUseCase) SendEmailOtp(ctx context.Context, email, scene string) error {
	cfg, ok := uc.conf.EmailScenes[scene]
	if !ok {
		return ErrorSceneNotFound
	}

	return uc.process(ctx, "email", scene, email, cfg, func(code string) error {
		return uc.email.Send(ctx, email, cfg.TemplateName, map[string]string{"code": code})
	})
}

// 内部抽象流程
func (uc *OtpUseCase) process(ctx context.Context, kind, scene, receiver string, cfg *conf.App_Otp_Scene, sendFn func(code string) error) error {
	intervalKey := fmt.Sprintf("otp:interval:%s:%s:%s", kind, scene, receiver)
	codeKey := fmt.Sprintf("opt:code:%s:%s:%s", kind, scene, receiver)
	// 1. 限频校验
	if exists, _ := uc.cache.Exists(ctx, intervalKey); exists {
		return ErrorOtpSendTooFrequent
	}

	// 2. 生成
	code := uc.generateCode(cfg.CodeLength)

	// 3. 发送
	if err := sendFn(code); err != nil {
		uc.log.Errorf("发送%s验证码失败: %v", kind, err)
		return ErrorOtpSendError
	}

	// 4. 存储
	expiration := cfg.ExpiresIn.AsDuration()
	if err := uc.cache.Set(ctx, codeKey, code, expiration); err != nil {
		uc.log.Errorf("缓存短信验证码失败: %v", err)
		return ErrorOtpSendError
	}
	if err := uc.cache.Set(ctx, intervalKey, "1", cfg.ResendInterval.AsDuration()); err != nil {
		uc.log.Errorf("缓存短信发送频率失败: %v", err)
		return ErrorOtpSendError
	}

	// 5. DEBUG
	if debug.IsDebug() {
		if info, ok := debug.FromContext(ctx); ok {
			info["otp"] = code
		}
	}

	return nil
}

// VerifyPhoneOtp 校验手机验证码
func (uc *OtpUseCase) VerifyPhoneOtp(ctx context.Context, phone, scene, code string) (bool, error) {
	return uc.verify(ctx, "phone", scene, phone, code)
}

// VerifyEmailOtp 校验邮箱验证码
func (uc *OtpUseCase) VerifyEmailOtp(ctx context.Context, email, scene, code string) (bool, error) {
	return uc.verify(ctx, "email", scene, email, code)
}

// 内部通用校验逻辑
func (uc *OtpUseCase) verify(ctx context.Context, kind, scene, receiver, inputCode string) (bool, error) {
	codeKey := fmt.Sprintf("opt:code:%s:%s:%s", kind, scene, receiver)
	stored, err := uc.cache.Get(ctx, codeKey)
	if err != nil || stored != inputCode {
		return false, ErrorOtpInvalid
	}
	return true, uc.cache.Del(ctx, codeKey)
}

// 生成验证码
func (uc *OtpUseCase) generateCode(l int32) string {
	if l <= 0 {
		l = 6
	}
	if l > 10 {
		l = 10
	}

	const digits = "0123456789"

	b := make([]byte, int(l))
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			b[i] = digits[0]
			continue
		}
		b[i] = digits[n.Int64()]
	}
	return string(b)
}
