package usecases

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	authclient "github.com/moneymate-2026/moneymate-backend/services/payment/internal/adapter/authClient"
	// merchantclient "github.com/moneymate-2026/moneymate-backend/services/payment/internal/adapter/merchantClient"
	"github.com/moneymate-2026/moneymate-backend/services/payment/internal/domain"
	apperrors "github.com/moneymate-2026/moneymate-backend/shared/pkg/errors"
	"github.com/moneymate-2026/moneymate-backend/shared/pkg/money"
)

type TransferInput struct {
	AuthenticatedUserID string
	ToHandle            string
	AmountPaise         int64
	IdempotencyKey      string
	Description         string
	CategoryID          *string
}

type TransferUsecase interface {
	Transfer(ctx context.Context, in TransferInput) (*domain.LedgerResult, error)
	GetByID(ctx context.Context, id string) (*domain.Transaction, error)
	ResolveHandle(ctx context.Context, handle string) (*ResolveResult, error)
	ListMyTransactions(ctx context.Context, in ListTransactionsInput) (*ListTransactionsResult, error)
}

type AuthClient interface {
	GetUserProfile(ctx context.Context, userID string) (*authclient.UserProfile, error)
}

type MerchantClient interface {
	GetStoreProfile(ctx context.Context, storeID string) (string, string, error)
}

type transferUsecase struct {
	accounts       domain.AccountRepository
	transactions   domain.TransactionRepository
	ledger         domain.LedgerRepository
	categories     domain.CategoryRepository
	authClient     AuthClient
	merchantClient MerchantClient
}

func NewTransferUsecase(
	accounts domain.AccountRepository,
	transactions domain.TransactionRepository,
	ledger domain.LedgerRepository,
	categories domain.CategoryRepository,
	authClient AuthClient,
	merchantClient MerchantClient,
) TransferUsecase {
	return &transferUsecase{
		accounts:       accounts,
		transactions:   transactions,
		ledger:         ledger,
		categories:     categories,
		authClient:     authClient,
		merchantClient: merchantClient,
	}
}
func (u *transferUsecase) Transfer(ctx context.Context, in TransferInput) (*domain.LedgerResult, error) {
	if in.AmountPaise <= 0 {
		return nil, apperrors.ErrInvalidInput
	}
	key := strings.TrimSpace(in.IdempotencyKey)
	if key == "" {
		return nil, apperrors.ErrInvalidInput
	}
	authUserID, err := uuid.Parse(in.AuthenticatedUserID)
	if err != nil {
		return nil, apperrors.ErrUnauthorized
	}

	fromAcc, err := u.accounts.GetWalletByUserID(ctx, authUserID)
	if err != nil {
		return nil, fmt.Errorf("get sender wallet: %w", err)
	}
	fromID := fromAcc.ID

	handle := strings.TrimSpace(in.ToHandle)
	if handle == "" {
		return nil, apperrors.ErrInvalidInput
	}

	toAcc, err := u.accounts.GetByHandle(ctx, handle)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return nil, apperrors.ErrInvalidInput
		}
		return nil, err
	}
	if toAcc.Type != domain.AccountTypeWallet && toAcc.Type != domain.AccountTypeMerchantSettlement {
    return nil, apperrors.ErrInvalidInput
}
	toID := toAcc.ID

	if fromID == toID {
		return nil, apperrors.ErrInvalidInput
	}

	existing, err := u.transactions.GetByIdempotencyKey(ctx, key, fromID)
	if err == nil && existing != nil {
		return u.replay(ctx, existing)
	}
	if err != nil && !errors.Is(err, apperrors.ErrNotFound) {
		return nil, err
	}

	var categoryID *uuid.UUID
	if in.CategoryID != nil && *in.CategoryID != "" {
		cid, err := uuid.Parse(*in.CategoryID)
		if err != nil {
			return nil, apperrors.ErrInvalidInput
		}
		cat, err := u.categories.GetByID(ctx, cid)
		if err != nil || cat.UserID != authUserID {
			return nil, apperrors.ErrInvalidInput
		}
		categoryID = &cid
	}

	result, err := u.ledger.ExecuteTransfer(ctx, &domain.Transaction{
		FromAccountID:  fromID,
		ToAccountID:    toID,
		Amount:         in.AmountPaise,
		Status:         domain.TxStatusPending,
		IdempotencyKey: key,
		Description:    in.Description,
		CategoryID:     categoryID,
	})
	if err != nil {
		if errors.Is(err, apperrors.ErrIdempotencyKeyUsed) {
			winner, getErr := u.transactions.GetByIdempotencyKey(ctx, key, fromID)
			if getErr != nil {
				return nil, getErr
			}
			return u.replay(ctx, winner)
		}
		return nil, err
	}
	return result, nil
}

func (u *transferUsecase) replay(ctx context.Context, t *domain.Transaction) (*domain.LedgerResult, error) {
	entries, err := u.transactions.GetEntriesByTransactionID(ctx, t.ID)
	if err != nil {
		return nil, err
	}
	result := &domain.LedgerResult{Transaction: t}
	for _, e := range entries {
		switch e.Direction {
		case "debit":
			result.DebitEntry = e
		case "credit":
			result.CreditEntry = e
		}
	}
	if from, err := u.accounts.GetByID(ctx, t.FromAccountID); err == nil {
		result.FromBalance = from.Balance
	}
	if to, err := u.accounts.GetByID(ctx, t.ToAccountID); err == nil {
		result.ToBalance = to.Balance
	}
	return result, nil
}

func (u *transferUsecase) GetByID(ctx context.Context, id string) (*domain.Transaction, error) {
	txID, err := uuid.Parse(id)
	if err != nil {
		return nil, apperrors.ErrInvalidInput
	}
	return u.transactions.GetByID(ctx, txID)
}

type ResolveResult struct {
	AccountID   string `json:"account_id"`
	Handle      string `json:"handle"`
	AccountType string `json:"account_type"`
	DisplayName string `json:"display_name"`
	PhotoURL    string `json:"photo_url"`
}

func (u *transferUsecase) ResolveHandle(ctx context.Context, handle string) (*ResolveResult, error) {
	handle = strings.TrimSpace(handle)
	if handle == "" {
		return nil, apperrors.ErrInvalidInput
	}

	acc, err := u.accounts.GetByHandle(ctx, handle)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return nil, apperrors.ErrInvalidInput
		}
		return nil, err
	}

	if acc.Type != domain.AccountTypeWallet && acc.Type != domain.AccountTypeMerchantSettlement {
		return nil, apperrors.ErrInvalidInput
	}

	handleVal := ""
	if acc.Handle != nil {
		handleVal = *acc.Handle
	}

	res := &ResolveResult{
		AccountID:   acc.ID.String(),
		Handle:      handleVal,
		AccountType: string(acc.Type),
	}

	if acc.Type == domain.AccountTypeWallet {
		if u.authClient != nil && acc.UserID != nil {
			profile, err := u.authClient.GetUserProfile(ctx, acc.UserID.String())
			if err == nil && profile != nil {
				res.DisplayName = profile.FullName
				res.PhotoURL = profile.ProfilePictureURL
			}
		}
	} else if acc.Type == domain.AccountTypeMerchantSettlement {
		if u.merchantClient != nil && acc.MerchantID != nil {
			name, logo, err := u.merchantClient.GetStoreProfile(ctx, acc.MerchantID.String())
			if err == nil {
				res.DisplayName = name
				res.PhotoURL = logo
			}
		}
	}

	return res, nil
}

type ParticipantProfile struct {
	UserID            string `json:"user_id"`
	AccountID         string `json:"account_id"`
	AccountType       string `json:"account_type"`
	Username          string `json:"username"`
	FullName          string `json:"full_name"`
	Handle            string `json:"handle"`
	Phone             string `json:"phone"`
	ProfilePicture    string `json:"profile_picture"`
	ProfilePictureURL string `json:"profile_picture_url"`
}

type TransactionDetail struct {
	PaymentID          string             `json:"payment_id"`
	ID                 string             `json:"id"`
	Amount             string             `json:"amount"`
	PaymentAmount      string             `json:"payment_amount"`
	Description        string             `json:"description"`
	PaymentDescription string             `json:"payment_description"`
	Category           string             `json:"category"`
	PaymentCategory    string             `json:"payment_category"`
	Status             string             `json:"status"`
	Direction          string             `json:"direction"` // "debit" (sent) or "credit" (received)
	CreatedAt          string             `json:"created_at"`
	From               ParticipantProfile `json:"from"`
	To                 ParticipantProfile `json:"to"`

	// Flat fields for backwards compatibility:
	FromUserID         string `json:"from_user_id"`
	ToUserID           string `json:"to_user_id"`
	FromAccountID      string `json:"from_account_id"`
	ToAccountID        string `json:"to_account_id"`
	FromUsername       string `json:"from_username"`
	FromHandle         string `json:"from_handle"`
	FromPhone          string `json:"from_phone"`
	FromProfilePicture string `json:"from_profile_picture"`
	ToUsername         string `json:"to_username"`
	ToHandle           string `json:"to_handle"`
	ToPhone            string `json:"to_phone"`
	ToProfilePicture   string `json:"to_profile_picture"`
}

type ListTransactionsInput struct {
	AuthenticatedUserID string
	CategoryID          *string
	From                *time.Time
	To                  *time.Time
	Page                int
	PageSize            int
}

type ListTransactionsResult struct {
	Transactions []*TransactionDetail
	TotalCount   int64
}

func (u *transferUsecase) ListMyTransactions(ctx context.Context, in ListTransactionsInput) (*ListTransactionsResult, error) {
	authUserID, err := uuid.Parse(in.AuthenticatedUserID)
	if err != nil {
		return nil, apperrors.ErrUnauthorized
	}
	acc, err := u.accounts.GetWalletByUserID(ctx, authUserID)
	if err != nil {
		return nil, err
	}
	if in.PageSize <= 0 || in.PageSize > 100 {
		in.PageSize = 20
	}
	if in.Page <= 0 {
		in.Page = 1
	}
	offset := (in.Page - 1) * in.PageSize

	if in.From != nil && in.To != nil && !in.To.After(*in.From) {
		return nil, apperrors.ErrInvalidInput
	}

	var categoryID *uuid.UUID
	if in.CategoryID != nil && strings.TrimSpace(*in.CategoryID) != "" {
		cid, err := uuid.Parse(strings.TrimSpace(*in.CategoryID))
		if err != nil {
			return nil, apperrors.ErrInvalidInput
		}
		categoryID = &cid
	}

	txs, total, err := u.transactions.ListByAccountPaginated(ctx, acc.ID, categoryID, in.From, in.To, int32(in.PageSize), int32(offset))
	if err != nil {
		return nil, err
	}

	accCache := make(map[uuid.UUID]*domain.Account)
	userCache := make(map[string]*authclient.UserProfile)
	merchantCache := make(map[string]struct{ name, logo string })
	catCache := make(map[uuid.UUID]string)

	getAccount := func(id uuid.UUID) *domain.Account {
		if a, ok := accCache[id]; ok {
			return a
		}
		a, err := u.accounts.GetByID(ctx, id)
		if err == nil && a != nil {
			accCache[id] = a
			return a
		}
		return nil
	}

	getUserProfile := func(userID string) *authclient.UserProfile {
		if p, ok := userCache[userID]; ok {
			return p
		}
		if u.authClient == nil {
			return nil
		}
		p, err := u.authClient.GetUserProfile(ctx, userID)
		if err == nil && p != nil {
			userCache[userID] = p
			return p
		}
		return nil
	}

	getMerchantProfile := func(merchantID string) (string, string) {
		if m, ok := merchantCache[merchantID]; ok {
			return m.name, m.logo
		}
		if u.merchantClient == nil {
			return "", ""
		}
		name, logo, err := u.merchantClient.GetStoreProfile(ctx, merchantID)
		if err == nil {
			merchantCache[merchantID] = struct{ name, logo string }{name: name, logo: logo}
			return name, logo
		}
		return "", ""
	}

	getCategoryName := func(catID uuid.UUID) string {
		if c, ok := catCache[catID]; ok {
			return c
		}
		cat, err := u.categories.GetByID(ctx, catID)
		if err == nil && cat != nil {
			catCache[catID] = cat.Name
			return cat.Name
		}
		return ""
	}

	buildParticipantProfile := func(acc *domain.Account, defaultAccountID uuid.UUID) ParticipantProfile {
		if acc == nil {
			return ParticipantProfile{
				AccountID: defaultAccountID.String(),
			}
		}

		profile := ParticipantProfile{
			AccountID:   acc.ID.String(),
			AccountType: string(acc.Type),
		}

		if acc.UserID != nil {
			profile.UserID = acc.UserID.String()
			if uProf := getUserProfile(profile.UserID); uProf != nil {
				profile.Username = uProf.FullName
				profile.FullName = uProf.FullName
				profile.Handle = uProf.Handle
				profile.Phone = uProf.Phone
				profile.ProfilePicture = uProf.ProfilePictureURL
				profile.ProfilePictureURL = uProf.ProfilePictureURL
			}
			if profile.Handle == "" && acc.Handle != nil {
				profile.Handle = *acc.Handle
			}
		} else if acc.MerchantID != nil {
			profile.UserID = acc.MerchantID.String()
			name, logo := getMerchantProfile(profile.UserID)
			profile.Username = name
			profile.FullName = name
			profile.ProfilePicture = logo
			profile.ProfilePictureURL = logo
			if acc.Handle != nil {
				profile.Handle = *acc.Handle
			}
		} else if acc.Type == domain.AccountTypeExternalSettlement {
			profile.Username = "External Settlement"
			profile.FullName = "External Settlement"
			profile.Handle = "system"
		} else {
			profile.Username = string(acc.Type)
			profile.FullName = string(acc.Type)
			profile.Handle = "system"
		}

		return profile
	}

	details := make([]*TransactionDetail, len(txs))
	for i, t := range txs {
		fromAcc := getAccount(t.FromAccountID)
		toAcc := getAccount(t.ToAccountID)

		fromProfile := buildParticipantProfile(fromAcc, t.FromAccountID)
		toProfile := buildParticipantProfile(toAcc, t.ToAccountID)

		direction := "credit"
		if fromAcc != nil && fromAcc.ID == acc.ID {
			direction = "debit"
		}

		categoryName := ""
		if t.CategoryID != nil {
			categoryName = getCategoryName(*t.CategoryID)
		}

		formattedAmount := money.FormatPaise(t.Amount)

		details[i] = &TransactionDetail{
			PaymentID:          t.ID.String(),
			ID:                 t.ID.String(),
			Amount:             formattedAmount,
			PaymentAmount:      formattedAmount,
			Description:        t.Description,
			PaymentDescription: t.Description,
			Category:           categoryName,
			PaymentCategory:    categoryName,
			Status:             string(t.Status),
			Direction:          direction,
			CreatedAt:          t.CreatedAt.Format(time.RFC3339),
			From:               fromProfile,
			To:                 toProfile,

			FromUserID:         fromProfile.UserID,
			ToUserID:           toProfile.UserID,
			FromAccountID:      fromProfile.AccountID,
			ToAccountID:        toProfile.AccountID,
			FromUsername:       fromProfile.Username,
			FromHandle:         fromProfile.Handle,
			FromPhone:          fromProfile.Phone,
			FromProfilePicture: fromProfile.ProfilePicture,
			ToUsername:         toProfile.Username,
			ToHandle:           toProfile.Handle,
			ToPhone:            toProfile.Phone,
			ToProfilePicture:   toProfile.ProfilePicture,
		}
	}

	return &ListTransactionsResult{Transactions: details, TotalCount: total}, nil
}