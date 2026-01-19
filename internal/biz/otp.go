package biz

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/conf"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/pkg/debug"
)

const (
	otpIntervalKeyPattern = "otp:interval:%s:%s:%s"
	otpCodeKeyPattern     = "otp:code:%s:%s:%s"
	otpFailKeyPattern     = "otp:fail:%s:%s:%s"
	otpMaxFailCount       = 5
	otpFailExpiration     = time.Hour
)

const (
	kindPhone = "phone"
	kindEmail = "email"
)

var (
	ErrorOtpSendError       = kerrors.InternalServer("OTP_SEND_ERROR", "发送验证码错误")
	ErrorOtpSendTooFrequent = kerrors.BadRequest("OTP_SEND_TOO_FAST", "发送过于频繁，请稍后再试")
	ErrorSceneNotFound      = kerrors.BadRequest("SCENE_NOT_FOUND", "验证码场景错误")
	ErrorOtpExpired         = kerrors.BadRequest("OTP_EXPIRED", "验证码已过期或未发送")
	ErrorOtpInvalid         = kerrors.BadRequest("OTP_INVALID", "验证码错误")
	ErrOtpCacheMiss         = kerrors.NotFound("OTP_CACHE_MISS", "验证码不存在或已过期")
)

type SmsSender interface {
	Send(ctx context.Context, phone, templateName string, params map[string]string) error
}

type EmailSender interface {
	Send(ctx context.Context, email, templateName string, params map[string]string) error
}

type OtpCache interface {
	Set(ctx context.Context, key string, value string, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Del(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	SetNX(ctx context.Context, key string, value string, expiration time.Duration) (bool, error)
	Incr(ctx context.Context, key string, expiration time.Duration) (int64, error)
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
func (uc *OtpUseCase) SendPhoneOtp(ctx context.Context, phone, scene string) (int64, error) {
	cfg, ok := uc.conf.PhoneScenes[scene]
	if !ok {
		return 0, ErrorSceneNotFound
	}

	kind := kindPhone
	// 发新验证码前清理上一轮的失败计数
	failKey := fmt.Sprintf(otpFailKeyPattern, kind, scene, phone)
	_ = uc.cache.Del(ctx, failKey)

	return uc.process(ctx, kindPhone, scene, phone, cfg, func(code string) error {
		return uc.sms.Send(ctx, phone, cfg.TemplateName, map[string]string{"code": code})
	})
}

// SendEmailOtp 发送邮箱验证码
func (uc *OtpUseCase) SendEmailOtp(ctx context.Context, email, scene string) (int64, error) {
	cfg, ok := uc.conf.EmailScenes[scene]
	if !ok {
		return 0, ErrorSceneNotFound
	}

	kind := kindEmail
	// 发新验证码前清理上一轮的失败计数
	failKey := fmt.Sprintf(otpFailKeyPattern, kind, scene, email)
	_ = uc.cache.Del(ctx, failKey)

	return uc.process(ctx, kindEmail, scene, email, cfg, func(code string) error {
		return uc.email.Send(ctx, email, cfg.TemplateName, map[string]string{"code": code})
	})
}

// 内部抽象流程
func (uc *OtpUseCase) process(ctx context.Context, kind, scene, receiver string, cfg *conf.App_Otp_Scene, sendFn func(code string) error) (int64, error) {
	intervalKey := fmt.Sprintf(otpIntervalKeyPattern, kind, scene, receiver)
	codeKey := fmt.Sprintf(otpCodeKeyPattern, kind, scene, receiver)

	resendInterval := cfg.ResendInterval.AsDuration()

	acquired, err := uc.cache.SetNX(ctx, intervalKey, "1", resendInterval)
	if err != nil {
		uc.log.Errorf("设置发送频率标记失败: %v", err)
		return 0, ErrorOtpSendError
	}
	if !acquired {
		return 0, ErrorOtpSendTooFrequent
	}

	code := uc.generateCode(cfg.CodeLength)

	if err := sendFn(code); err != nil {
		uc.log.Errorf("发送%s验证码失败: %v", kind, err)
		return 0, ErrorOtpSendError
	}

	expiration := cfg.ExpiresIn.AsDuration()
	if err := uc.cache.Set(ctx, codeKey, code, expiration); err != nil {
		uc.log.Errorf("验证码已发送，但缓存验证码失败: %v", err)
		return 0, ErrorOtpSendError
	}

	if debug.IsDebug() {
		if info, ok := debug.FromContext(ctx); ok {
			info["otp"] = code
		}
	}

	return time.Now().Add(expiration).Unix(), nil
}

// VerifyPhoneOtp 校验手机验证码
func (uc *OtpUseCase) VerifyPhoneOtp(ctx context.Context, phone, scene, code string) (bool, error) {
	return uc.verify(ctx, kindPhone, scene, phone, code)
}

// VerifyEmailOtp 校验邮箱验证码
func (uc *OtpUseCase) VerifyEmailOtp(ctx context.Context, email, scene, code string) (bool, error) {
	return uc.verify(ctx, kindEmail, scene, email, code)
}

// 内部通用校验逻辑
func (uc *OtpUseCase) verify(ctx context.Context, kind, scene, receiver, inputCode string) (bool, error) {
	codeKey := fmt.Sprintf(otpCodeKeyPattern, kind, scene, receiver)
	failKey := fmt.Sprintf(otpFailKeyPattern, kind, scene, receiver)

	stored, err := uc.cache.Get(ctx, codeKey)
	if err != nil {
		if errors.Is(err, ErrOtpCacheMiss) {
			return false, ErrorOtpExpired
		}
		uc.log.Errorf("查询验证码失败: %v", err)
		return false, ErrorOtpSendError
	}

	if stored != inputCode {
		failCount, incrErr := uc.cache.Incr(ctx, failKey, otpFailExpiration)
		if incrErr != nil {
			uc.log.Errorf("增加验证码失败计数失败: %v", incrErr)
		}
		if failCount >= otpMaxFailCount {
			return false, ErrorOtpExpired
		}
		return false, ErrorOtpInvalid
	}

	if err := uc.cache.Del(ctx, codeKey); err != nil {
		uc.log.Errorf("删除验证码失败: %v", err)
		return true, ErrorOtpSendError
	}
	_ = uc.cache.Del(ctx, failKey)

	return true, nil
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
