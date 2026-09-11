package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/moneymate-2026/moneymate-backend/services/payment/internal/domain"
	"github.com/moneymate-2026/moneymate-backend/services/payment/sqlc/generated"
	"github.com/moneymate-2026/moneymate-backend/shared/pkg/pgxtx"
)

type PodRepo struct {
	q *generated.Queries
}

func NewPodRepo(pool *pgxpool.Pool) *PodRepo {
	return &PodRepo{q: generated.New(pool)}
}

func (r *PodRepo) queries(ctx context.Context) *generated.Queries {
	if tx, ok := pgxtx.FromContext(ctx); ok {
		return r.q.WithTx(tx)
	}
	return r.q
}

func (r *PodRepo) Create(ctx context.Context, pod *domain.Pod) error {
	if pod.ID == uuid.Nil {
		pod.ID = uuid.New()
	}
	if pod.Status == "" {
		pod.Status = "active"
	}
	var iconPtr *string
	if pod.Icon != "" {
		iconPtr = &pod.Icon
	}
	row, err := r.queries(ctx).CreatePod(ctx, generated.CreatePodParams{
		ID:           pod.ID,
		AccountID:    pod.AccountID,
		UserID:       pod.UserID,
		Name:         pod.Name,
		TargetAmount: pod.TargetAmount,
		TargetDate:   datePtrToPgtype(pod.TargetDate),
		Icon:         iconPtr,
		Status:       pod.Status,
	})
	if err != nil {
		return mapDBErr(err)
	}
	*pod = *toDomainPod(row)
	return nil
}

func (r *PodRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Pod, error) {
	row, err := r.queries(ctx).GetPodByID(ctx, id)
	if err != nil {
		return nil, mapDBErr(err)
	}
	return toDomainPod(row), nil
}

func (r *PodRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Pod, error) {
	rows, err := r.queries(ctx).ListPodsByUser(ctx, userID)
	if err != nil {
		return nil, mapDBErr(err)
	}
	out := make([]*domain.Pod, len(rows))
	for i, row := range rows {
		out[i] = toDomainPod(row)
	}
	return out, nil
}

func (r *PodRepo) Update(ctx context.Context, pod *domain.Pod) error {
	var iconPtr *string
	if pod.Icon != "" {
		iconPtr = &pod.Icon
	}
	row, err := r.queries(ctx).UpdatePod(ctx, generated.UpdatePodParams{
		ID:           pod.ID,
		Name:         pod.Name,
		TargetAmount: pod.TargetAmount,
		TargetDate:   datePtrToPgtype(pod.TargetDate),
		Icon:         iconPtr,
		Status:       pod.Status,
	})
	if err != nil {
		return mapDBErr(err)
	}
	*pod = *toDomainPod(row)
	return nil
}

func (r *PodRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return mapDBErr(r.queries(ctx).DeletePod(ctx, id))
}

func toDomainPod(row generated.PaymentPod) *domain.Pod {
	var icon string
	if row.Icon != nil {
		icon = *row.Icon
	}
	return &domain.Pod{
		ID:           row.ID,
		AccountID:    row.AccountID,
		UserID:       row.UserID,
		Name:         row.Name,
		TargetAmount: row.TargetAmount,
		TargetDate:   pgtypeDateToTimePtr(row.TargetDate),
		Icon:         icon,
		Status:       row.Status,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}
