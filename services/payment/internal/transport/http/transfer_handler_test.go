package http

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/moneymate-2026/moneymate-backend/services/payment/internal/domain"
	"github.com/moneymate-2026/moneymate-backend/services/payment/internal/usecases"
)

type mockTransferUsecase struct {
	capturedInput usecases.ListTransactionsInput
	listRes       *usecases.ListTransactionsResult
	err           error
}

func (m *mockTransferUsecase) Transfer(ctx context.Context, in usecases.TransferInput) (*domain.LedgerResult, error) {
	return nil, nil
}

func (m *mockTransferUsecase) GetByID(ctx context.Context, id string) (*domain.Transaction, error) {
	return nil, nil
}

func (m *mockTransferUsecase) ResolveHandle(ctx context.Context, handle string) (*usecases.ResolveResult, error) {
	return nil, nil
}

func (m *mockTransferUsecase) ListMyTransactions(ctx context.Context, in usecases.ListTransactionsInput) (*usecases.ListTransactionsResult, error) {
	m.capturedInput = in
	if m.err != nil {
		return nil, m.err
	}
	if m.listRes != nil {
		return m.listRes, nil
	}
	return &usecases.ListTransactionsResult{
		Transactions: []*usecases.TransactionDetail{},
		TotalCount:   0,
	}, nil
}

func TestListMyTransactions_DateFiltering(t *testing.T) {
	userID := uuid.New().String()

	setupApp := func(uc *mockTransferUsecase) *fiber.App {
		app := fiber.New()
		// Auth middleware mock
		app.Use(func(c fiber.Ctx) error {
			c.Locals("userID", userID)
			return c.Next()
		})
		handler := NewTransferHandler(uc)
		app.Get("/transactions/me", handler.ListMyTransactions)
		return app
	}

	t.Run("success with single day range YYYY-MM-DD", func(t *testing.T) {
		mockUC := &mockTransferUsecase{}
		app := setupApp(mockUC)

		req := httptest.NewRequest("GET", "/transactions/me?from=2026-09-03&to=2026-09-03", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}

		expectedFrom := time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)
		expectedTo := time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC) // midnight of next day

		if mockUC.capturedInput.From == nil || !mockUC.capturedInput.From.Equal(expectedFrom) {
			t.Errorf("expected from %v, got %v", expectedFrom, mockUC.capturedInput.From)
		}
		if mockUC.capturedInput.To == nil || !mockUC.capturedInput.To.Equal(expectedTo) {
			t.Errorf("expected to %v, got %v", expectedTo, mockUC.capturedInput.To)
		}
	})

	t.Run("success with no from/to (all transactions)", func(t *testing.T) {
		mockUC := &mockTransferUsecase{}
		app := setupApp(mockUC)

		req := httptest.NewRequest("GET", "/transactions/me", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}

		if mockUC.capturedInput.From != nil {
			t.Errorf("expected nil from, got %v", mockUC.capturedInput.From)
		}
		if mockUC.capturedInput.To != nil {
			t.Errorf("expected nil to, got %v", mockUC.capturedInput.To)
		}
	})

	t.Run("success with combined from/to and category_id", func(t *testing.T) {
		mockUC := &mockTransferUsecase{}
		app := setupApp(mockUC)

		catID := uuid.New().String()
		req := httptest.NewRequest("GET", "/transactions/me?from=2026-09-01&to=2026-09-05&category_id="+catID, nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}

		expectedFrom := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
		expectedTo := time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC)

		if mockUC.capturedInput.CategoryID == nil || *mockUC.capturedInput.CategoryID != catID {
			t.Errorf("expected category_id %s, got %v", catID, mockUC.capturedInput.CategoryID)
		}
		if mockUC.capturedInput.From == nil || !mockUC.capturedInput.From.Equal(expectedFrom) {
			t.Errorf("expected from %v, got %v", expectedFrom, mockUC.capturedInput.From)
		}
		if mockUC.capturedInput.To == nil || !mockUC.capturedInput.To.Equal(expectedTo) {
			t.Errorf("expected to %v, got %v", expectedTo, mockUC.capturedInput.To)
		}
	})

	t.Run("success with RFC3339 timestamps", func(t *testing.T) {
		mockUC := &mockTransferUsecase{}
		app := setupApp(mockUC)

		fromStr := "2026-09-03T10:00:00Z"
		toStr := "2026-09-03T18:00:00Z"
		req := httptest.NewRequest("GET", "/transactions/me?from="+fromStr+"&to="+toStr, nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}

		expectedFrom, _ := time.Parse(time.RFC3339, fromStr)
		expectedTo, _ := time.Parse(time.RFC3339, toStr)

		if mockUC.capturedInput.From == nil || !mockUC.capturedInput.From.Equal(expectedFrom) {
			t.Errorf("expected from %v, got %v", expectedFrom, mockUC.capturedInput.From)
		}
		if mockUC.capturedInput.To == nil || !mockUC.capturedInput.To.Equal(expectedTo) {
			t.Errorf("expected to %v, got %v", expectedTo, mockUC.capturedInput.To)
		}
	})

	t.Run("error on invalid from date format", func(t *testing.T) {
		mockUC := &mockTransferUsecase{}
		app := setupApp(mockUC)

		req := httptest.NewRequest("GET", "/transactions/me?from=invalid-date", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("error on invalid to date format", func(t *testing.T) {
		mockUC := &mockTransferUsecase{}
		app := setupApp(mockUC)

		req := httptest.NewRequest("GET", "/transactions/me?to=invalid-date", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("error when to is before from (RFC3339)", func(t *testing.T) {
		mockUC := &mockTransferUsecase{}
		app := setupApp(mockUC)

		fromStr := "2026-09-03T18:00:00Z"
		toStr := "2026-09-03T10:00:00Z"
		req := httptest.NewRequest("GET", "/transactions/me?from="+fromStr+"&to="+toStr, nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("returns transactions and pagination in response", func(t *testing.T) {
		txID := uuid.New().String()
		mockUC := &mockTransferUsecase{
			listRes: &usecases.ListTransactionsResult{
				Transactions: []*usecases.TransactionDetail{
					{
						ID:       txID,
						Amount:   "50.00",
						Status:   "completed",
						Category: "Food",
					},
				},
				TotalCount: 1,
			},
		}
		app := setupApp(mockUC)

		req := httptest.NewRequest("GET", "/transactions/me?page=1&page_size=10", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}

		var body struct {
			Success bool `json:"success"`
			Data    struct {
				Transactions []struct {
					ID       string `json:"id"`
					Category string `json:"category"`
				} `json:"transactions"`
				TotalCount int64 `json:"total_count"`
				Page       int   `json:"page"`
				PageSize   int   `json:"page_size"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if body.Data.TotalCount != 1 || len(body.Data.Transactions) != 1 {
			t.Fatalf("expected 1 transaction, got count %d, len %d", body.Data.TotalCount, len(body.Data.Transactions))
		}
		if body.Data.Transactions[0].ID != txID {
			t.Errorf("expected tx ID %s, got %s", txID, body.Data.Transactions[0].ID)
		}
	})
}
