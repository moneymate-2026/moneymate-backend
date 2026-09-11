package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/moneymate-2026/moneymate-backend/services/payment/internal/usecases"
	apperrors "github.com/moneymate-2026/moneymate-backend/shared/pkg/errors"
)

type mockPodUsecase struct {
	createdPod     *usecases.PodDetail
	createErr      error
	pods           []*usecases.PodDetail
	listErr        error
	getPod         *usecases.PodDetail
	getErr         error
	transferResult *usecases.PodTransferResult
	transferErr    error
	updatedPod     *usecases.PodDetail
	updateErr      error
	deleteErr      error
}

func (m *mockPodUsecase) CreatePod(ctx context.Context, in usecases.CreatePodInput) (*usecases.PodDetail, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	if m.createdPod != nil {
		return m.createdPod, nil
	}
	return &usecases.PodDetail{
		ID:           uuid.New().String(),
		AccountID:    uuid.New().String(),
		UserID:       in.AuthenticatedUserID,
		Name:         in.Name,
		Status:       "active",
		BalancePaise: 0,
		Balance:      "0.00",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}, nil
}

func (m *mockPodUsecase) ListPods(ctx context.Context, authUserID string) ([]*usecases.PodDetail, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.pods, nil
}

func (m *mockPodUsecase) GetPodByID(ctx context.Context, authUserID, podID string) (*usecases.PodDetail, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.getPod, nil
}

func (m *mockPodUsecase) TransferToPod(ctx context.Context, in usecases.PodTransferInput) (*usecases.PodTransferResult, error) {
	if m.transferErr != nil {
		return nil, m.transferErr
	}
	if m.transferResult != nil {
		return m.transferResult, nil
	}
	return &usecases.PodTransferResult{
		TransactionID:      uuid.New().String(),
		PodID:              in.PodID,
		Direction:          in.Direction,
		AmountPaise:        in.AmountPaise,
		Amount:             "100.00",
		WalletBalancePaise: 90000,
		WalletBalance:      "900.00",
		PodBalancePaise:    10000,
		PodBalance:         "100.00",
		Status:             "completed",
		CreatedAt:          time.Now().UTC(),
	}, nil
}

func (m *mockPodUsecase) UpdatePod(ctx context.Context, in usecases.UpdatePodInput) (*usecases.PodDetail, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	if m.updatedPod != nil {
		return m.updatedPod, nil
	}
	name := "Default Pod"
	if in.Name != nil {
		name = *in.Name
	}
	return &usecases.PodDetail{
		ID:           in.PodID,
		AccountID:    uuid.New().String(),
		UserID:       in.AuthenticatedUserID,
		Name:         name,
		Status:       "active",
		BalancePaise: 0,
		Balance:      "0.00",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}, nil
}

func (m *mockPodUsecase) DeletePod(ctx context.Context, authUserID, podID string) error {
	return m.deleteErr
}

func TestPodHandler_Endpoints(t *testing.T) {
	userID := uuid.New().String()

	setupApp := func(uc *mockPodUsecase) *fiber.App {
		app := fiber.New()
		app.Use(func(c fiber.Ctx) error {
			c.Locals("userID", userID)
			return c.Next()
		})
		handler := NewPodHandler(uc)
		app.Post("/payment/pods", handler.Create)
		app.Get("/payment/pods", handler.List)
		app.Get("/payment/pods/:id", handler.GetByID)
		app.Patch("/payment/pods/:id", handler.Update)
		app.Delete("/payment/pods/:id", handler.Delete)
		app.Post("/payment/pods/:id/transfer", handler.Transfer)
		return app
	}

	t.Run("POST /payment/pods creates pod successfully", func(t *testing.T) {
		mockUC := &mockPodUsecase{}
		app := setupApp(mockUC)

		body := map[string]any{
			"name":                "Holiday Savings",
			"target_amount_paise": 50000,
			"target_date":         "2026-12-25",
			"icon":                "tree",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/payment/pods", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusCreated {
			t.Fatalf("expected status 201 Created, got %d", resp.StatusCode)
		}

		var res struct {
			Success bool               `json:"success"`
			Data    usecases.PodDetail `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if res.Data.Name != "Holiday Savings" {
			t.Errorf("expected name Holiday Savings, got %s", res.Data.Name)
		}
		if res.Data.Balance != "0.00" {
			t.Errorf("expected balance 0.00, got %s", res.Data.Balance)
		}
	})

	t.Run("POST /payment/pods returns 400 on missing name", func(t *testing.T) {
		mockUC := &mockPodUsecase{}
		app := setupApp(mockUC)

		body := map[string]any{
			"name": "",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/payment/pods", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Fatalf("expected status 400 Bad Request, got %d", resp.StatusCode)
		}
	})

	t.Run("GET /payment/pods lists pods", func(t *testing.T) {
		mockUC := &mockPodUsecase{
			pods: []*usecases.PodDetail{
				{
					ID:           uuid.New().String(),
					Name:         "Gadget Fund",
					Balance:      "150.00",
					BalancePaise: 15000,
					Status:       "active",
				},
			},
		}
		app := setupApp(mockUC)

		req := httptest.NewRequest("GET", "/payment/pods", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected status 200 OK, got %d", resp.StatusCode)
		}

		var res struct {
			Success bool                 `json:"success"`
			Data    []usecases.PodDetail `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(res.Data) != 1 || res.Data[0].Name != "Gadget Fund" {
			t.Errorf("expected 1 pod with name Gadget Fund, got %v", res.Data)
		}
	})

	t.Run("GET /payment/pods/:id returns pod for owner", func(t *testing.T) {
		podID := uuid.New().String()
		mockUC := &mockPodUsecase{
			getPod: &usecases.PodDetail{
				ID:           podID,
				Name:         "Bike",
				Balance:      "300.00",
				BalancePaise: 30000,
				Status:       "active",
			},
		}
		app := setupApp(mockUC)

		req := httptest.NewRequest("GET", "/payment/pods/"+podID, nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected status 200 OK, got %d", resp.StatusCode)
		}
	})

	t.Run("GET /payment/pods/:id returns 404 (NotFound) when pod belongs to different user", func(t *testing.T) {
		podID := uuid.New().String()
		mockUC := &mockPodUsecase{
			getErr: apperrors.ErrNotFound,
		}
		app := setupApp(mockUC)

		req := httptest.NewRequest("GET", "/payment/pods/"+podID, nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusNotFound {
			t.Fatalf("expected status 404 Not Found, got %d", resp.StatusCode)
		}
	})

	t.Run("POST /payment/pods/:id/transfer deposit success", func(t *testing.T) {
		podID := uuid.New().String()
		mockUC := &mockPodUsecase{
			transferResult: &usecases.PodTransferResult{
				TransactionID:      uuid.New().String(),
				PodID:              podID,
				Direction:          "deposit",
				AmountPaise:        20000,
				Amount:             "200.00",
				WalletBalancePaise: 80000,
				WalletBalance:      "800.00",
				PodBalancePaise:    45000,
				PodBalance:         "450.00",
				Status:             "completed",
				CreatedAt:          time.Now().UTC(),
			},
		}
		app := setupApp(mockUC)

		body := map[string]any{
			"amount_paise": 20000,
			"direction":    "deposit",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/payment/pods/"+podID+"/transfer", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected status 200 OK, got %d", resp.StatusCode)
		}

		var res struct {
			Success bool                       `json:"success"`
			Data    usecases.PodTransferResult `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if res.Data.Direction != "deposit" || res.Data.Amount != "200.00" {
			t.Errorf("unexpected transfer response: %+v", res.Data)
		}
	})

	t.Run("POST /payment/pods/:id/transfer returns 403 on unauthorized user", func(t *testing.T) {
		podID := uuid.New().String()
		mockUC := &mockPodUsecase{
			transferErr: apperrors.ErrForbidden,
		}
		app := setupApp(mockUC)

		body := map[string]any{
			"amount_paise": 5000,
			"direction":    "deposit",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/payment/pods/"+podID+"/transfer", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusForbidden {
			t.Fatalf("expected status 403 Forbidden, got %d", resp.StatusCode)
		}
	})

	t.Run("POST /payment/pods/:id/transfer returns 409 on insufficient funds", func(t *testing.T) {
		podID := uuid.New().String()
		mockUC := &mockPodUsecase{
			transferErr: apperrors.ErrInsufficientFunds,
		}
		app := setupApp(mockUC)

		body := map[string]any{
			"amount_paise": 9999999,
			"direction":    "withdraw",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/payment/pods/"+podID+"/transfer", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusConflict {
			t.Fatalf("expected status 409 Conflict, got %d", resp.StatusCode)
		}
	})

	t.Run("PATCH /payment/pods/:id updates pod successfully", func(t *testing.T) {
		podID := uuid.New().String()
		mockUC := &mockPodUsecase{
			updatedPod: &usecases.PodDetail{
				ID:        podID,
				Name:      "New Holiday",
				Icon:      "plane",
				Status:    "active",
				Balance:   "0.00",
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			},
		}
		app := setupApp(mockUC)

		body := map[string]any{
			"name": "New Holiday",
			"icon": "plane",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("PATCH", "/payment/pods/"+podID, bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected status 200 OK, got %d", resp.StatusCode)
		}
	})

	t.Run("PATCH /payment/pods/:id returns 404 on non-owner", func(t *testing.T) {
		podID := uuid.New().String()
		mockUC := &mockPodUsecase{
			updateErr: apperrors.ErrNotFound,
		}
		app := setupApp(mockUC)

		body := map[string]any{
			"name": "Hacked Pod",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("PATCH", "/payment/pods/"+podID, bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusNotFound {
			t.Fatalf("expected status 404 Not Found, got %d", resp.StatusCode)
		}
	})

	t.Run("DELETE /payment/pods/:id returns 409 when balance is non-zero", func(t *testing.T) {
		podID := uuid.New().String()
		mockUC := &mockPodUsecase{
			deleteErr: apperrors.ErrPodNonZeroBalance,
		}
		app := setupApp(mockUC)

		req := httptest.NewRequest("DELETE", "/payment/pods/"+podID, nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusConflict {
			t.Fatalf("expected status 409 Conflict, got %d", resp.StatusCode)
		}
	})

	t.Run("DELETE /payment/pods/:id returns 404 on non-owner", func(t *testing.T) {
		podID := uuid.New().String()
		mockUC := &mockPodUsecase{
			deleteErr: apperrors.ErrNotFound,
		}
		app := setupApp(mockUC)

		req := httptest.NewRequest("DELETE", "/payment/pods/"+podID, nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusNotFound {
			t.Fatalf("expected status 404 Not Found, got %d", resp.StatusCode)
		}
	})

	t.Run("DELETE /payment/pods/:id succeeds when balance is zero", func(t *testing.T) {
		podID := uuid.New().String()
		mockUC := &mockPodUsecase{
			deleteErr: nil,
		}
		app := setupApp(mockUC)

		req := httptest.NewRequest("DELETE", "/payment/pods/"+podID, nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected status 200 OK, got %d", resp.StatusCode)
		}
	})
}

