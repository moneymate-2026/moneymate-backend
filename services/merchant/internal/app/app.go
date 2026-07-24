package app

import (
	"context"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/moneymate-2026/moneymate-backend/services/merchant/config"
	"github.com/moneymate-2026/moneymate-backend/services/merchant/internal/adapter/postgres"
	"github.com/moneymate-2026/moneymate-backend/services/merchant/internal/adapter/postgres/repo"
	transporthttp "github.com/moneymate-2026/moneymate-backend/services/merchant/internal/transport/http"
	"github.com/moneymate-2026/moneymate-backend/services/merchant/internal/usecases"
)

type App struct {
	HTTPServer *fiber.App
	DB         *pgxpool.Pool
	Config     *config.Config
	HTTPAddr   string
}

func Build(cfg *config.Config) (*App, error) {
	ctx := context.Background()

	pool, err := postgres.ConnectDB(ctx, *cfg)
	if err != nil {
		return nil, fmt.Errorf("connect db: %w", err)
	}

	storeRepo := repo.NewStoreRepo(pool)
	storeUseCase := usecases.NewStoreUseCase(storeRepo)

	// HTTP Setup
	httpHandler := transporthttp.NewMerchantHandler(storeUseCase)
	httpServer := setupHTTPServer(cfg, httpHandler)

	httpAddr := cfg.Server.HTTPAddr
	if httpAddr == "" {
		httpAddr = "0.0.0.0:50053"
	}

	return &App{
		HTTPServer: httpServer,
		DB:         pool,
		Config:     cfg,
		HTTPAddr:   httpAddr,
	}, nil
}

func setupHTTPServer(cfg *config.Config, handler *transporthttp.MerchantHandler) *fiber.App {
	server := fiber.New(fiber.Config{
		AppName: "merchant-service",
	})

	server.Use(recover.New())
	server.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
	}))

	server.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "merchant"})
	})

	// No-op auth middleware for now, or use real one when JWT config is wired
	noopAuth := func(c fiber.Ctx) error { return c.Next() }
	transporthttp.RegisterRoutes(server, handler, noopAuth)

	return server
}

func (a *App) Run() error {
	// Start HTTP server
	log.Printf("Starting HTTP server on %s", a.HTTPAddr)
	return a.HTTPServer.Listen(a.HTTPAddr)
}

func (a *App) Close() {
	if a.HTTPServer != nil {
		a.HTTPServer.Shutdown()
	}
	if a.DB != nil {
		a.DB.Close()
	}
}
