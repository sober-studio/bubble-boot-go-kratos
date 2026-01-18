package email

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
)

type MockSender struct {
	log *log.Helper
}

func NewMockSender(logger log.Logger) Sender {
	return &MockSender{
		log: log.NewHelper(logger),
	}
}

func (m *MockSender) Send(ctx context.Context, target string, subject string, params map[string]string) error {
	m.log.Infof("mock send email: to=%s, subject=%s, params=%v", target, subject, params)
	return nil
}
