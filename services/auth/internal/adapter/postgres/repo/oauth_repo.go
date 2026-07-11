package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/moneymate-2026/moneymate-backend/auth/internal/domain"
	db "github.com/moneymate-2026/moneymate-backend/auth/sqlc/generated"
	apperrors "github.com/moneymate-2026/moneymate-backend/shared/pkg/errors"
)

type oauthRepo struct { q *db.Queries }

func NewOAuthAccountRepo(pool *pgxpool.Pool) domain.OAuthAccountRepository {
	return &oauthRepo{q: db.New(pool)}
}

func (r *oauthRepo) Create(ctx context.Context, acc *domain.OAuthAccount) error {
	row, err := r.q.CreateOAuthAccount(ctx, db.CreateOAuthAccountParams{
		ID:             uuidToPgtype(acc.ID),
		UserID:         uuidToPgtype(acc.UserID),
		Provider:       acc.Provider,
		ProviderUserID: acc.ProviderUserID,
	})
	if err != nil { return apperrors.MapDBErrors(err) }
	acc.CreatedAt = row.CreatedAt.Time
	return nil
}

func (r *oauthRepo) Get(ctx context.Context, provider, providerUserID string) (*domain.OAuthAccount, error) {
	row, err := r.q.GetOAuthAccount(ctx, db.GetOAuthAccountParams{
		Provider:       provider,
		ProviderUserID: providerUserID,
	})
	if err != nil { return nil, apperrors.MapDBErrors(err) }
	
	return &domain.OAuthAccount{
		ID:             uuid.UUID(row.ID.Bytes),
		UserID:         uuid.UUID(row.UserID.Bytes),
		Provider:       row.Provider,
		ProviderUserID: row.ProviderUserID,
		CreatedAt:      row.CreatedAt.Time,
	}, nil
}

func (r *oauthRepo) GetByUser(ctx context.Context, userID uuid.UUID) ([]domain.OAuthAccount, error) {
	rows, err := r.q.GetOAuthAccountsByUser(ctx, uuidToPgtype(userID))
	if err != nil { return nil, apperrors.MapDBErrors(err) }
	
	accounts := make([]domain.OAuthAccount, len(rows))
	for i, row := range rows {
		accounts[i] = domain.OAuthAccount{
			ID:             uuid.UUID(row.ID.Bytes),
			UserID:         uuid.UUID(row.UserID.Bytes),
			Provider:       row.Provider,
			ProviderUserID: row.ProviderUserID,
			CreatedAt:      row.CreatedAt.Time,
		}
	}
	return accounts, nil
}