package sms

import (
	"context"

	"github.com/go-kratos/kratos/v2/errors"
)

var (
	ErrorTemplateNotConfigured = errors.InternalServer("SMS_TEMPLATE_NOT_CONFIGURED", "短信模板未配置")
)

type Sender interface {
	Send(ctx context.Context, phone string, template string, params map[string]string) error
}
