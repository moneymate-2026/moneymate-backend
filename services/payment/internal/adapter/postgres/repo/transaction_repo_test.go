package repo_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/moneymate-2026/moneymate-backend/services/payment/internal/adapter/postgres/repo"
	"github.com/moneymate-2026/moneymate-backend/services/payment/internal/domain"
)

func TestTransactionRepo_SpendAnalyticsDebitOnly(t *testing.T) {
	candidateDSNs := []string{
		os.Getenv("DATABASE_URL"),
		"postgres://superuser:somekindofsupersecretpassword@localhost:5433/moneymate?sslmode=disable&search_path=payment",
		"postgres://payment_user:payment_password@localhost:5433/moneymate?sslmode=disable&search_path=payment",
		"postgres://superuser:somekindofsupersecretpassword@localhost:5432/moneymate?sslmode=disable&search_path=payment",
		"postgres://payment_user:payment_password@localhost:5432/moneymate?sslmode=disable&search_path=payment",
	}

	var pool *pgxpool.Pool
	var connectedDSN string
	for _, dsn := range candidateDSNs {
		if dsn == "" {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		p, err := pgxpool.New(ctx, dsn)
		if err == nil {
			if err := p.Ping(ctx); err == nil {
				pool = p
				connectedDSN = dsn
				cancel()
				break
			}
			p.Close()
		}
		cancel()
	}

	if pool == nil {
		t.Skip("skipping live database test: unable to connect to Postgres on candidates")
		return
	}
	defer pool.Close()
	t.Logf("Connected to database at %s", connectedDSN)

	ctx := context.Background()

	txRepo := repo.NewTransactionRepo(pool)
	accountRepo := repo.NewAccountRepo(pool)
	categoryRepo := repo.NewCategoryRepo(pool)

	userA := uuid.New()
	userB := uuid.New()

	accA, err := accountRepo.CreateWallet(ctx, &domain.Account{
		UserID:   &userA,
		Currency: "INR",
	})
	if err != nil {
		t.Fatalf("failed to create accA: %v", err)
	}
	accB, err := accountRepo.CreateWallet(ctx, &domain.Account{
		UserID:   &userB,
		Currency: "INR",
	})
	if err != nil {
		t.Fatalf("failed to create accB: %v", err)
	}

	foodCat, err := categoryRepo.Create(ctx, userA, "Food-"+uuid.New().String()[:8])
	if err != nil {
		t.Fatalf("failed to create food category: %v", err)
	}

	travelCat, err := categoryRepo.Create(ctx, userA, "Travel-"+uuid.New().String()[:8])
	if err != nil {
		t.Fatalf("failed to create travel category: %v", err)
	}

	now := time.Now().UTC()
	yesterday := now.Add(-24 * time.Hour)
	tomorrow := now.Add(24 * time.Hour)

	// 1. Completed Debit from A -> B: 5000 paise (Food)
	err = txRepo.Create(ctx, &domain.Transaction{
		ID:             uuid.New(),
		FromAccountID:  accA.ID,
		ToAccountID:    accB.ID,
		Amount:         5000,
		Status:         domain.TxStatusCompleted,
		IdempotencyKey: uuid.New().String(),
		Description:    "Dinner",
		CategoryID:     &foodCat.ID,
		CompletedAt:    &now,
	})
	if err != nil {
		t.Fatalf("failed to create tx1: %v", err)
	}

	// 2. Completed Debit from A -> B: 2500 paise (Food)
	err = txRepo.Create(ctx, &domain.Transaction{
		ID:             uuid.New(),
		FromAccountID:  accA.ID,
		ToAccountID:    accB.ID,
		Amount:         2500,
		Status:         domain.TxStatusCompleted,
		IdempotencyKey: uuid.New().String(),
		Description:    "Snack",
		CategoryID:     &foodCat.ID,
		CompletedAt:    &now,
	})
	if err != nil {
		t.Fatalf("failed to create tx2: %v", err)
	}

	// 3. Completed Debit from A -> B: 10000 paise (Travel)
	err = txRepo.Create(ctx, &domain.Transaction{
		ID:             uuid.New(),
		FromAccountID:  accA.ID,
		ToAccountID:    accB.ID,
		Amount:         10000,
		Status:         domain.TxStatusCompleted,
		IdempotencyKey: uuid.New().String(),
		Description:    "Flight",
		CategoryID:     &travelCat.ID,
		CompletedAt:    &now,
	})
	if err != nil {
		t.Fatalf("failed to create tx3: %v", err)
	}

	// 4. Completed Credit TO A from B: 20000 paise (Food) - incoming to A, NOT debit!
	err = txRepo.Create(ctx, &domain.Transaction{
		ID:             uuid.New(),
		FromAccountID:  accB.ID,
		ToAccountID:    accA.ID,
		Amount:         20000,
		Status:         domain.TxStatusCompleted,
		IdempotencyKey: uuid.New().String(),
		Description:    "Reimbursement to A",
		CategoryID:     &foodCat.ID,
		CompletedAt:    &now,
	})
	if err != nil {
		t.Fatalf("failed to create incoming tx: %v", err)
	}

	// 5. Pending Debit from A -> B: 50000 paise (Food) - NOT completed!
	err = txRepo.Create(ctx, &domain.Transaction{
		ID:             uuid.New(),
		FromAccountID:  accA.ID,
		ToAccountID:    accB.ID,
		Amount:         50000,
		Status:         domain.TxStatusPending,
		IdempotencyKey: uuid.New().String(),
		Description:    "Pending bill",
		CategoryID:     &foodCat.ID,
	})
	if err != nil {
		t.Fatalf("failed to create pending tx: %v", err)
	}

	// Test Spend by Category with explicit from
	catSpend, err := txRepo.GetSpendByCategory(ctx, accA.ID, &yesterday, tomorrow)
	if err != nil {
		t.Fatalf("GetSpendByCategory failed: %v", err)
	}

	var travelTotal, foodTotal int64
	var travelCount, foodCount int64
	for _, c := range catSpend {
		if c.Category == travelCat.Name {
			travelTotal = c.TotalAmount
			travelCount = c.TransactionCount
		}
		if c.Category == foodCat.Name {
			foodTotal = c.TotalAmount
			foodCount = c.TransactionCount
		}
	}

	if travelTotal != 10000 || travelCount != 1 {
		t.Errorf("expected travel spend 10000 and count 1, got total %d, count %d", travelTotal, travelCount)
	}
	// Food total must be exactly 5000 + 2500 = 7500. The 20000 credit and 50000 pending must NOT be included!
	if foodTotal != 7500 || foodCount != 2 {
		t.Errorf("expected food spend 7500 and count 2, got total %d, count %d (credits/pending leaked into aggregation!)", foodTotal, foodCount)
	}

	// Test Spend by Category with all-time (nil from)
	catSpendAllTime, err := txRepo.GetSpendByCategory(ctx, accA.ID, nil, tomorrow)
	if err != nil {
		t.Fatalf("GetSpendByCategory (all time) failed: %v", err)
	}
	if len(catSpendAllTime) != 2 {
		t.Errorf("expected 2 categories in all-time spend, got %d", len(catSpendAllTime))
	}

	// Test Spend by Period with DATE_TRUNC parameterization: day, week, month
	for _, gran := range []string{"day", "week", "month"} {
		periodSpend, err := txRepo.GetSpendByPeriod(ctx, accA.ID, &yesterday, tomorrow, gran)
		if err != nil {
			t.Fatalf("GetSpendByPeriod with granularity %q failed: %v", gran, err)
		}

		var totalPeriodAmount int64
		var totalPeriodCount int64
		for _, p := range periodSpend {
			totalPeriodAmount += p.TotalAmount
			totalPeriodCount += p.TransactionCount
		}

		if totalPeriodAmount != 17500 { // 5000 + 2500 + 10000
			t.Errorf("granularity %q: expected total debit 17500, got %d", gran, totalPeriodAmount)
		}
		if totalPeriodCount != 3 {
			t.Errorf("granularity %q: expected total count 3, got %d", gran, totalPeriodCount)
		}

		// Test Spend by Period with all-time (nil from)
		periodSpendAllTime, err := txRepo.GetSpendByPeriod(ctx, accA.ID, nil, tomorrow, gran)
		if err != nil {
			t.Fatalf("GetSpendByPeriod all-time with granularity %q failed: %v", gran, err)
		}
		var totalAllTimeAmount int64
		for _, p := range periodSpendAllTime {
			totalAllTimeAmount += p.TotalAmount
		}
		if totalAllTimeAmount != 17500 {
			t.Errorf("granularity %q (all-time): expected total debit 17500, got %d", gran, totalAllTimeAmount)
		}
	}

	// Test ListByAccountPaginated with date range and category filtering
	t.Run("ListByAccountPaginated date range and category", func(t *testing.T) {
		// All time (nil from and to)
		allTxs, totalCount, err := txRepo.ListByAccountPaginated(ctx, accA.ID, nil, nil, nil, 10, 0)
		if err != nil {
			t.Fatalf("ListByAccountPaginated failed: %v", err)
		}
		if totalCount != 5 || len(allTxs) != 5 {
			t.Errorf("expected 5 transactions, got %d (total: %d)", len(allTxs), totalCount)
		}

		// Filter by Food category and date range [yesterday, tomorrow)
		foodTxs, foodCount, err := txRepo.ListByAccountPaginated(ctx, accA.ID, &foodCat.ID, &yesterday, &tomorrow, 10, 0)
		if err != nil {
			t.Fatalf("ListByAccountPaginated with date and category failed: %v", err)
		}
		if foodCount != 4 || len(foodTxs) != 4 { // 2 completed debit + 1 completed credit + 1 pending debit
			t.Errorf("expected 4 food transactions, got %d (total: %d)", len(foodTxs), foodCount)
		}

		// Filter by future date (should return 0)
		future := tomorrow.Add(24 * time.Hour)
		futureTxs, futureCount, err := txRepo.ListByAccountPaginated(ctx, accA.ID, nil, &future, nil, 10, 0)
		if err != nil {
			t.Fatalf("ListByAccountPaginated with future date failed: %v", err)
		}
		if futureCount != 0 || len(futureTxs) != 0 {
			t.Errorf("expected 0 transactions for future date, got %d (total: %d)", len(futureTxs), futureCount)
		}
	})
}
