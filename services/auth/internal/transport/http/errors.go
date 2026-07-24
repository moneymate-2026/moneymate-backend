package http

import (
	"errors"
	"fmt"
	"log"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"

	apperrors "github.com/moneymate-2026/moneymate-backend/shared/pkg/errors"
	response "github.com/moneymate-2026/moneymate-backend/shared/pkg/responses"
)

func handleError(c fiber.Ctx, err error) error {
	appErr := apperrors.ParseError(err)
	log.Printf("[ERROR] %s %s | %d %s: %v", c.Method(), c.Path(), appErr.StatusCode, appErr.Code, appErr.Err)
	return response.Response(c, appErr.StatusCode, appErr.Message, appErr.Details, false)
}


func formatValidationErrors(err error) map[string]string {
	fieldErrors := make(map[string]string)
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		for _, fe := range ve {
			fieldErrors[fe.Field()] = validationMessage(fe)
		}
	}
	return fieldErrors
}

func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "this field is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return fmt.Sprintf("must be at least %s characters", fe.Param())
	case "max":
		return fmt.Sprintf("must be at most %s characters", fe.Param())
	case "len":
		return fmt.Sprintf("must be exactly %s characters", fe.Param())
	case "numeric":
		return "must contain only digits"
	case "e164":
		return "must be a valid phone number, e.g. +14155552671"
	default:
		return "invalid value"
	}
}