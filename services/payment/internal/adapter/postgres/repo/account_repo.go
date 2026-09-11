package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/moneymate-2026/moneymate-backend/services/payment/internal/domain"
	"github.com/moneymate-2026/moneymate-backend/services/payment/sqlc/generated"
	apperrors "github.com/moneymate-2026/moneymate-backend/shared/pkg/errors"
	"github.com/moneymate-2026/moneymate-backend/shared/pkg/pgxtx"
)

type AccountRepo struct {
	pool *pgxpool.Pool
	q    *generated.Queries
}

func NewAccountRepo(pool *pgxpool.Pool) *AccountRepo {
	return &AccountRepo{pool: pool, q: generated.New(pool)}
}

func (r *AccountRepo) queries(ctx context.Context) *generated.Queries {
	if tx, ok := pgxtx.FromContext(ctx); ok {
		return r.q.WithTx(tx)
	}
	return r.q
}

func (r *AccountRepo) Create(ctx context.Context, a *domain.Account) (*domain.Account, error) {
	row, err := r.queries(ctx).CreateAccount(ctx, generated.CreateAccountParams{
		UserID:     uuidPtrToPgtype(a.UserID),
		MerchantID: uuidPtrToPgtype(a.MerchantID),
		Column3:    generated.PaymentAccountType(a.Type),
		Currency:   a.Currency,
	})
	if err != nil {
		return nil, mapDBErr(err)
	}
	return rowToAccount(generated.GetAccountByIDRow(row)), nil
}

func (r *AccountRepo) CreateWallet(ctx context.Context, a *domain.Account) (*domain.Account, error) {
	row, err := r.queries(ctx).CreateWallet(ctx, generated.CreateWalletParams{
		UserID:   uuidPtrToPgtype(a.UserID),
		Currency: a.Currency,
		Handle:   a.Handle,
	})
	if err != nil {
		return nil, mapDBErr(err)
	}
	return rowToAccount(generated.GetAccountByIDRow(row)), nil
}

func (r *AccountRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Account, error) {
	row, err := r.queries(ctx).GetAccountByID(ctx, id)
	if err != nil {
		return nil, mapDBErr(err)
	}
	return rowToAccount(row), nil
}

func (r *AccountRepo) GetByHandle(ctx context.Context, handle string) (*domain.Account, error) {
	row, err := r.queries(ctx).GetAccountByHandle(ctx, &handle)
	if err != nil {
		return nil, mapDBErr(err)
	}
	return rowToAccount(generated.GetAccountByIDRow(row)), nil
}

func (r *AccountRepo) GetWalletByUserID(ctx context.Context, userID uuid.UUID) (*domain.Account, error) {
	row, err := r.queries(ctx).GetWalletByUserID(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		return nil, mapDBErr(err)
	}
	return rowToAccount(generated.GetAccountByIDRow(row)), nil
}

func (r *AccountRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Account, error) {
	rows, err := r.queries(ctx).ListAccountsByUser(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		return nil, mapDBErr(err)
	}
	accounts := make([]*domain.Account, 0, len(rows))
	for _, row := range rows {
		accounts = append(accounts, rowToAccount(generated.GetAccountByIDRow(row)))
	}
	return accounts, nil
}

func (r *AccountRepo) GetTotalBalanceByUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	total, err := r.queries(ctx).GetTotalBalanceByUser(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		return 0, mapDBErr(err)
	}
	return total, nil
}

func (r *AccountRepo) AddBalance(ctx context.Context, id uuid.UUID, amount int64) error {
	return mapDBErr(r.queries(ctx).AddBalance(ctx, generated.AddBalanceParams{ID: id, Balance: amount}))
}

func rowToAccount(row generated.GetAccountByIDRow) *domain.Account {
	var userID, merchantID *uuid.UUID
	if row.UserID.Valid {
		id := uuid.UUID(row.UserID.Bytes)
		userID = &id
	}
	if row.MerchantID.Valid {
		id := uuid.UUID(row.MerchantID.Bytes)
		merchantID = &id
	}
	return &domain.Account{
		ID:         row.ID,
		UserID:     userID,
		MerchantID: merchantID,
		Type:       domain.AccountType(row.Type),
		Currency:   row.Currency,
		Balance:    row.Balance,
		Version:    row.Version,
		Handle:     row.Handle,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}
}

func (r *AccountRepo) GetExternalSettlementAccount(ctx context.Context) (*domain.Account, error) {
	row, err := r.queries(ctx).GetExternalSettlementAccount(ctx)
	if err != nil {
		return nil, mapDBErr(err)
	}
	return rowToAccount(generated.GetAccountByIDRow(row)), nil
}

func (r *AccountRepo) CreateExternalSettlementAccount(ctx context.Context) (*domain.Account, error) {
	row, err := r.queries(ctx).CreateExternalSettlementAccount(ctx)
	if err != nil {
		return nil, mapDBErr(err)
	}
	return rowToAccount(generated.GetAccountByIDRow(row)), nil
}

func mapDBErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, apperrors.ErrNotFound) {
		return err
	}
	mapped := apperrors.MapDBErrors(err)
	if mapped != nil {
		return mapped
	}
	return fmt.Errorf("db: %w", err)
}

func (r *AccountRepo) GetSystemAccountByType(ctx context.Context, accountType domain.AccountType) (*domain.Account, error) {
	row, err := r.queries(ctx).GetSystemAccountByType(ctx, generated.PaymentAccountType(accountType))
	if err != nil {
		return nil, mapDBErr(err)
	}
	return rowToAccount(generated.GetAccountByIDRow{
		ID:         row.ID,
		UserID:     row.UserID,
		MerchantID: row.MerchantID,
		Type:       string(row.Type),
		Currency:   row.Currency,
		Balance:    row.Balance,
		Version:    row.Version,
		Handle:     row.Handle,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}), nil
}