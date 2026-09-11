package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type TxStatus string

const (
	TxStatusPending   TxStatus = "pending"
	TxStatusCompleted TxStatus = "completed"
	TxStatusFailed    TxStatus = "failed"
	TxStatusReversed  TxStatus = "reversed"
)

type Transaction struct {
	ID             uuid.UUID
	FromAccountID  uuid.UUID
	ToAccountID    uuid.UUID
	Amount         int64 // paise
	Status         TxStatus
	IdempotencyKey string
	Description    string
	CategoryID     *uuid.UUID 
	CreatedAt      time.Time
	CompletedAt    *time.Time
}

type Category struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CategoryRepository interface {
	Create(ctx context.Context, userID uuid.UUID, name string) (*Category, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]Category, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Category, error)
	Update(ctx context.Context, id, userID uuid.UUID, name string) (*Category, error)
	Delete(ctx context.Context, id, userID uuid.UUID) error
}

type SpendByCategory struct {
	Category         string
	TransactionCount int64
	TotalAmount      int64 // paise
}

type SpendByPeriod struct {
	Period           time.Time
	TotalAmount      int64 // paise
	TransactionCount int64
}

type JournalEntry struct {
	ID            uuid.UUID
	TransactionID uuid.UUID
	AccountID     uuid.UUID
	Amount        int64 // paise
	Direction     string // "debit" | "credit"
	CreatedAt     time.Time
}

type TransactionRepository interface {
	Create(ctx context.Context, t *Transaction) error
	GetByID(ctx context.Context, id uuid.UUID) (*Transaction, error)
	GetByIdempotencyKey(ctx context.Context, key string, fromAccountID uuid.UUID) (*Transaction, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status TxStatus) error
	ListByAccount(ctx context.Context, accountID uuid.UUID) ([]*Transaction, error)
	GetEntriesByTransactionID(ctx context.Context, txID uuid.UUID) ([]*JournalEntry, error)
	ListWithdrawals(ctx context.Context, settlementAccountID uuid.UUID, fromAccountID *uuid.UUID, limit, offset int32) ([]*Transaction, int64, error)
	ListByAccountPaginated(ctx context.Context, accountID uuid.UUID, categoryID *uuid.UUID, from, to *time.Time, limit, offset int32) ([]*Transaction, int64, error)
	GetSpendByCategory(ctx context.Context, accountID uuid.UUID, from *time.Time, to time.Time) ([]SpendByCategory, error)
	GetSpendByPeriod(ctx context.Context, accountID uuid.UUID, from *time.Time, to time.Time, granularity string) ([]SpendByPeriod, error)
}