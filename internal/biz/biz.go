package biz

import (
	"context"

	"github.com/google/wire"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/pkg/email"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/pkg/sms"
)

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(
	NewCaptchaUseCase,
	NewOtpUseCase,
	sms.NewSmsSender,
	email.NewEmailSender,
	wire.Bind(new(SmsSender), new(sms.Sender)),
	wire.Bind(new(EmailSender), new(email.Sender)),
)

// Transaction 事务接口
type Transaction interface {
	InTx(context.Context, func(ctx context.Context) error) error
}
