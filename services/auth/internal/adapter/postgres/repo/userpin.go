package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/moneymate-2026/moneymate-backend/auth/internal/domain"
	db "github.com/moneymate-2026/moneymate-backend/auth/sqlc/generated"
	apperrors "github.com/moneymate-2026/moneymate-backend/shared/pkg/errors"
)

type userPinRepo struct{ q *db.Queries }

func NewUserPinRepo(pool *pgxpool.Pool) domain.UserPinRepository {
	return &userPinRepo{q: db.New(pool)}
}

func (r *userPinRepo) Create(ctx context.Context, pin *domain.UserPin) error {
	err := r.q.CreatePIN(ctx, db.CreatePINParams{
		ID:      uuidToPgtype(pin.ID),
		UserID:  uuidToPgtype(pin.UserID),
		PinHash: pin.PinHash,
	})
	if err != nil {
		return apperrors.MapDBErrors(err)
	}
	return nil
}

func (r *userPinRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.UserPin, error) {
	row, err := r.q.GetPINByUserID(ctx, uuidToPgtype(userID))
	if err != nil {
		return nil, apperrors.MapDBErrors(err)
	}

	pin := &domain.UserPin{
		ID:             uuid.UUID(row.ID.Bytes),
		UserID:         uuid.UUID(row.UserID.Bytes),
		PinHash:        row.PinHash,
		FailedAttempts: row.FailedAttempts,
		CreatedAt:      row.CreatedAt.Time,
		UpdatedAt:      row.UpdatedAt.Time,
	}
	if row.LockedUntil.Valid {
		pin.LockedUntil = &row.LockedUntil.Time
	}
	return pin, nil
}

func (r *userPinRepo) IncrementFailedAttempts(ctx context.Context, userID uuid.UUID) (int32, error) {
	attempts, err := r.q.IncrementPINFailedAttempts(ctx, uuidToPgtype(userID))
	if err != nil {
		return 0, apperrors.MapDBErrors(err)
	}
	return attempts, nil
}

func (r *userPinRepo) Lock(ctx context.Context, userID uuid.UUID, lockedUntil time.Time) error {
	err := r.q.LockPIN(ctx, db.LockPINParams{
		UserID:      uuidToPgtype(userID),
		LockedUntil: timeToPgTimestamptz(lockedUntil),
	})
	return apperrors.MapDBErrors(err)
}

func (r *userPinRepo) Exists(ctx context.Context, userID uuid.UUID) (bool, error) {
	return r.q.PINExists(ctx, uuidToPgtype(userID))
}

func (r *userPinRepo) ResetFailedAttempts(ctx context.Context, userID uuid.UUID) error {
	return apperrors.MapDBErrors(r.q.ResetPINFailedAttempts(ctx, uuidToPgtype(userID)))
}

func (r *userPinRepo) UpdateHash(ctx context.Context, userID uuid.UUID, pinHash string) error {
	err := r.q.UpdatePINHash(ctx, db.UpdatePINHashParams{
		UserID:  uuidToPgtype(userID),
		PinHash: pinHash,
	})
	return apperrors.MapDBErrors(err)
}