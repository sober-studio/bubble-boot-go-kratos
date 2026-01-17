package data

import (
	"context"
	"strings"
	"time"

	"github.com/mojocn/base64Captcha"
)

// RedisCaptchaStore 实现 base64Captcha.Store 接口
type RedisCaptchaStore struct {
	data *Data
}

func NewRedisCaptchaStore(data *Data) base64Captcha.Store {
	return &RedisCaptchaStore{data: data}
}

func (s *RedisCaptchaStore) Set(id string, value string) error {
	// 验证码通常有效期较短，设为 5-10 分钟
	return s.data.RDB().Set(context.Background(), "captcha:"+id, value, 10*time.Minute).Err()
}

func (s *RedisCaptchaStore) Get(id string, clear bool) string {
	ctx := context.Background()
	key := "captcha:" + id
	val, err := s.data.RDB().Get(ctx, key).Result()
	if err != nil {
		return ""
	}
	if clear {
		s.data.RDB().Del(ctx, key)
	}
	return val
}

func (s *RedisCaptchaStore) Verify(id, answer string, clear bool) bool {
	vv := s.Get(id, clear)
	return strings.ToLower(vv) == strings.ToLower(answer)
}
