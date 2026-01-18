package email

import (
	"context"
	"embed"
)

//go:embed templates/*.html
var templateFS embed.FS

type Sender interface {
	Send(ctx context.Context, to, template string, params map[string]string) error
}
