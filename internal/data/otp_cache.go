package data

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/biz"
)

type redisOtpCache struct {
	data *Data
}

func NewRedisOtpCache(data *Data) biz.OtpCache {
	return &redisOtpCache{data: data}
}

func (r *redisOtpCache) Set(ctx context.Context, k, v string, exp time.Duration) error {
	return r.data.RDB().Set(ctx, k, v, exp).Err()
}

func (r *redisOtpCache) Get(ctx context.Context, k string) (string, error) {
	res, err := r.data.RDB().Get(ctx, k).Result()
	if errors.Is(err, redis.Nil) {
		return "", biz.ErrOtpCacheMiss
	}
	if err != nil {
		return "", err
	}
	return res, nil
}

func (r *redisOtpCache) Del(ctx context.Context, k string) error {
	return r.data.RDB().Del(ctx, k).Err()
}

func (r *redisOtpCache) Exists(ctx context.Context, k string) (bool, error) {
	i, err := r.data.RDB().Exists(ctx, k).Result()
	return i > 0, err
}
