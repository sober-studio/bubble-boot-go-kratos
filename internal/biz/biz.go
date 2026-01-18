package biz

import (
	"context"

	"github.com/google/wire"
)

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(
	NewCaptchaUseCase,
	NewOtpUseCase,
)

// Transaction 事务接口
type Transaction interface {
	InTx(context.Context, func(ctx context.Context) error) error
}
