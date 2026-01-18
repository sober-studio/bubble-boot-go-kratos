package auth

import (
	"time"

	"github.com/sober-studio/bubble-boot-go-kratos/internal/conf"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/pkg/auth/store"

	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
)

var ProviderSet = wire.NewSet(
	NewTokenService,
	NewTokenStore,
)

func NewTokenService(c *conf.App, store store.TokenStore) TokenService {
	// 默认有效期 30 天
	expire := 30 * 24 * time.Hour
	return NewJWTTokenService(c.Auth.Jwt.Secret, expire, store)
}

func NewTokenStore(c *conf.App, redis *redis.Client) store.TokenStore {
	// TODO: 按配置类型创建不同的 Token 存储器
	return store.NewRedisTokenStore(redis)
}
