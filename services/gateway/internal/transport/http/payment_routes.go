package http

import (
	"github.com/gofiber/fiber/v3"
	"github.com/moneymate-2026/moneymate-backend/gateway/internal/proxy"
)

func registerPaymentRoutes(api fiber.Router, authMiddleware fiber.Handler, registry *proxy.ServiceRegistry) {
	payment := api.Group("/payment")
	payment.Use(authMiddleware)

	payment.Get("/wallets/me", proxy.HTTPProxy(registry, "payment", "/payment/wallets/me"))
	payment.Get("/wallets/:id", proxy.HTTPProxy(registry, "payment", "/payment/wallets/:id"))

	payment.Get("/resolve", proxy.HTTPProxy(registry, "payment", "/payment/resolve"))

	payment.Post("/transfers", proxy.HTTPProxy(registry, "payment", "/payment/transfers"))
	payment.Get("/transactions/:id", proxy.HTTPProxy(registry, "payment", "/payment/transactions/:id"))
	payment.Get("/transactions/me", proxy.HTTPProxy(registry, "payment", "/payment/transactions/me"))

	payment.Get("/analytics/spend-by-category", proxy.HTTPProxy(registry, "payment", "/payment/analytics/spend-by-category"))
	payment.Get("/analytics/spend-by-period", proxy.HTTPProxy(registry, "payment", "/payment/analytics/spend-by-period"))

	payment.Post("/categories", proxy.HTTPProxy(registry, "payment", "/payment/categories"))
	payment.Get("/categories", proxy.HTTPProxy(registry, "payment", "/payment/categories"))
	payment.Put("/categories/:id", proxy.HTTPProxy(registry, "payment", "/payment/categories/:id"))
	payment.Delete("/categories/:id", proxy.HTTPProxy(registry, "payment", "/payment/categories/:id"))

	payment.Post("/deposits", proxy.HTTPProxy(registry, "payment", "/payment/deposits"))
	payment.Post("/deposits/confirm", proxy.HTTPProxy(registry, "payment", "/payment/deposits/confirm"))
	payment.Get("/deposits", proxy.HTTPProxy(registry, "payment", "/payment/deposits"))

	payment.Post("/withdrawals", proxy.HTTPProxy(registry, "payment", "/payment/withdrawals"))
	payment.Get("/withdrawals", proxy.HTTPProxy(registry, "payment", "/payment/withdrawals"))

	payment.Post("/pods", proxy.HTTPProxy(registry, "payment", "/payment/pods"))
	payment.Get("/pods", proxy.HTTPProxy(registry, "payment", "/payment/pods"))
	payment.Get("/pods/:id", proxy.HTTPProxy(registry, "payment", "/payment/pods/:id"))
	payment.Patch("/pods/:id", proxy.HTTPProxy(registry, "payment", "/payment/pods/:id"))
	payment.Delete("/pods/:id", proxy.HTTPProxy(registry, "payment", "/payment/pods/:id"))
	payment.Post("/pods/:id/transfer", proxy.HTTPProxy(registry, "payment", "/payment/pods/:id/transfer"))
}
