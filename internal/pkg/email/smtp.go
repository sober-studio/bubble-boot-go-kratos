package email

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"math"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/conf"
	"gopkg.in/gomail.v2"
)

// 重试配置常量
const (
	maxRetries  = 3               // 最大重试次数
	baseBackoff = 1 * time.Second // 基础退避时间
	maxBackoff  = 6 * time.Second // 最大等待时间
)

type smtpSender struct {
	conf   *conf.Data_Email
	dialer *gomail.Dialer
	tmpl   *template.Template
	log    *log.Helper
}

func NewSmtpSender(c *conf.Data_Email, logger log.Logger) Sender {
	// 1. 预编译所有模板到内存池，提高发送性能
	// 注意：模板名对应文件名，如 "bind_email.html"
	tmpl := template.Must(template.ParseFS(templateFS, "templates/*.html"))

	// 2. 预创建分配器实例 (单例配置)
	dialer := gomail.NewDialer(c.Smtp.Host, int(c.Smtp.Port), c.Smtp.Username, c.Smtp.Password)

	return &smtpSender{
		conf:   c,
		dialer: dialer,
		tmpl:   tmpl,
		log:    log.NewHelper(logger),
	}
}

func (s *smtpSender) Send(ctx context.Context, target string, logicTemplate string, params map[string]string) error {
	// 1. 独立渲染逻辑
	htmlBody, err := s.render(logicTemplate, params)
	if err != nil {
		return err
	}

	// 2. 构造邮件
	m := gomail.NewMessage()
	m.SetHeader("From", s.conf.From)
	m.SetHeader("To", target)
	m.SetHeader("Subject", s.conf.SubjectMapping[logicTemplate])
	m.SetBody("text/html", htmlBody)

	// 3. 带指数退避的重试发送逻辑
	return s.sendWithRetry(ctx, m)
}

func (s *smtpSender) sendWithRetry(ctx context.Context, m *gomail.Message) error {
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		// 执行发送
		err := s.dialer.DialAndSend(m)
		if err == nil {
			if i > 0 {
				s.log.Infof("Email sent successfully after %d retries", i)
			}
			return nil
		}

		lastErr = err

		// 如果是最后一次尝试，不再等待
		if i == maxRetries-1 {
			break
		}

		// 计算退避时间: base * 2^attempt
		// 第 1 次失败后等 1s, 第 2 次失败后等 2s, 第 3 次失败后等 4s...
		backoff := time.Duration(math.Pow(2, float64(i))) * baseBackoff
		if backoff > maxBackoff {
			backoff = maxBackoff
		}

		s.log.Warnf("Email send failed (attempt %d/%d): %v. Retrying in %v...", i+1, maxRetries, err, backoff)

		// 等待并响应 context 取消（防止在等待重试时进程退出或请求超时）
		select {
		case <-time.After(backoff):
			// 继续下一次循环
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return errors.InternalServer("SMTP_SEND_FAILED", "邮件发送失败").WithCause(
		fmt.Errorf("failed to send email after %d attempts: %w", maxRetries, lastErr),
	)
}

// render 实现模板渲染，逻辑上独立于发送协议
func (s *smtpSender) render(logicTemplate string, params map[string]string) (string, error) {
	var body bytes.Buffer
	// 执行预编译好的模板
	err := s.tmpl.ExecuteTemplate(&body, logicTemplate+".html", params)
	if err != nil {
		return "", err
	}
	return body.String(), nil
}
