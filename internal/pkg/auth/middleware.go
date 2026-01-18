package auth

import (
	"context"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	jwtv5 "github.com/golang-jwt/jwt/v5"
)

// PathAccessConfig 路径访问配置
type PathAccessConfig struct {
	// 无需认证的路径
	PublicPaths map[string]struct{}
	// 认证后可访问的路径
	AuthPaths map[string]struct{}
}

// NewDefaultPathAccessConfig 创建默认路径访问配置
func NewDefaultPathAccessConfig() *PathAccessConfig {
	return &PathAccessConfig{
		PublicPaths: map[string]struct{}{
			"": {},
		},
		AuthPaths: map[string]struct{}{
			// 目前暂不判断，除公开接口列表中的路径外，均需要认证
		},
	}
}

// PathAccessConfigWithPublicList 创建路径访问配置
func PathAccessConfigWithPublicList(publicPaths []string) *PathAccessConfig {
	pathAccessConfig := &PathAccessConfig{
		PublicPaths: make(map[string]struct{}),
	}
	for _, path := range publicPaths {
		pathAccessConfig.PublicPaths[path] = struct{}{}
	}
	return pathAccessConfig
}

// IsPublicPath 判断是否为公开路径
func IsPublicPath(ctx context.Context, operation string, config *PathAccessConfig) bool {
	return Match(operation, config.PublicPaths)
}

// Match 判断路径是否匹配
func Match(operation string, paths map[string]struct{}) bool {
	_, ok := paths[operation]
	// 路径匹配
	if ok {
		return true
	}
	// 前缀匹配
	for path := range paths {
		if len(path) > 0 && path[len(path)-1] == '/' && len(operation) >= len(path) {
			if operation[:len(path)] == path {
				return true
			}
		}
	}
	return false
}

// JWTRecheck JWT 再次验证，从 TokenStore 中查询信息并验证
func JWTRecheck(tokenService TokenService) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// 验证 token
			_, err := tokenService.ParseTokenFromContext(ctx)
			if err != nil {
				return nil, err
			}
			// token 验证通过，继续处理
			return handler(ctx, req)
		}
	}
}

// JWTMiddleware 创建 JWT 认证中间件
func JWTMiddleware(tokenService TokenService) middleware.Middleware {
	return jwt.Server(
		func(token *jwtv5.Token) (interface{}, error) {
			return tokenService.GetSecretKey(), nil
		},
		jwt.WithSigningMethod(jwtv5.SigningMethodHS256),
		jwt.WithClaims(func() jwtv5.Claims {
			return &jwtv5.RegisteredClaims{}
		}),
	)
}

// Middleware 创建认证中间件
func Middleware(tokenService TokenService, config *PathAccessConfig) middleware.Middleware {
	return selector.Server(
		JWTMiddleware(tokenService),
		JWTRecheck(tokenService),
	).Match(func(ctx context.Context, operation string) bool {
		return !IsPublicPath(ctx, operation, config)
	}).Build()
}
