package repo


import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/moneymate-2026/moneymate-backend/auth/internal/domain"
	db "github.com/moneymate-2026/moneymate-backend/auth/sqlc/generated"
	apperrors "github.com/moneymate-2026/moneymate-backend/shared/pkg/errors"
)

type refreshTokenRepo struct { q *db.Queries }

func NewRefreshTokenRepo(pool *pgxpool.Pool) domain.RefreshTokenRepository {
	return &refreshTokenRepo{q: db.New(pool)}
}

func (r *refreshTokenRepo) Create(ctx context.Context, token *domain.RefreshToken) error {
	row, err := r.q.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		ID:        uuidToPgtype(token.ID),
		UserID:    uuidToPgtype(token.UserID),
		TokenHash: token.TokenHash,
		ExpiresAt: timeToPgTimestamptz(token.ExpiresAt),
	})
	if err != nil { return apperrors.MapDBErrors(err) }
	
	token.CreatedAt = row.CreatedAt.Time
	return nil
}

func (r *refreshTokenRepo) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	row, err := r.q.GetRefreshToken(ctx, tokenHash)
	if err != nil { return nil, apperrors.MapDBErrors(err) }
	
	token := &domain.RefreshToken{
		ID:        uuid.UUID(row.ID.Bytes),
		UserID:    uuid.UUID(row.UserID.Bytes),
		TokenHash: row.TokenHash,
		ExpiresAt: row.ExpiresAt.Time,
		CreatedAt: row.CreatedAt.Time,
	}
	if row.RevokedAt.Valid { token.RevokedAt = &row.RevokedAt.Time }
	return token, nil
}

func (r *refreshTokenRepo) Revoke(ctx context.Context, tokenHash string) error {
	return apperrors.MapDBErrors(r.q.RevokeRefreshToken(ctx, tokenHash))
}

func (r *refreshTokenRepo) DeleteExpired(ctx context.Context) error {
	return apperrors.MapDBErrors(r.q.DeleteExpiredRefreshTokens(ctx))
}