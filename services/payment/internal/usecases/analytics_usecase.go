package usecases

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/moneymate-2026/moneymate-backend/services/payment/internal/domain"
	apperrors "github.com/moneymate-2026/moneymate-backend/shared/pkg/errors"
	"github.com/moneymate-2026/moneymate-backend/shared/pkg/money"
)

type SpendByCategoryItem struct {
	Category         string `json:"category"`
	TransactionCount int64  `json:"transaction_count"`
	TotalAmount      string `json:"total_amount"`
}

type SpendByPeriodItem struct {
	Period           string `json:"period"`
	TotalAmount      string `json:"total_amount"`
	TransactionCount int64  `json:"transaction_count"`
}

type AnalyticsUsecase interface {
	SpendByCategory(ctx context.Context, authUserID, fromStr, toStr string) ([]SpendByCategoryItem, error)
	SpendByPeriod(ctx context.Context, authUserID, granularityStr, fromStr, toStr string) ([]SpendByPeriodItem, error)
}

type analyticsUsecase struct {
	accounts     domain.AccountRepository
	transactions domain.TransactionRepository
}

func NewAnalyticsUsecase(accounts domain.AccountRepository, transactions domain.TransactionRepository) AnalyticsUsecase {
	return &analyticsUsecase{
		accounts:     accounts,
		transactions: transactions,
	}
}

func (u *analyticsUsecase) SpendByCategory(ctx context.Context, authUserID, fromStr, toStr string) ([]SpendByCategoryItem, error) {
	uid, err := uuid.Parse(authUserID)
	if err != nil {
		return nil, apperrors.ErrUnauthorized
	}

	acc, err := u.accounts.GetWalletByUserID(ctx, uid)
	if err != nil {
		return nil, err
	}

	from, to, err := ParseDateRange(fromStr, toStr, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	rows, err := u.transactions.GetSpendByCategory(ctx, acc.ID, from, to)
	if err != nil {
		return nil, err
	}

	items := make([]SpendByCategoryItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, SpendByCategoryItem{
			Category:         r.Category,
			TransactionCount: r.TransactionCount,
			TotalAmount:      money.FormatPaise(r.TotalAmount),
		})
	}

	return items, nil
}

func (u *analyticsUsecase) SpendByPeriod(ctx context.Context, authUserID, granularityStr, fromStr, toStr string) ([]SpendByPeriodItem, error) {
	uid, err := uuid.Parse(authUserID)
	if err != nil {
		return nil, apperrors.ErrUnauthorized
	}

	acc, err := u.accounts.GetWalletByUserID(ctx, uid)
	if err != nil {
		return nil, err
	}

	granularity := strings.ToLower(strings.TrimSpace(granularityStr))
	if granularity == "" {
		granularity = "day"
	}
	if granularity != "day" && granularity != "week" && granularity != "month" {
		return nil, apperrors.ErrInvalidInput
	}

	from, to, err := ParseDateRange(fromStr, toStr, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	rows, err := u.transactions.GetSpendByPeriod(ctx, acc.ID, from, to, granularity)
	if err != nil {
		return nil, err
	}

	items := make([]SpendByPeriodItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, SpendByPeriodItem{
			Period:           r.Period.UTC().Format("2006-01-02"),
			TotalAmount:      money.FormatPaise(r.TotalAmount),
			TransactionCount: r.TransactionCount,
		})
	}

	return items, nil
}

func ParseDateRange(fromStr, toStr string, now time.Time) (*time.Time, time.Time, error) {
	var from *time.Time
	var to time.Time

	if strings.TrimSpace(fromStr) != "" {
		parsedFrom, err := ParseFromDate(strings.TrimSpace(fromStr))
		if err != nil {
			return nil, time.Time{}, apperrors.ErrInvalidInput
		}
		from = &parsedFrom
	}

	if strings.TrimSpace(toStr) == "" {
		to = now
	} else {
		parsedTo, err := ParseToDate(strings.TrimSpace(toStr))
		if err != nil {
			return nil, time.Time{}, apperrors.ErrInvalidInput
		}
		to = parsedTo
	}

	if from != nil && !to.After(*from) {
		return nil, time.Time{}, apperrors.ErrInvalidInput
	}

	return from, to, nil
}

func parseDateRange(fromStr, toStr string, now time.Time) (*time.Time, time.Time, error) {
	return ParseDateRange(fromStr, toStr, now)
}

func ParseFromDate(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse("2006-01-02T15:04:05", s); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
	}
	return time.Time{}, apperrors.ErrInvalidInput
}

func parseFromDate(s string) (time.Time, error) {
	return ParseFromDate(s)
}

func ParseToDate(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse("2006-01-02T15:04:05", s); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		// Date-only: include the full day up to midnight of next day
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, 1), nil
	}
	return time.Time{}, apperrors.ErrInvalidInput
}

func parseToDate(s string) (time.Time, error) {
	return ParseToDate(s)
}
