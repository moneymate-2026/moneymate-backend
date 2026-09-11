package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/moneymate-2026/moneymate-backend/services/payment/internal/domain"
	apperrors "github.com/moneymate-2026/moneymate-backend/shared/pkg/errors"
)

type mockTxManager struct {
	withTxCalls int
	shouldFail  bool
}

func (m *mockTxManager) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	m.withTxCalls++
	if m.shouldFail {
		return errors.New("tx failed")
	}
	return fn(ctx)
}

type mockPodRepo struct {
	pods       map[uuid.UUID]*domain.Pod
	createErr  error
	getErr     error
	listErr    error
	createdPod *domain.Pod
}

func (m *mockPodRepo) Create(ctx context.Context, pod *domain.Pod) error {
	if m.createErr != nil {
		return m.createErr
	}
	if pod.ID == uuid.Nil {
		pod.ID = uuid.New()
	}
	pod.CreatedAt = time.Now().UTC()
	pod.UpdatedAt = time.Now().UTC()
	if m.pods == nil {
		m.pods = make(map[uuid.UUID]*domain.Pod)
	}
	m.pods[pod.ID] = pod
	m.createdPod = pod
	return nil
}

func (m *mockPodRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Pod, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if p, ok := m.pods[id]; ok {
		return p, nil
	}
	return nil, apperrors.ErrNotFound
}

func (m *mockPodRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Pod, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var out []*domain.Pod
	for _, p := range m.pods {
		if p.UserID == userID {
			out = append(out, p)
		}
	}
	return out, nil
}

func (m *mockPodRepo) Update(ctx context.Context, pod *domain.Pod) error {
	if p, ok := m.pods[pod.ID]; ok {
		*p = *pod
		return nil
	}
	return apperrors.ErrNotFound
}

func (m *mockPodRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.pods, id)
	return nil
}

type mockPodLedgerRepo struct {
	transfers    []*domain.Transaction
	failTransfer error
	fromBal      int64
	toBal        int64
}

func (m *mockPodLedgerRepo) ExecuteTransfer(ctx context.Context, t *domain.Transaction) (*domain.LedgerResult, error) {
	if m.failTransfer != nil {
		return nil, m.failTransfer
	}
	m.transfers = append(m.transfers, t)
	t.ID = uuid.New()
	t.CreatedAt = time.Now().UTC()
	t.Status = domain.TxStatusCompleted
	return &domain.LedgerResult{
		Transaction: t,
		FromBalance: m.fromBal,
		ToBalance:   m.toBal,
	}, nil
}

type mockPodAccountRepo struct {
	domain.AccountRepository
	accounts     map[uuid.UUID]*domain.Account
	userAccounts map[uuid.UUID][]*domain.Account
	wallets      map[uuid.UUID]*domain.Account
	createErr    error
}

func (m *mockPodAccountRepo) Create(ctx context.Context, a *domain.Account) (*domain.Account, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	if m.accounts == nil {
		m.accounts = make(map[uuid.UUID]*domain.Account)
	}
	m.accounts[a.ID] = a
	if a.UserID != nil {
		if m.userAccounts == nil {
			m.userAccounts = make(map[uuid.UUID][]*domain.Account)
		}
		m.userAccounts[*a.UserID] = append(m.userAccounts[*a.UserID], a)
	}
	return a, nil
}

func (m *mockPodAccountRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Account, error) {
	if a, ok := m.accounts[id]; ok {
		return a, nil
	}
	return nil, apperrors.ErrNotFound
}

func (m *mockPodAccountRepo) GetWalletByUserID(ctx context.Context, userID uuid.UUID) (*domain.Account, error) {
	if w, ok := m.wallets[userID]; ok {
		return w, nil
	}
	return nil, apperrors.ErrNotFound
}

func (m *mockPodAccountRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Account, error) {
	if accs, ok := m.userAccounts[userID]; ok {
		return accs, nil
	}
	return nil, nil
}

func TestPodUsecase_CreatePod(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	setup := func() (*podUsecase, *mockPodAccountRepo, *mockPodRepo, *mockTxManager, *mockPodLedgerRepo) {
		accRepo := &mockPodAccountRepo{
			accounts:     make(map[uuid.UUID]*domain.Account),
			userAccounts: make(map[uuid.UUID][]*domain.Account),
			wallets:      make(map[uuid.UUID]*domain.Account),
		}
		podRepo := &mockPodRepo{
			pods: make(map[uuid.UUID]*domain.Pod),
		}
		txManager := &mockTxManager{}
		ledgerRepo := &mockPodLedgerRepo{}
		uc := NewPodUsecase(accRepo, podRepo, txManager, ledgerRepo).(*podUsecase)
		return uc, accRepo, podRepo, txManager, ledgerRepo
	}

	t.Run("success creating pod with target amount and date", func(t *testing.T) {
		uc, accRepo, podRepo, txManager, _ := setup()
		targetAmount := int64(100000) // 1000.00 INR
		targetDate := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)

		in := CreatePodInput{
			AuthenticatedUserID: userID.String(),
			Name:                "Emergency Fund",
			TargetAmountPaise:   &targetAmount,
			TargetDate:          &targetDate,
			Icon:                "shield",
		}

		res, err := uc.CreatePod(ctx, in)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if txManager.withTxCalls != 1 {
			t.Errorf("expected WithTx to be called once, got %d", txManager.withTxCalls)
		}
		if res.Name != "Emergency Fund" {
			t.Errorf("expected name Emergency Fund, got %s", res.Name)
		}
		if res.BalancePaise != 0 || res.Balance != "0.00" {
			t.Errorf("expected initial balance 0, got %d / %s", res.BalancePaise, res.Balance)
		}
		if res.TargetAmount == nil || *res.TargetAmount != "1000.00" {
			t.Errorf("expected target amount 1000.00, got %v", res.TargetAmount)
		}
		if res.TargetDate == nil || *res.TargetDate != "2026-12-31" {
			t.Errorf("expected target date 2026-12-31, got %v", res.TargetDate)
		}

		accID, _ := uuid.Parse(res.AccountID)
		acc, ok := accRepo.accounts[accID]
		if !ok {
			t.Fatalf("expected account %v to exist in account repo", accID)
		}
		if acc.Type != domain.AccountTypePod {
			t.Errorf("expected account type pod, got %v", acc.Type)
		}
		if acc.UserID == nil || *acc.UserID != userID {
			t.Errorf("expected user ID %v, got %v", userID, acc.UserID)
		}

		podID, _ := uuid.Parse(res.ID)
		if _, ok := podRepo.pods[podID]; !ok {
			t.Errorf("expected pod %v to exist in pod repo", podID)
		}
	})

	t.Run("empty name returns ErrInvalidInput", func(t *testing.T) {
		uc, _, _, _, _ := setup()
		_, err := uc.CreatePod(ctx, CreatePodInput{
			AuthenticatedUserID: userID.String(),
			Name:                "   ",
		})
		if err != apperrors.ErrInvalidInput {
			t.Fatalf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("invalid target amount returns ErrInvalidInput", func(t *testing.T) {
		uc, _, _, _, _ := setup()
		invalidAmount := int64(-500)
		_, err := uc.CreatePod(ctx, CreatePodInput{
			AuthenticatedUserID: userID.String(),
			Name:                "Vacation",
			TargetAmountPaise:   &invalidAmount,
		})
		if err != apperrors.ErrInvalidInput {
			t.Fatalf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("transaction failure rolls back and returns error", func(t *testing.T) {
		uc, _, _, txManager, _ := setup()
		txManager.shouldFail = true

		_, err := uc.CreatePod(ctx, CreatePodInput{
			AuthenticatedUserID: userID.String(),
			Name:                "Car",
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestPodUsecase_GetAndListPods(t *testing.T) {
	ctx := context.Background()
	userA := uuid.New()
	userB := uuid.New()

	accID1 := uuid.New()
	accID2 := uuid.New()

	acc1 := &domain.Account{
		ID:       accID1,
		UserID:   &userA,
		Type:     domain.AccountTypePod,
		Currency: "INR",
		Balance:  50000, // 500.00
	}
	acc2 := &domain.Account{
		ID:       accID2,
		UserID:   &userB,
		Type:     domain.AccountTypePod,
		Currency: "INR",
		Balance:  20000,
	}

	accRepo := &mockPodAccountRepo{
		accounts: map[uuid.UUID]*domain.Account{
			accID1: acc1,
			accID2: acc2,
		},
		userAccounts: map[uuid.UUID][]*domain.Account{
			userA: {acc1},
			userB: {acc2},
		},
	}

	pod1 := &domain.Pod{
		ID:        uuid.New(),
		AccountID: accID1,
		UserID:    userA,
		Name:      "User A Pod",
		Status:    "active",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	pod2 := &domain.Pod{
		ID:        uuid.New(),
		AccountID: accID2,
		UserID:    userB,
		Name:      "User B Pod",
		Status:    "active",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	podRepo := &mockPodRepo{
		pods: map[uuid.UUID]*domain.Pod{
			pod1.ID: pod1,
			pod2.ID: pod2,
		},
	}

	uc := NewPodUsecase(accRepo, podRepo, &mockTxManager{}, &mockPodLedgerRepo{})

	t.Run("GetPodByID success for owner", func(t *testing.T) {
		res, err := uc.GetPodByID(ctx, userA.String(), pod1.ID.String())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if res.ID != pod1.ID.String() {
			t.Errorf("expected pod ID %v, got %v", pod1.ID, res.ID)
		}
		if res.Balance != "500.00" || res.BalancePaise != 50000 {
			t.Errorf("expected balance 500.00 (50000 paise), got %s (%d)", res.Balance, res.BalancePaise)
		}
	})

	t.Run("GetPodByID returns 404 (ErrNotFound) for pod belonging to different user", func(t *testing.T) {
		_, err := uc.GetPodByID(ctx, userA.String(), pod2.ID.String())
		if err != apperrors.ErrNotFound {
			t.Fatalf("expected ErrNotFound (404), got %v", err)
		}
	})

	t.Run("GetPodByID returns 404 for non-existent pod", func(t *testing.T) {
		_, err := uc.GetPodByID(ctx, userA.String(), uuid.New().String())
		if err != apperrors.ErrNotFound {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("ListPods returns only user's own pods", func(t *testing.T) {
		res, err := uc.ListPods(ctx, userA.String())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(res) != 1 {
			t.Fatalf("expected 1 pod for user A, got %d", len(res))
		}
		if res[0].ID != pod1.ID.String() {
			t.Errorf("expected pod1, got %v", res[0].ID)
		}
		if res[0].Balance != "500.00" {
			t.Errorf("expected balance 500.00, got %s", res[0].Balance)
		}
	})
}

func TestPodUsecase_TransferToPod(t *testing.T) {
	ctx := context.Background()
	userA := uuid.New()
	userB := uuid.New()

	walletIDA := uuid.New()
	podAccIDA := uuid.New()

	walletA := &domain.Account{
		ID:       walletIDA,
		UserID:   &userA,
		Type:     domain.AccountTypeWallet,
		Currency: "INR",
		Balance:  100000, // 1000.00
	}
	podAccA := &domain.Account{
		ID:       podAccIDA,
		UserID:   &userA,
		Type:     domain.AccountTypePod,
		Currency: "INR",
		Balance:  25000, // 250.00
	}

	podA := &domain.Pod{
		ID:        uuid.New(),
		AccountID: podAccIDA,
		UserID:    userA,
		Name:      "Savings",
		Status:    "active",
	}

	podB := &domain.Pod{
		ID:        uuid.New(),
		AccountID: uuid.New(),
		UserID:    userB,
		Name:      "User B Pod",
		Status:    "active",
	}

	accRepo := &mockPodAccountRepo{
		accounts: map[uuid.UUID]*domain.Account{
			walletIDA: walletA,
			podAccIDA: podAccA,
		},
		wallets: map[uuid.UUID]*domain.Account{
			userA: walletA,
		},
	}
	podRepo := &mockPodRepo{
		pods: map[uuid.UUID]*domain.Pod{
			podA.ID: podA,
			podB.ID: podB,
		},
	}

	t.Run("Deposit to pod transfers wallet -> pod", func(t *testing.T) {
		ledgerRepo := &mockPodLedgerRepo{
			fromBal: 80000, // wallet after 200.00 deposit
			toBal:   45000, // pod after 200.00 deposit
		}
		uc := NewPodUsecase(accRepo, podRepo, &mockTxManager{}, ledgerRepo)

		res, err := uc.TransferToPod(ctx, PodTransferInput{
			AuthenticatedUserID: userA.String(),
			PodID:               podA.ID.String(),
			AmountPaise:         20000,
			Direction:           "deposit",
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(ledgerRepo.transfers) != 1 {
			t.Fatalf("expected 1 transfer, got %d", len(ledgerRepo.transfers))
		}
		tx := ledgerRepo.transfers[0]
		if tx.FromAccountID != walletIDA || tx.ToAccountID != podAccIDA {
			t.Errorf("expected transfer from %v to %v, got from %v to %v", walletIDA, podAccIDA, tx.FromAccountID, tx.ToAccountID)
		}
		if tx.Amount != 20000 {
			t.Errorf("expected amount 20000, got %d", tx.Amount)
		}
		if res.WalletBalance != "800.00" || res.PodBalance != "450.00" {
			t.Errorf("expected wallet 800.00 and pod 450.00, got wallet %s, pod %s", res.WalletBalance, res.PodBalance)
		}
	})

	t.Run("Withdraw from pod transfers pod -> wallet", func(t *testing.T) {
		ledgerRepo := &mockPodLedgerRepo{
			fromBal: 15000,  // pod after 100.00 withdraw
			toBal:   110000, // wallet after 100.00 withdraw
		}
		uc := NewPodUsecase(accRepo, podRepo, &mockTxManager{}, ledgerRepo)

		res, err := uc.TransferToPod(ctx, PodTransferInput{
			AuthenticatedUserID: userA.String(),
			PodID:               podA.ID.String(),
			AmountPaise:         10000,
			Direction:           "withdraw",
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(ledgerRepo.transfers) != 1 {
			t.Fatalf("expected 1 transfer, got %d", len(ledgerRepo.transfers))
		}
		tx := ledgerRepo.transfers[0]
		if tx.FromAccountID != podAccIDA || tx.ToAccountID != walletIDA {
			t.Errorf("expected transfer from %v to %v, got from %v to %v", podAccIDA, walletIDA, tx.FromAccountID, tx.ToAccountID)
		}
		if res.PodBalance != "150.00" || res.WalletBalance != "1100.00" {
			t.Errorf("expected pod 150.00 and wallet 1100.00, got pod %s, wallet %s", res.PodBalance, res.WalletBalance)
		}
	})

	t.Run("Withdraw fails cleanly on insufficient funds from ledger", func(t *testing.T) {
		ledgerRepo := &mockPodLedgerRepo{
			failTransfer: apperrors.ErrInsufficientFunds,
		}
		uc := NewPodUsecase(accRepo, podRepo, &mockTxManager{}, ledgerRepo)

		_, err := uc.TransferToPod(ctx, PodTransferInput{
			AuthenticatedUserID: userA.String(),
			PodID:               podA.ID.String(),
			AmountPaise:         9999999,
			Direction:           "withdraw",
		})
		if !errors.Is(err, apperrors.ErrInsufficientFunds) {
			t.Fatalf("expected ErrInsufficientFunds, got %v", err)
		}
	})

	t.Run("Transfer into/out of pod belonging to different user is rejected with authorization error", func(t *testing.T) {
		ledgerRepo := &mockPodLedgerRepo{}
		uc := NewPodUsecase(accRepo, podRepo, &mockTxManager{}, ledgerRepo)

		// User A tries to transfer to User B's pod
		_, err := uc.TransferToPod(ctx, PodTransferInput{
			AuthenticatedUserID: userA.String(),
			PodID:               podB.ID.String(),
			AmountPaise:         5000,
			Direction:           "deposit",
		})
		if !errors.Is(err, apperrors.ErrForbidden) {
			t.Fatalf("expected ErrForbidden, got %v", err)
		}
		if len(ledgerRepo.transfers) != 0 {
			t.Errorf("ledger transfer should not be called when authorization fails")
		}
	})

	t.Run("Invalid direction returns ErrInvalidInput", func(t *testing.T) {
		ledgerRepo := &mockPodLedgerRepo{}
		uc := NewPodUsecase(accRepo, podRepo, &mockTxManager{}, ledgerRepo)

		_, err := uc.TransferToPod(ctx, PodTransferInput{
			AuthenticatedUserID: userA.String(),
			PodID:               podA.ID.String(),
			AmountPaise:         5000,
			Direction:           "unknown",
		})
		if !errors.Is(err, apperrors.ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput, got %v", err)
		}
	})
}

func TestPodUsecase_UpdatePod(t *testing.T) {
	ctx := context.Background()
	userA := uuid.New()
	userB := uuid.New()

	podAccID := uuid.New()
	podID := uuid.New()
	initialTarget := int64(100000)
	podA := &domain.Pod{
		ID:           podID,
		AccountID:    podAccID,
		UserID:       userA,
		Name:         "Trip",
		TargetAmount: &initialTarget,
		Status:       "active",
		CreatedAt:    time.Now().UTC(),
	}

	podRepo := &mockPodRepo{
		pods: map[uuid.UUID]*domain.Pod{
			podID: podA,
		},
	}
	accRepo := &mockPodAccountRepo{
		accounts: map[uuid.UUID]*domain.Account{
			podAccID: {ID: podAccID, Balance: 25000},
		},
	}
	uc := NewPodUsecase(accRepo, podRepo, &mockTxManager{}, &mockPodLedgerRepo{})

	t.Run("success updating name, target amount, target date, icon", func(t *testing.T) {
		newName := "Summer Vacation"
		newTarget := int64(200000)
		newTargetDate := time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC)
		newIcon := "beach"

		updated, err := uc.UpdatePod(ctx, UpdatePodInput{
			AuthenticatedUserID: userA.String(),
			PodID:               podID.String(),
			Name:                &newName,
			TargetAmountPaise:   &newTarget,
			TargetDate:          &newTargetDate,
			Icon:                &newIcon,
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if updated.Name != newName {
			t.Errorf("expected name %q, got %q", newName, updated.Name)
		}
		if updated.TargetAmountPaise == nil || *updated.TargetAmountPaise != newTarget {
			t.Errorf("expected target amount %d, got %v", newTarget, updated.TargetAmountPaise)
		}
		if updated.Icon != newIcon {
			t.Errorf("expected icon %q, got %q", newIcon, updated.Icon)
		}
		if updated.TargetDate == nil || *updated.TargetDate != "2027-06-01" {
			t.Errorf("expected target date '2027-06-01', got %v", updated.TargetDate)
		}
		if updated.BalancePaise != 25000 {
			t.Errorf("expected balance 25000, got %d", updated.BalancePaise)
		}
	})

	t.Run("unauthorized user returns ErrNotFound (404)", func(t *testing.T) {
		newName := "Hacked Pod"
		_, err := uc.UpdatePod(ctx, UpdatePodInput{
			AuthenticatedUserID: userB.String(),
			PodID:               podID.String(),
			Name:                &newName,
		})
		if !errors.Is(err, apperrors.ErrNotFound) {
			t.Fatalf("expected ErrNotFound for non-owner, got %v", err)
		}
	})

	t.Run("non-existent pod returns ErrNotFound", func(t *testing.T) {
		newName := "New Name"
		_, err := uc.UpdatePod(ctx, UpdatePodInput{
			AuthenticatedUserID: userA.String(),
			PodID:               uuid.New().String(),
			Name:                &newName,
		})
		if !errors.Is(err, apperrors.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("empty name returns ErrInvalidInput", func(t *testing.T) {
		emptyName := "   "
		_, err := uc.UpdatePod(ctx, UpdatePodInput{
			AuthenticatedUserID: userA.String(),
			PodID:               podID.String(),
			Name:                &emptyName,
		})
		if !errors.Is(err, apperrors.ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("invalid target amount returns ErrInvalidInput", func(t *testing.T) {
		negativeTarget := int64(-500)
		_, err := uc.UpdatePod(ctx, UpdatePodInput{
			AuthenticatedUserID: userA.String(),
			PodID:               podID.String(),
			TargetAmountPaise:   &negativeTarget,
		})
		if !errors.Is(err, apperrors.ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput, got %v", err)
		}
	})
}

func TestPodUsecase_DeletePod(t *testing.T) {
	ctx := context.Background()
	userA := uuid.New()
	userB := uuid.New()

	podAccID := uuid.New()
	podID := uuid.New()
	podA := &domain.Pod{
		ID:        podID,
		AccountID: podAccID,
		UserID:    userA,
		Name:      "Emergency Fund",
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	}

	podRepo := &mockPodRepo{
		pods: map[uuid.UUID]*domain.Pod{
			podID: podA,
		},
	}
	podAccount := &domain.Account{ID: podAccID, UserID: &userA, Type: "pod", Balance: 50000}
	accRepo := &mockPodAccountRepo{
		accounts: map[uuid.UUID]*domain.Account{
			podAccID: podAccount,
		},
	}
	uc := NewPodUsecase(accRepo, podRepo, &mockTxManager{}, &mockPodLedgerRepo{})

	t.Run("attempt to delete pod with non-zero balance is rejected", func(t *testing.T) {
		err := uc.DeletePod(ctx, userA.String(), podID.String())
		if !errors.Is(err, apperrors.ErrPodNonZeroBalance) {
			t.Fatalf("expected ErrPodNonZeroBalance, got %v", err)
		}
		// Confirm pod still exists in repo
		if _, exists := podRepo.pods[podID]; !exists {
			t.Fatalf("pod should not have been deleted from repo")
		}
	})

	t.Run("attempt to delete pod belonging to different user returns ErrNotFound (404)", func(t *testing.T) {
		err := uc.DeletePod(ctx, userB.String(), podID.String())
		if !errors.Is(err, apperrors.ErrNotFound) {
			t.Fatalf("expected ErrNotFound for non-owner, got %v", err)
		}
	})

	t.Run("withdraw to zero, then delete successfully", func(t *testing.T) {
		// Set balance to zero
		podAccount.Balance = 0

		err := uc.DeletePod(ctx, userA.String(), podID.String())
		if err != nil {
			t.Fatalf("expected no error when deleting pod with 0 balance, got %v", err)
		}

		// Confirm pod no longer exists in repository
		if _, exists := podRepo.pods[podID]; exists {
			t.Fatalf("pod should have been deleted from repo")
		}

		// Confirm GetPodByID returns 404
		_, err = uc.GetPodByID(ctx, userA.String(), podID.String())
		if !errors.Is(err, apperrors.ErrNotFound) {
			t.Fatalf("expected ErrNotFound on GetPodByID, got %v", err)
		}

		// Confirm ListPods no longer returns it
		list, err := uc.ListPods(ctx, userA.String())
		if err != nil {
			t.Fatalf("expected no error listing pods, got %v", err)
		}
		if len(list) != 0 {
			t.Fatalf("expected 0 pods in list, got %d", len(list))
		}
	})
}

