package domain

import (
	"context"
	"time"
	"github.com/google/uuid"
)

type OAuthAccount struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	Provider       string
	ProviderUserID string
	CreatedAt      time.Time
}

type OAuthAccountRepository interface {
	Create(ctx context.Context, account *OAuthAccount) error
	Get(ctx context.Context, provider, providerUserID string) (*OAuthAccount, error)
	GetByUser(ctx context.Context, userID uuid.UUID) ([]OAuthAccount, error)
}