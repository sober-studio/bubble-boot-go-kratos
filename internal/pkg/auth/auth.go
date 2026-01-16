package auth

import (
	"context"
	"strconv"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/pkg/auth/model"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/pkg/auth/store"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken     = errors.Unauthorized("INVALID_TOKEN", "无效的 Token")
	ErrTokenExpired     = errors.Unauthorized("TOKEN_EXPIRED", "Token 已过期")
	ErrJWTGenerateError = errors.Unauthorized("JWT_GENERATE_ERROR", "JWT 生成错误")
)

// TokenService 令牌服务接口，用于生成和解析 JWT 令牌
type TokenService interface {
	// GenerateToken 生成令牌
	GenerateToken(ctx context.Context, userID string) (string, error)
	// ParseTokenFromTokenString 解析令牌，返回用户ID
	ParseTokenFromTokenString(ctx context.Context, tokenStr string) (string, error)
	// ParseTokenFromContext 解析令牌，返回用户ID
	ParseTokenFromContext(ctx context.Context) (string, error)
	// GetUserIDFromTokenString 获取用户ID
	GetUserIDFromTokenString(ctx context.Context, tokenStr string) (int64, error)
	// GetUserIDFromContext 获取用户ID
	GetUserIDFromContext(ctx context.Context) (int64, error)
	// GetUserTokens 获取用户令牌
	GetUserTokens(ctx context.Context, userID string) (*[]model.UserToken, error)
	// RevokeToken 撤销令牌
	RevokeToken(ctx context.Context, jti string) error
	// RevokeAllTokens 撤销用户所有令牌
	RevokeAllTokens(ctx context.Context) error
	// GetSecretKey 获取密钥
	GetSecretKey() []byte
}

var _ TokenService = (*JWTTokenService)(nil)

// JWTTokenService JWT 令牌服务接口
type JWTTokenService struct {
	secretKey []byte
	ttl       time.Duration
	store     store.TokenStore
}

func NewJWTTokenService(secretKey string, ttl time.Duration, store store.TokenStore) TokenService {
	return &JWTTokenService{
		secretKey: []byte(secretKey),
		ttl:       ttl,
		store:     store,
	}
}

func (s *JWTTokenService) GenerateToken(ctx context.Context, userID string) (string, error) {
	jti := uuid.New().String()
	now := time.Now()
	claims := jwtv5.RegisteredClaims{
		Subject:   userID,
		ExpiresAt: jwtv5.NewNumericDate(now.Add(s.ttl)),
		IssuedAt:  jwtv5.NewNumericDate(now),
		ID:        jti,
	}
	tokenStr, err := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims).SignedString(s.secretKey)
	if err != nil {
		log.Errorf("Failed to generate token: %v", err)
		return "", ErrJWTGenerateError
	}
	log.Infof("Generated token: %s", tokenStr)
	token := &model.UserToken{
		JTI:       jti,
		UserID:    userID,
		IssuedAt:  now,
		ExpiresAt: now.Add(s.ttl),
		TokenStr:  tokenStr,
	}

	if err := s.store.SaveToken(ctx, token); err != nil {
		log.Error("Failed to save token: %v", err)
		return "", ErrJWTGenerateError
	}

	return tokenStr, nil
}

func (s *JWTTokenService) ParseTokenFromTokenString(ctx context.Context, tokenStr string) (string, error) {
	t, err := jwtv5.ParseWithClaims(tokenStr, &jwtv5.RegisteredClaims{}, func(token *jwtv5.Token) (interface{}, error) {
		return s.secretKey, nil
	})
	if err != nil || !t.Valid {
		return "", ErrInvalidToken
	}
	claims, ok := t.Claims.(*jwtv5.RegisteredClaims)
	if !ok {
		return "", ErrInvalidToken
	}

	stored, err := s.store.GetToken(ctx, claims.ID)
	if err != nil || stored.ExpiresAt.Before(time.Now()) {
		return "", ErrTokenExpired
	}
	return stored.UserID, nil
}

func (s *JWTTokenService) ParseTokenFromContext(ctx context.Context) (string, error) {
	claims, ok := jwt.FromContext(ctx)
	if !ok {
		log.Errorf("invalid token")
		return "", ErrInvalidToken
	}
	registeredClaims, ok := claims.(*jwtv5.RegisteredClaims)
	if !ok {
		log.Errorf("invalid token")
		return "", ErrInvalidToken
	}

	stored, err := s.store.GetToken(ctx, registeredClaims.ID)
	if err != nil || stored.ExpiresAt.Before(time.Now()) {
		return "", ErrTokenExpired
	}
	return stored.UserID, nil
}

func (s *JWTTokenService) GetUserIDFromTokenString(ctx context.Context, tokenStr string) (int64, error) {
	userID, err := s.ParseTokenFromTokenString(ctx, tokenStr)
	if err != nil {
		return 0, err
	}
	return parseUserID(userID)
}

func (s *JWTTokenService) GetUserIDFromContext(ctx context.Context) (int64, error) {
	userID, err := s.ParseTokenFromContext(ctx)
	if err != nil {
		return 0, err
	}
	return parseUserID(userID)
}

func (s *JWTTokenService) GetUserTokens(ctx context.Context, userID string) (*[]model.UserToken, error) {
	return s.store.GetUserTokens(ctx, userID)
}

// 将 storedUserID 字符串转换为 int64 类型的 userID
func parseUserID(storedUserID string) (int64, error) {
	userID, err := strconv.ParseInt(storedUserID, 10, 64)
	if err != nil {
		return 0, ErrInvalidToken.WithCause(err)
	}
	return userID, nil
}

func (s *JWTTokenService) RevokeToken(ctx context.Context, jti string) error {
	userID, err := s.ParseTokenFromContext(ctx)
	if err != nil {
		return err
	}
	return s.store.DeleteUserToken(ctx, userID, jti)
}

func (s *JWTTokenService) RevokeAllTokens(ctx context.Context) error {
	userID, err := s.ParseTokenFromContext(ctx)
	if err != nil {
		return err
	}
	return s.store.DeleteUserTokens(ctx, userID)
}

func (s *JWTTokenService) GetSecretKey() []byte {
	return s.secretKey
}
