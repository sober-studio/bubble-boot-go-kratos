package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sober-studio/bubble-boot-go-kratos/internal/pkg/auth/model"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"

	"time"
)

var _ TokenStore = (*RedisTokenStore)(nil)

// RedisTokenStore 基于 Redis 的用户s Token 存储
type RedisTokenStore struct {
	client *redis.Client
}

/*
Redis Key 设计：
jwt:token:{jti} => string(json of UserToken) # 单个 Token
jwt:user:{userID}:tokens => set of jti # 用户 Token 索引
*/

func NewRedisTokenStore(redis *redis.Client) TokenStore {
	return &RedisTokenStore{
		client: redis,
	}
}

func (s *RedisTokenStore) SaveToken(ctx context.Context, token *model.UserToken) error {
	data, _ := json.Marshal(token)
	ttl := time.Until(token.ExpiresAt)
	if err := s.client.Set(ctx, s.tokenKey(token.JTI), data, ttl).Err(); err != nil {
		log.Errorf("Failed to save token: %v", err)
		return err
	}
	userKey := s.userSetKey(token.UserID)
	if err := s.client.SAdd(ctx, userKey, token.JTI).Err(); err != nil {
		log.Errorf("Failed to add token to user: %v", err)
		return err
	}
	return nil
}

func (s *RedisTokenStore) GetToken(ctx context.Context, jti string) (*model.UserToken, error) {
	data, err := s.client.Get(ctx, s.tokenKey(jti)).Bytes()
	if err != nil {
		return nil, err
	}
	var token model.UserToken
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

func (s *RedisTokenStore) DeleteUserToken(ctx context.Context, userID, jti string) error {
	userKey := s.userSetKey(userID)
	jtiSet, _ := s.client.SMembers(ctx, userKey).Result()
	contains := false
	for _, v := range jtiSet {
		if v == jti {
			contains = true
			break
		}
	}
	if !contains {
		return nil
	}
	return s.DeleteToken(ctx, jti)
}

func (s *RedisTokenStore) DeleteToken(ctx context.Context, jti string) error {
	token, err := s.GetToken(ctx, jti)
	if err == nil {
		s.client.SRem(ctx, s.userSetKey(token.UserID), jti)
	}
	return s.client.Del(ctx, s.tokenKey(jti)).Err()
}

func (s *RedisTokenStore) DeleteUserTokens(ctx context.Context, userID string) error {
	userKey := s.userSetKey(userID)
	jtiSet, _ := s.client.SMembers(ctx, userKey).Result()
	if len(jtiSet) > 0 {
		keys := make([]string, len(jtiSet))
		for i, jti := range jtiSet {
			keys[i] = s.tokenKey(jti)
		}
		s.client.Del(ctx, keys...)
	}
	return s.client.Del(ctx, userKey).Err()
}

func (s *RedisTokenStore) GetUserTokens(ctx context.Context, userID string) (*[]model.UserToken, error) {
	userKey := s.userSetKey(userID)
	jtiSet, err := s.client.SMembers(ctx, userKey).Result()
	// 创建 model.UserToken 数组，并遍历查询加入数组
	var tokens []model.UserToken
	if err == nil {
		for _, jti := range jtiSet {
			userToken, err := s.GetToken(ctx, jti)
			if err == nil {
				tokens = append(tokens, *userToken)
			} else {
				// Token 不存在或获取失败，从集合中移除
				s.client.SRem(ctx, userKey, jti)
			}
		}
	}
	return &tokens, nil
}

func (s *RedisTokenStore) tokenKey(jti string) string {
	return fmt.Sprintf("jwt:token:%s", jti)
}

func (s *RedisTokenStore) userSetKey(userID string) string {
	return fmt.Sprintf("jwt:user:%s:tokens", userID)
}
