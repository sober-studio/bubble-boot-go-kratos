package sms

import "context"

type Sender interface {
	Send(ctx context.Context, phone string, template string, params map[string]string) error
}
