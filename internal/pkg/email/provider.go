package email

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/conf"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/pkg/env"
)

var ProviderSet = wire.NewSet(NewEmailSender)

func NewEmailSender(c *conf.Data, logger log.Logger) Sender {
	if env.IsDev() {
		return NewMockSender(logger)
	}
	// 默认使用 SMTP 实现
	return NewSmtpSender(c.Email, logger)
}
