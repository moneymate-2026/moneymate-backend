package http

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/moneymate-2026/moneymate-backend/services/payment/internal/usecases"
	response "github.com/moneymate-2026/moneymate-backend/shared/pkg/responses"
)

type PodHandler struct {
	pods usecases.PodUsecase
}

func NewPodHandler(pods usecases.PodUsecase) *PodHandler {
	return &PodHandler{pods: pods}
}

type createPodRequest struct {
	Name              string  `json:"name" validate:"required,min=1,max=100"`
	TargetAmountPaise *int64 `json:"target_amount_paise" validate:"omitempty,gt=0"`
	TargetDate        *string `json:"target_date" validate:"omitempty"`
	Icon              string  `json:"icon" validate:"omitempty,max=50"`
}

func (h *PodHandler) Create(c fiber.Ctx) error {
	userID := userIDFromLocals(c)
	if userID == "" {
		return response.Unauthorized(c, "authentication required")
	}

	var req createPodRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, nil, "invalid request body")
	}

	if err := validate.Struct(req); err != nil {
		return response.BadRequest(c, nil, err.Error())
	}

	var targetDate *time.Time
	if req.TargetDate != nil && strings.TrimSpace(*req.TargetDate) != "" {
		parsedDate, err := usecases.ParseFromDate(strings.TrimSpace(*req.TargetDate))
		if err != nil {
			return response.BadRequest(c, nil, "invalid target_date format")
		}
		targetDate = &parsedDate
	}

	res, err := h.pods.CreatePod(c.Context(), usecases.CreatePodInput{
		AuthenticatedUserID: userID,
		Name:                req.Name,
		TargetAmountPaise:   req.TargetAmountPaise,
		TargetDate:          targetDate,
		Icon:                req.Icon,
	})
	if err != nil {
		return handleError(c, err)
	}

	return response.Created(c, "pod created", res)
}

func (h *PodHandler) List(c fiber.Ctx) error {
	userID := userIDFromLocals(c)
	if userID == "" {
		return response.Unauthorized(c, "authentication required")
	}

	pods, err := h.pods.ListPods(c.Context(), userID)
	if err != nil {
		return handleError(c, err)
	}

	return response.OK(c, "pods fetched", pods)
}

func (h *PodHandler) GetByID(c fiber.Ctx) error {
	userID := userIDFromLocals(c)
	if userID == "" {
		return response.Unauthorized(c, "authentication required")
	}

	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c, nil, "pod id is required")
	}

	pod, err := h.pods.GetPodByID(c.Context(), userID, id)
	if err != nil {
		return handleError(c, err)
	}

	return response.OK(c, "pod fetched", pod)
}

type podTransferRequest struct {
	AmountPaise    int64  `json:"amount_paise" validate:"required,gt=0"`
	Direction      string `json:"direction" validate:"required,oneof=deposit withdraw"`
	IdempotencyKey string `json:"idempotency_key" validate:"omitempty"`
}

func (h *PodHandler) Transfer(c fiber.Ctx) error {
	userID := userIDFromLocals(c)
	if userID == "" {
		return response.Unauthorized(c, "authentication required")
	}

	podID := c.Params("id")
	if podID == "" {
		return response.BadRequest(c, nil, "pod id is required")
	}

	var req podTransferRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, nil, "invalid request body")
	}

	if err := validate.Struct(req); err != nil {
		return response.BadRequest(c, nil, err.Error())
	}

	result, err := h.pods.TransferToPod(c.Context(), usecases.PodTransferInput{
		AuthenticatedUserID: userID,
		PodID:               podID,
		AmountPaise:         req.AmountPaise,
		Direction:           req.Direction,
		IdempotencyKey:      req.IdempotencyKey,
	})
	if err != nil {
		return handleError(c, err)
	}

	return response.OK(c, "transfer completed", result)
}

type updatePodRequest struct {
	Name              *string `json:"name" validate:"omitempty,min=1,max=100"`
	TargetAmountPaise *int64  `json:"target_amount_paise" validate:"omitempty,gt=0"`
	TargetDate        *string `json:"target_date" validate:"omitempty"`
	Icon              *string `json:"icon" validate:"omitempty,max=50"`
}

func (h *PodHandler) Update(c fiber.Ctx) error {
	userID := userIDFromLocals(c)
	if userID == "" {
		return response.Unauthorized(c, "authentication required")
	}

	podID := c.Params("id")
	if podID == "" {
		return response.BadRequest(c, nil, "pod id is required")
	}

	var req updatePodRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, nil, "invalid request body")
	}

	if err := validate.Struct(req); err != nil {
		return response.BadRequest(c, nil, err.Error())
	}

	var targetDate *time.Time
	if req.TargetDate != nil {
		trimmed := strings.TrimSpace(*req.TargetDate)
		if trimmed != "" {
			parsedDate, err := usecases.ParseFromDate(trimmed)
			if err != nil {
				return response.BadRequest(c, nil, "invalid target_date format")
			}
			targetDate = &parsedDate
		}
	}

	res, err := h.pods.UpdatePod(c.Context(), usecases.UpdatePodInput{
		AuthenticatedUserID: userID,
		PodID:               podID,
		Name:                req.Name,
		TargetAmountPaise:   req.TargetAmountPaise,
		TargetDate:          targetDate,
		Icon:                req.Icon,
	})
	if err != nil {
		return handleError(c, err)
	}

	return response.OK(c, "pod updated", res)
}

func (h *PodHandler) Delete(c fiber.Ctx) error {
	userID := userIDFromLocals(c)
	if userID == "" {
		return response.Unauthorized(c, "authentication required")
	}

	podID := c.Params("id")
	if podID == "" {
		return response.BadRequest(c, nil, "pod id is required")
	}

	if err := h.pods.DeletePod(c.Context(), userID, podID); err != nil {
		return handleError(c, err)
	}

	return response.OK(c, "pod deleted", nil)
}

