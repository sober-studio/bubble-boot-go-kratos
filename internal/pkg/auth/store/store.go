package store

import (
	"context"

	"github.com/sober-studio/bubble-boot-go-kratos/internal/pkg/auth/model"
)

type TokenStore interface {
	SaveToken(ctx context.Context, token *model.UserToken) error
	GetToken(ctx context.Context, jti string) (*model.UserToken, error)
	DeleteUserToken(ctx context.Context, userID, jti string) error
	DeleteToken(ctx context.Context, jti string) error
	DeleteUserTokens(ctx context.Context, userID string) error
	GetUserTokens(ctx context.Context, userID string) (*[]model.UserToken, error)
}
