package usecases

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/moneymate-2026/moneymate-backend/services/payment/internal/domain"
	apperrors "github.com/moneymate-2026/moneymate-backend/shared/pkg/errors"
	"github.com/moneymate-2026/moneymate-backend/shared/pkg/money"
)

type CreatePodInput struct {
	AuthenticatedUserID string
	Name                string
	TargetAmountPaise   *int64
	TargetDate          *time.Time
	Icon                string
}

type PodTransferInput struct {
	AuthenticatedUserID string
	PodID               string
	AmountPaise         int64
	Direction           string // "deposit" | "withdraw"
	IdempotencyKey      string
}

type PodTransferResult struct {
	TransactionID      string    `json:"transaction_id"`
	PodID              string    `json:"pod_id"`
	Direction          string    `json:"direction"`
	AmountPaise        int64     `json:"amount_paise"`
	Amount             string    `json:"amount"`
	WalletBalancePaise int64     `json:"wallet_balance_paise"`
	WalletBalance      string    `json:"wallet_balance"`
	PodBalancePaise    int64     `json:"pod_balance_paise"`
	PodBalance         string    `json:"pod_balance"`
	Status             string    `json:"status"`
	CreatedAt          time.Time `json:"created_at"`
}

type UpdatePodInput struct {
	AuthenticatedUserID string
	PodID               string
	Name                *string
	TargetAmountPaise   *int64
	TargetDate          *time.Time
	Icon                *string
}

type PodDetail struct {
	ID                string    `json:"id"`
	AccountID         string    `json:"account_id"`
	UserID            string    `json:"user_id"`
	Name              string    `json:"name"`
	TargetAmountPaise *int64    `json:"target_amount_paise"`
	TargetAmount      *string   `json:"target_amount"`
	TargetDate        *string   `json:"target_date"`
	Icon              string    `json:"icon"`
	Status            string    `json:"status"`
	BalancePaise      int64     `json:"balance_paise"`
	Balance           string    `json:"balance"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type PodUsecase interface {
	CreatePod(ctx context.Context, in CreatePodInput) (*PodDetail, error)
	ListPods(ctx context.Context, authUserID string) ([]*PodDetail, error)
	GetPodByID(ctx context.Context, authUserID, podID string) (*PodDetail, error)
	TransferToPod(ctx context.Context, in PodTransferInput) (*PodTransferResult, error)
	UpdatePod(ctx context.Context, in UpdatePodInput) (*PodDetail, error)
	DeletePod(ctx context.Context, authUserID, podID string) error
}

type podUsecase struct {
	accounts domain.AccountRepository
	pods     domain.PodRepository
	tx       domain.TxManager
	ledger   domain.LedgerRepository
}

func NewPodUsecase(accounts domain.AccountRepository, pods domain.PodRepository, tx domain.TxManager, ledger domain.LedgerRepository) PodUsecase {
	return &podUsecase{
		accounts: accounts,
		pods:     pods,
		tx:       tx,
		ledger:   ledger,
	}
}

func (u *podUsecase) CreatePod(ctx context.Context, in CreatePodInput) (*PodDetail, error) {
	authUserID, err := uuid.Parse(in.AuthenticatedUserID)
	if err != nil {
		return nil, apperrors.ErrUnauthorized
	}

	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, apperrors.ErrInvalidInput
	}

	if in.TargetAmountPaise != nil && *in.TargetAmountPaise <= 0 {
		return nil, apperrors.ErrInvalidInput
	}

	var (
		createdAcc *domain.Account
		createdPod *domain.Pod
	)

	err = u.tx.WithTx(ctx, func(txCtx context.Context) error {
		acc, err := u.accounts.Create(txCtx, &domain.Account{
			UserID:   &authUserID,
			Type:     domain.AccountTypePod,
			Currency: "INR",
			Balance:  0,
		})
		if err != nil {
			return err
		}
		createdAcc = acc

		pod := &domain.Pod{
			AccountID:    acc.ID,
			UserID:       authUserID,
			Name:         name,
			TargetAmount: in.TargetAmountPaise,
			TargetDate:   in.TargetDate,
			Icon:         strings.TrimSpace(in.Icon),
			Status:       "active",
		}
		if err := u.pods.Create(txCtx, pod); err != nil {
			return err
		}
		createdPod = pod
		return nil
	})
	if err != nil {
		return nil, err
	}

	return toPodDetail(createdPod, createdAcc.Balance), nil
}

func (u *podUsecase) ListPods(ctx context.Context, authUserID string) ([]*PodDetail, error) {
	uid, err := uuid.Parse(authUserID)
	if err != nil {
		return nil, apperrors.ErrUnauthorized
	}

	pods, err := u.pods.ListByUser(ctx, uid)
	if err != nil {
		return nil, err
	}

	accounts, err := u.accounts.ListByUser(ctx, uid)
	if err != nil {
		return nil, err
	}
	accMap := make(map[uuid.UUID]*domain.Account, len(accounts))
	for _, acc := range accounts {
		accMap[acc.ID] = acc
	}

	out := make([]*PodDetail, len(pods))
	for i, pod := range pods {
		var balance int64
		if acc, ok := accMap[pod.AccountID]; ok {
			balance = acc.Balance
		}
		out[i] = toPodDetail(pod, balance)
	}

	return out, nil
}

func (u *podUsecase) GetPodByID(ctx context.Context, authUserID, podIDStr string) (*PodDetail, error) {
	uid, err := uuid.Parse(authUserID)
	if err != nil {
		return nil, apperrors.ErrUnauthorized
	}

	pid, err := uuid.Parse(podIDStr)
	if err != nil {
		return nil, apperrors.ErrNotFound
	}

	pod, err := u.pods.GetByID(ctx, pid)
	if err != nil {
		return nil, err
	}

	if pod.UserID != uid {
		return nil, apperrors.ErrNotFound
	}

	acc, err := u.accounts.GetByID(ctx, pod.AccountID)
	if err != nil {
		return nil, err
	}

	return toPodDetail(pod, acc.Balance), nil
}

func (u *podUsecase) TransferToPod(ctx context.Context, in PodTransferInput) (*PodTransferResult, error) {
	authUserID, err := uuid.Parse(in.AuthenticatedUserID)
	if err != nil {
		return nil, apperrors.ErrUnauthorized
	}

	podID, err := uuid.Parse(in.PodID)
	if err != nil {
		return nil, apperrors.ErrNotFound
	}

	if in.AmountPaise <= 0 {
		return nil, apperrors.ErrInvalidInput
	}

	direction := strings.ToLower(strings.TrimSpace(in.Direction))
	if direction != "deposit" && direction != "withdraw" {
		return nil, apperrors.ErrInvalidInput
	}

	key := strings.TrimSpace(in.IdempotencyKey)
	if key == "" {
		key = uuid.New().String()
	}

	pod, err := u.pods.GetByID(ctx, podID)
	if err != nil {
		return nil, err
	}

	// Critical security requirement: before executing any transfer, verify the pod's user_id matches the authenticated user's ID.
	if pod.UserID != authUserID {
		return nil, apperrors.ErrForbidden
	}

	wallet, err := u.accounts.GetWalletByUserID(ctx, authUserID)
	if err != nil {
		return nil, err
	}

	var fromID, toID uuid.UUID
	var desc string
	if direction == "deposit" {
		fromID = wallet.ID
		toID = pod.AccountID
		desc = fmt.Sprintf("Deposit to pod: %s", pod.Name)
	} else {
		fromID = pod.AccountID
		toID = wallet.ID
		desc = fmt.Sprintf("Withdraw from pod: %s", pod.Name)
	}

	res, err := u.ledger.ExecuteTransfer(ctx, &domain.Transaction{
		FromAccountID:  fromID,
		ToAccountID:    toID,
		Amount:         in.AmountPaise,
		Status:         domain.TxStatusPending,
		IdempotencyKey: key,
		Description:    desc,
	})
	if err != nil {
		return nil, err
	}

	var walletBal, podBal int64
	if direction == "deposit" {
		walletBal = res.FromBalance
		podBal = res.ToBalance
	} else {
		podBal = res.FromBalance
		walletBal = res.ToBalance
	}

	return &PodTransferResult{
		TransactionID:      res.Transaction.ID.String(),
		PodID:              pod.ID.String(),
		Direction:          direction,
		AmountPaise:        in.AmountPaise,
		Amount:             money.FormatPaise(in.AmountPaise),
		WalletBalancePaise: walletBal,
		WalletBalance:      money.FormatPaise(walletBal),
		PodBalancePaise:    podBal,
		PodBalance:         money.FormatPaise(podBal),
		Status:             string(res.Transaction.Status),
		CreatedAt:          res.Transaction.CreatedAt,
	}, nil
}

func (u *podUsecase) UpdatePod(ctx context.Context, in UpdatePodInput) (*PodDetail, error) {
	authUserID, err := uuid.Parse(in.AuthenticatedUserID)
	if err != nil {
		return nil, apperrors.ErrUnauthorized
	}

	podID, err := uuid.Parse(in.PodID)
	if err != nil {
		return nil, apperrors.ErrNotFound
	}

	pod, err := u.pods.GetByID(ctx, podID)
	if err != nil {
		return nil, err
	}

	// Ownership check: must belong to the authenticated user, return 404 otherwise
	if pod.UserID != authUserID {
		return nil, apperrors.ErrNotFound
	}

	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			return nil, apperrors.ErrInvalidInput
		}
		pod.Name = name
	}

	if in.TargetAmountPaise != nil {
		if *in.TargetAmountPaise <= 0 {
			return nil, apperrors.ErrInvalidInput
		}
		pod.TargetAmount = in.TargetAmountPaise
	}

	if in.TargetDate != nil {
		pod.TargetDate = in.TargetDate
	}

	if in.Icon != nil {
		pod.Icon = strings.TrimSpace(*in.Icon)
	}

	if err := u.pods.Update(ctx, pod); err != nil {
		return nil, err
	}

	acc, err := u.accounts.GetByID(ctx, pod.AccountID)
	if err != nil {
		return nil, err
	}

	return toPodDetail(pod, acc.Balance), nil
}

func (u *podUsecase) DeletePod(ctx context.Context, authUserID, podIDStr string) error {
	uid, err := uuid.Parse(authUserID)
	if err != nil {
		return apperrors.ErrUnauthorized
	}

	pid, err := uuid.Parse(podIDStr)
	if err != nil {
		return apperrors.ErrNotFound
	}

	pod, err := u.pods.GetByID(ctx, pid)
	if err != nil {
		return err
	}

	// Ownership check: must belong to the authenticated user, return 404 otherwise
	if pod.UserID != uid {
		return apperrors.ErrNotFound
	}

	acc, err := u.accounts.GetByID(ctx, pod.AccountID)
	if err != nil {
		return err
	}

	// Before allowing deletion, check pod balance. If non-zero, reject delete.
	if acc.Balance != 0 {
		return apperrors.ErrPodNonZeroBalance
	}

	return u.pods.Delete(ctx, pid)
}

func toPodDetail(pod *domain.Pod, balance int64) *PodDetail {
	var targetAmountStr *string
	if pod.TargetAmount != nil {
		f := money.FormatPaise(*pod.TargetAmount)
		targetAmountStr = &f
	}
	var targetDateStr *string
	if pod.TargetDate != nil {
		d := pod.TargetDate.UTC().Format("2006-01-02")
		targetDateStr = &d
	}

	return &PodDetail{
		ID:                pod.ID.String(),
		AccountID:         pod.AccountID.String(),
		UserID:            pod.UserID.String(),
		Name:              pod.Name,
		TargetAmountPaise: pod.TargetAmount,
		TargetAmount:      targetAmountStr,
		TargetDate:        targetDateStr,
		Icon:              pod.Icon,
		Status:            pod.Status,
		BalancePaise:      balance,
		Balance:           money.FormatPaise(balance),
		CreatedAt:         pod.CreatedAt,
		UpdatedAt:         pod.UpdatedAt,
	}
}
