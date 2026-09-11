package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Pod struct {
	ID           uuid.UUID
	AccountID    uuid.UUID
	UserID       uuid.UUID
	Name         string
	TargetAmount *int64 // paise, nullable
	TargetDate   *time.Time
	Icon         string
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type PodRepository interface {
	Create(ctx context.Context, pod *Pod) error
	GetByID(ctx context.Context, id uuid.UUID) (*Pod, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*Pod, error)
	Update(ctx context.Context, pod *Pod) error
	Delete(ctx context.Context, id uuid.UUID) error
}
