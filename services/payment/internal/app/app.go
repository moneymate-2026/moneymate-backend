package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/moneymate-2026/moneymate-backend/services/payment/config"
	authclient "github.com/moneymate-2026/moneymate-backend/services/payment/internal/adapter/authClient"
	merchantclient "github.com/moneymate-2026/moneymate-backend/services/payment/internal/adapter/merchantClient"
	"github.com/moneymate-2026/moneymate-backend/services/payment/internal/adapter/postgres"
	"github.com/moneymate-2026/moneymate-backend/services/payment/internal/adapter/postgres/repo"
	transporthttp "github.com/moneymate-2026/moneymate-backend/services/payment/internal/transport/http"
	"github.com/moneymate-2026/moneymate-backend/services/payment/internal/usecases"
	sharedjwt "github.com/moneymate-2026/moneymate-backend/shared/pkg/jwt"
	"github.com/moneymate-2026/moneymate-backend/shared/pkg/kafka"
	"github.com/moneymate-2026/moneymate-backend/shared/pkg/payment"
	"github.com/moneymate-2026/moneymate-backend/shared/pkg/pgxtx"
)

type App struct {
	HTTPServer    *fiber.App
	DB            *pgxpool.Pool
	HTTPAddr      string
	KafkaConsumer *kafka.Consumer
	WalletUC      usecases.WalletUsecase
}

func Build(cfg *config.Config) (*App, error) {
	ctx := context.Background()

	pool, err := postgres.ConnectDB(ctx, *cfg)
	if err != nil {
		return nil, fmt.Errorf("connect db: %w", err)
	}

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s&search_path=payment",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
		cfg.Database.SslMode,
	)
	if err := postgres.RunMigrations(dsn, cfg.Database.MigrationsPath); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	accountRepo := repo.NewAccountRepo(pool)
	transactionRepo := repo.NewTransactionRepo(pool)
	ledgerRepo := repo.NewLedgerRepo(pool)
	depositRepo := repo.NewDepositRepo(pool)
	categoryRepo := repo.NewCategoryRepo(pool)
	podRepo := repo.NewPodRepo(pool)
	txManager := pgxtx.New(pool)

	externalSettlementID, err := seedExternalSettlementAccount(ctx, accountRepo)
	if err != nil {
		return nil, fmt.Errorf("seed external settlement account: %w", err)
	}

	razorpayClient := payment.NewRazorpayClient(cfg.Razorpay.KeyID, cfg.Razorpay.KeySecret)

	authClient := authclient.New(cfg.AuthServiceURL, cfg.InternalServiceSecret)
	merchantClient := merchantclient.New(cfg.MerchantServiceURL, cfg.InternalServiceSecret)

	walletUC := usecases.NewWalletUsecase(accountRepo)
	transferUC := usecases.NewTransferUsecase(accountRepo, transactionRepo, ledgerRepo, categoryRepo, authClient, merchantClient)
	depositUC := usecases.NewDepositUsecase(depositRepo, accountRepo, razorpayClient, cfg.Razorpay.KeyID, externalSettlementID)
	withdrawalUC := usecases.NewWithdrawalUsecase(accountRepo, transactionRepo, ledgerRepo, externalSettlementID)
	categoryUC:= usecases.NewCategoryUsecase(categoryRepo)
	systemTransferUC := usecases.NewSystemTransferUsecase(ledgerRepo)
	analyticsUC := usecases.NewAnalyticsUsecase(accountRepo, transactionRepo)
	podUC := usecases.NewPodUsecase(accountRepo, podRepo, txManager, ledgerRepo)
	
	walletHandler := transporthttp.NewWalletHandler(walletUC)
	transferHandler := transporthttp.NewTransferHandler(transferUC)
	depositHandler := transporthttp.NewDepositHandler(depositUC, razorpayClient)
	withdrawalHandler := transporthttp.NewWithdrawalHandler(withdrawalUC)
	cateGoryhandler:=transporthttp.NewCategoryHandler(categoryUC)
	systemTransferHandler := transporthttp.NewSystemTransferHandler(systemTransferUC)
	analyticsHandler := transporthttp.NewAnalyticsHandler(analyticsUC)
	podHandler := transporthttp.NewPodHandler(podUC)

	jwtCfg := sharedjwt.Config{
		AccessSecret:     cfg.JWT.AccessSecret,
		RefreshSecret:    cfg.JWT.RefreshSecret,
		AccessExpiryMins: cfg.JWT.AccessExpiryMinutes,
		RefreshExpiryHrs: cfg.JWT.RefreshExpiryHours,
	}

	server := setupHTTPServer(walletHandler, transferHandler, systemTransferHandler, depositHandler, withdrawalHandler, cateGoryhandler, analyticsHandler, podHandler, jwtCfg, authClient, merchantClient, cfg.InternalServiceSecret)

	kafkaConsumer, err := kafka.NewConsumer(kafka.ConsumerConfig{
		Brokers:  cfg.Kafka.Brokers,
		Username: cfg.Kafka.Username,
		Password: cfg.Kafka.Password,
		CACert:   cfg.Kafka.CACert,
		Topic:    "user.registered",
		GroupID:  "payment-svc",
	})
	if err != nil {
		return nil, fmt.Errorf("create kafka consumer: %w", err)
	}

	httpAddr := cfg.Server.HTTPAddr
	if port := os.Getenv("PORT"); port != "" {
		httpAddr = port
	}
	if httpAddr == "" {
		httpAddr = "9094"
	}
	if !strings.Contains(httpAddr, ":") {
		httpAddr = "0.0.0.0:" + httpAddr
	}

	return &App{
		HTTPServer:    server,
		DB:            pool,
		HTTPAddr:      httpAddr,
		KafkaConsumer: kafkaConsumer,
		WalletUC:      walletUC,
	}, nil
}

func setupHTTPServer(wh *transporthttp.WalletHandler, th *transporthttp.TransferHandler, sth *transporthttp.SystemTransferHandler, dh *transporthttp.DepositHandler, wdh *transporthttp.WithdrawalHandler, ch *transporthttp.CategoryHandler, ah *transporthttp.AnalyticsHandler, ph *transporthttp.PodHandler, jwtCfg sharedjwt.Config, authClient *authclient.Client, merchantClient *merchantclient.Client, internalSecret string) *fiber.App {
	server := fiber.New(fiber.Config{AppName: "payment-service"})
	server.Use(recover.New())
	server.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Transaction-Token"},
	}))

	server.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "payment"})
	})

	transporthttp.RegisterRoutes(server, wh, th, sth, dh, wdh, ch, ah, ph, jwtCfg, authClient, merchantClient, internalSecret)
	return server
}

func (a *App) Run() error {
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
	if a.KafkaConsumer != nil {
		a.KafkaConsumer.Close()
	}
}
