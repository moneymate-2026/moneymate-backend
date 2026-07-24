package http

import (
	"github.com/gofiber/fiber/v3"
)

func RegisterRoutes(router fiber.Router, h *MerchantHandler, authMiddleware fiber.Handler) {
	merchant := router.Group("/merchant")

	// Apply auth middleware if provided
	if authMiddleware != nil {
		merchant.Use(authMiddleware)
	}

	merchant.Post("/register", h.RegisterStore)
	merchant.Get("/status/:owner_id", h.GetStore)
	merchant.Get("/pending", h.GetPendingStores)
}
