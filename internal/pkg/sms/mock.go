package sms

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
)

type mockSender struct {
	log *log.Helper
}

func NewMockSender(logger log.Logger) Sender {
	return &mockSender{
		log: log.NewHelper(logger),
	}
}

func (s *mockSender) Send(ctx context.Context, phone string, template string, params map[string]string) error {
	s.log.Infof("mock send sms: phone=%s, template=%s, params=%v", phone, template, params)
	return nil
}
