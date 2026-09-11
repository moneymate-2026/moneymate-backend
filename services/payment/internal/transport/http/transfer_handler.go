package http

import (
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"

	"github.com/moneymate-2026/moneymate-backend/services/payment/internal/domain"
	"github.com/moneymate-2026/moneymate-backend/services/payment/internal/usecases"
	apperrors "github.com/moneymate-2026/moneymate-backend/shared/pkg/errors"
	"github.com/moneymate-2026/moneymate-backend/shared/pkg/money"
	response "github.com/moneymate-2026/moneymate-backend/shared/pkg/responses"
)

var validate = validator.New()

type TransferHandler struct {
	transfers usecases.TransferUsecase
}

func NewTransferHandler(transfers usecases.TransferUsecase) *TransferHandler {
	return &TransferHandler{transfers: transfers}
}

func (h *TransferHandler) Transfer(c fiber.Ctx) error {
	userID := userIDFromLocals(c)
	if userID == "" {
		return response.Unauthorized(c, "authentication required")
	}

	var req transferRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, nil, "invalid request body")
	}
	if err := validate.Struct(req); err != nil {
		return response.BadRequest(c, nil, "validation failed")
	}

	amountPaise, err := money.ParseRupees(req.Amount)
	if err != nil {
		return response.BadRequest(c, nil, "invalid amount")
	}

	result, err := h.transfers.Transfer(c.Context(), usecases.TransferInput{
		AuthenticatedUserID: userID,
		ToHandle:            req.ToHandle,
		AmountPaise:         amountPaise,
		IdempotencyKey:      req.IdempotencyKey,
		Description:         req.Description,
		CategoryID:          req.CategoryID,
	})
	if err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "transfer completed", transferResponse{
		Transaction: toTransactionResponse(result.Transaction),
		FromBalance: money.FormatPaise(result.FromBalance),
		ToBalance:   money.FormatPaise(result.ToBalance),
	})
}

func (h *TransferHandler) GetTransaction(c fiber.Ctx) error {
	tx, err := h.transfers.GetByID(c.Context(), c.Params("id"))
	if err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "transaction found", toTransactionResponse(tx))
}

func (h *TransferHandler) ResolveHandle(c fiber.Ctx) error {
	handle := c.Query("handle")
	if handle == "" {
		return response.BadRequest(c, nil, "handle is required")
	}

	res, err := h.transfers.ResolveHandle(c.Context(), handle)
	if err != nil {
		return handleError(c, err)
	}

	return response.OK(c, "handle resolved", res)
}

func toTransactionResponse(t *domain.Transaction) transactionResponse {
	return transactionResponse{
		ID:             t.ID.String(),
		FromAccountID:  t.FromAccountID.String(),
		ToAccountID:    t.ToAccountID.String(),
		Amount:         money.FormatPaise(t.Amount),
		Status:         string(t.Status),
		IdempotencyKey: t.IdempotencyKey,
		Description:    t.Description,
		CreatedAt:      t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func (h *TransferHandler) ListMyTransactions(c fiber.Ctx) error {
	userID := userIDFromLocals(c)
	if userID == "" {
		return response.Unauthorized(c, "authentication required")
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))

	var categoryID *string
	if rawCat := strings.TrimSpace(c.Query("category_id")); rawCat != "" {
		categoryID = &rawCat
	}

	var from, to *time.Time
	if rawFrom := strings.TrimSpace(c.Query("from")); rawFrom != "" {
		parsedFrom, err := usecases.ParseFromDate(rawFrom)
		if err != nil {
			return handleError(c, apperrors.ErrInvalidInput)
		}
		from = &parsedFrom
	}

	if rawTo := strings.TrimSpace(c.Query("to")); rawTo != "" {
		parsedTo, err := usecases.ParseToDate(rawTo)
		if err != nil {
			return handleError(c, apperrors.ErrInvalidInput)
		}
		to = &parsedTo
	}

	if from != nil && to != nil && !to.After(*from) {
		return handleError(c, apperrors.ErrInvalidInput)
	}

	result, err := h.transfers.ListMyTransactions(c.Context(), usecases.ListTransactionsInput{
		AuthenticatedUserID: userID,
		CategoryID:          categoryID,
		From:                from,
		To:                  to,
		Page:                page,
		PageSize:            pageSize,
	})
	if err != nil {
		return handleError(c, err)
	}

	return response.OK(c, "transactions fetched", fiber.Map{
		"transactions": result.Transactions,
		"total_count":  result.TotalCount,
		"page":         page,
		"page_size":    pageSize,
	})
}