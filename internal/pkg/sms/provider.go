package sms

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/conf"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/pkg/env"
)

// ProviderSet 给 Wire 使用
var ProviderSet = wire.NewSet(NewSmsSender)

func NewSmsSender(c *conf.Data, logger log.Logger) Sender {
	// 1. 如果是开发环境，强制返回 Mock
	if env.IsDev() {
		return NewMockSender(logger)
	}

	// 2. 根据配置文件决定使用哪个供应商
	switch c.Sms.Provider {
	case "aliyun":
		return NewAliyunSender(c, logger)
	case "tencent":
		// return NewTencentSender(c.Sms)
	}
	return NewMockSender(logger)
}
