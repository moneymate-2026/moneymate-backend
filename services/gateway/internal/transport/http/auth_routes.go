package http

import (
	"github.com/gofiber/fiber/v3"
	"github.com/moneymate-2026/moneymate-backend/gateway/internal/proxy"
)

func registerAuthRoutes(api fiber.Router, authAddr string, authMiddleware fiber.Handler) {
	userAuth := api.Group("/auth")
	userAuth.Get("/health", proxy.AuthProxy(authAddr, "/auth/health"))
	userAuth.Post("/register", proxy.AuthProxy(authAddr, "/auth/user/register"))
	userAuth.Post("/login", proxy.AuthProxy(authAddr, "/auth/login"))
	userAuth.Post("/admin/login", proxy.AuthProxy(authAddr, "/auth/admin/login"))
	userAuth.Post("/logout", authMiddleware, proxy.AuthProxy(authAddr, "/auth/logout"))
	userAuth.Post("/otp/send", proxy.AuthProxy(authAddr, "/auth/otp/send"))
	userAuth.Post("/otp/verify", proxy.AuthProxy(authAddr, "/auth/otp/verify"))
	userAuth.Post("/password/change", authMiddleware, proxy.AuthProxy(authAddr, "/auth/password/change"))
	userAuth.Post("/password/forgot", proxy.AuthProxy(authAddr, "/auth/password/forgot"))
	userAuth.Post("/password/reset", proxy.AuthProxy(authAddr, "/auth/password/reset"))
	userAuth.Post("/refresh", proxy.AuthProxy(authAddr, "/auth/refresh"))

	merchantAuth := api.Group("/merchant/auth")
	merchantAuth.Post("/register", proxy.AuthProxy(authAddr, "/auth/merchant/register"))
	merchantAuth.Post("/login", proxy.AuthProxy(authAddr, "/auth/login"))
	merchantAuth.Post("/logout", authMiddleware, proxy.AuthProxy(authAddr, "/auth/logout"))
	merchantAuth.Post("/refresh", proxy.AuthProxy(authAddr, "/auth/refresh"))

	users := api.Group("/users")
	users.Use(authMiddleware)
	users.Get("/me", proxy.AuthProxy(authAddr, "/users/me"))
	users.Get("/lookup", proxy.AuthProxy(authAddr, "/users/lookup"))
}

func registerPinRoutes(api fiber.Router, authAddr string, authMiddleware fiber.Handler) {
	userPin := api.Group("/pin")
	userPin.Use(authMiddleware)
	userPin.Post("/", proxy.AuthProxy(authAddr, "/user/pin"))
	userPin.Put("/", proxy.AuthProxy(authAddr, "/user/pin"))
	userPin.Post("/verify", proxy.AuthProxy(authAddr, "/user/pin/verify"))
}

func registerProfilePictureRoutes(api fiber.Router, authAddr string, authMiddleware fiber.Handler) {
	profile := api.Group("/profile")
	profile.Use(authMiddleware)

	profile.Get("/me", proxy.AuthProxy(authAddr, "/users/me/profile/"))
	profile.Post("/presign", proxy.AuthProxy(authAddr, "/users/me/profile/presign"))
	profile.Post("/", proxy.AuthProxy(authAddr, "/users/me/profile/"))
}