package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/moneymate-2026/moneymate-backend/auth/internal/domain"
)

func RegisterRoutes(router fiber.Router, h *AuthHandler, authMiddleware fiber.Handler) {
	auth := router.Group("/auth")
	
	//auth endpoints 
    auth.Post("/login", h.Login)
    auth.Post("/logout", authMiddleware, h.Logout)
    auth.Post("/otp/send", h.SendRegistrationOTP)
    auth.Post("/otp/verify", h.VerifyRegistrationOTP)
    
    // Specific Registration Routes
    auth.Post("/user/register", h.Register(domain.AccountTypeUser))
    auth.Post("/merchant/register", h.Register(domain.AccountTypeMerchant))

    // Internal endpoints
    internal := router.Group("/internal")
    internal.Post("/auth/verify-access-token", h.VerifyAccessToken)
    internal.Post("/auth/verify-transaction-token", h.VerifyTransactionToken)
    internal.Get("/auth/users/:id", h.GetUserByID)
}
