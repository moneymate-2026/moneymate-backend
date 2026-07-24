package http

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	sharedjwt "github.com/moneymate-2026/moneymate-backend/shared/pkg/jwt"
)

func RequireAuth(jwtSecret string) fiber.Handler {
	return func(c fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing authorization header"})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid authorization format"})
		}

		tokenString := parts[1]
		claims, err := sharedjwt.ParseAccessToken(tokenString, jwtSecret)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid or expired token"})
		}

		c.Locals("user_id", claims.Subject)
		return c.Next()
	}
}
