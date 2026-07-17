package http

import (
	"github.com/gofiber/fiber/v3"
	response "github.com/moneymate-2026/moneymate-backend/shared/pkg/responses"
)

func RequireUserID(c fiber.Ctx) error {
	userID := c.Get("X-User-Id")
	if userID == "" {
		return response.Unauthorized(c, "missing authentication context")
	}
	c.Locals("userID", userID)
	return c.Next()
}