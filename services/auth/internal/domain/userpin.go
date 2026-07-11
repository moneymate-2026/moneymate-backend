package domain

import (
	"context"
	"time"
	"github.com/google/uuid"
)

type UserPin struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	PinHash        string
	FailedAttempts int32
	LockedUntil    *time.Time // Pointer because it can be NULL
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type UserPinRepository interface {
	Create(ctx context.Context, pin *UserPin) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (*UserPin, error)
	IncrementFailedAttempts(ctx context.Context, userID uuid.UUID) (int32, error)
	Lock(ctx context.Context, userID uuid.UUID, lockedUntil time.Time) error
	Exists(ctx context.Context, userID uuid.UUID) (bool, error)
	ResetFailedAttempts(ctx context.Context, userID uuid.UUID) error
	UpdateHash(ctx context.Context, userID uuid.UUID, pinHash string) error
}