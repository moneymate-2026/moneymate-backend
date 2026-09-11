package http

import (
	"errors"

	"github.com/gofiber/fiber/v3"

	apperrors "github.com/moneymate-2026/moneymate-backend/shared/pkg/errors"
	response "github.com/moneymate-2026/moneymate-backend/shared/pkg/responses"
)

func handleError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		return response.NotFound(c, "not found")
	case errors.Is(err, apperrors.ErrInsufficientFunds):
		return response.Conflict(c, "insufficient funds")
	case errors.Is(err, apperrors.ErrPodNonZeroBalance):
		return response.Conflict(c, "cannot delete pod with non-zero balance; withdraw all funds first")
	case errors.Is(err, apperrors.ErrIdempotencyKeyUsed):
		return response.Conflict(c, "this transaction has already been processed")
	case errors.Is(err, apperrors.ErrInvalidInput):
		return response.BadRequest(c, nil, "invalid input")
	case errors.Is(err, apperrors.ErrForbidden):
		return response.Forbidden(c, nil, "you do not have access to this resource")
	case errors.Is(err, apperrors.ErrUnauthorized):
		return response.Unauthorized(c, "authentication required")
	case errors.Is(err, apperrors.ErrTransactionLocked):
		return response.Conflict(c, "transaction locked, please retry")
	default:
		return response.InternalServerError(c)
	}
}
