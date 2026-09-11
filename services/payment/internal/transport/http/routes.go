package http

import (
	"github.com/gofiber/fiber/v3"

	authclient "github.com/moneymate-2026/moneymate-backend/services/payment/internal/adapter/authClient"
	merchantclient "github.com/moneymate-2026/moneymate-backend/services/payment/internal/adapter/merchantClient"
	sharedjwt "github.com/moneymate-2026/moneymate-backend/shared/pkg/jwt"
)

func RegisterRoutes(router fiber.Router, wh *WalletHandler, th *TransferHandler, sth *SystemTransferHandler, dh *DepositHandler, wdh *WithdrawalHandler, ch *CategoryHandler, ah *AnalyticsHandler, ph *PodHandler, jwtCfg sharedjwt.Config, authClient *authclient.Client, merchantClient *merchantclient.Client, internalSecret string) {
	pay := router.Group("/payment", RequireUserID(jwtCfg))

	pay.Get("/wallets/me", RequireTransactionToken(authClient), wh.GetMyWallet)
	pay.Get("/wallets/:id", wh.GetWalletByID)
	

	pay.Post("/transfers", RequireTransactionToken(authClient), th.Transfer)
	pay.Get("/transactions/me", th.ListMyTransactions)
	pay.Get("/transactions/:id", th.GetTransaction)
	pay.Get("/resolve", th.ResolveHandle)

	pay.Get("/analytics/spend-by-category", ah.SpendByCategory)
	pay.Get("/analytics/spend-by-period", ah.SpendByPeriod)

	pay.Post("/categories", ch.Create)
	pay.Get("/categories", ch.List)
	pay.Put("/categories/:id", ch.Update)
	pay.Delete("/categories/:id", ch.Delete)

	pay.Post("/deposits", RequireTransactionToken(authClient), dh.Initiate)
	pay.Post("/deposits/confirm", dh.Confirm)
	pay.Get("/deposits", dh.List)

	pay.Post("/withdrawals", RequireTransactionToken(authClient), wdh.Request)
	pay.Get("/withdrawals", wdh.List)

	pay.Post("/pods", ph.Create)
	pay.Get("/pods", ph.List)
	pay.Get("/pods/:id", ph.GetByID)
	pay.Patch("/pods/:id", ph.Update)
	pay.Delete("/pods/:id", ph.Delete)
	pay.Post("/pods/:id/transfer", ph.Transfer)

	internal := router.Group("/internal", RequireInternalSecret(internalSecret))
	internal.Post("/payment/wallets", wh.CreateWalletInternal)
	internal.Post("/payment/system-transfer", sth.Transfer)
}
