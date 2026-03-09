package driven

import (
	"context"
	"time"

	"graphiti/internal/domain"
)

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

type AuthProvider interface {
	GetAuthURL(state string) string
	ExchangeCode(ctx context.Context, code string) (*TokenPair, error)
	GetUserInfo(ctx context.Context, accessToken string) (*domain.UserInfo, error)
	RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error)
}
