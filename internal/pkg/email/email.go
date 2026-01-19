package email

import (
	"context"
	"embed"

	"github.com/go-kratos/kratos/v2/errors"
)

var (
	ErrorTemplateNotConfigured = errors.InternalServer("EMAIL_TEMPLATE_NOT_CONFIGURED", "邮件模板未配置")
)

//go:embed templates/*.html
var templateFS embed.FS

type Sender interface {
	Send(ctx context.Context, to, template string, params map[string]string) error
}
