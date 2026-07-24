package app

import (
	"context"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/moneymate-2026/moneymate-backend/auth/config"
	"github.com/moneymate-2026/moneymate-backend/auth/internal/adapter/postgres"
	"github.com/moneymate-2026/moneymate-backend/auth/internal/adapter/postgres/repo"
	rediscard "github.com/moneymate-2026/moneymate-backend/auth/internal/adapter/redis"
	"github.com/moneymate-2026/moneymate-backend/auth/internal/infra/hasher"
	"github.com/moneymate-2026/moneymate-backend/auth/internal/infra/idgen"
	"github.com/moneymate-2026/moneymate-backend/auth/internal/infra/mailer"
	"github.com/moneymate-2026/moneymate-backend/auth/internal/infra/tokenissuer"
	transporthttp "github.com/moneymate-2026/moneymate-backend/auth/internal/transport/http"
	usecase "github.com/moneymate-2026/moneymate-backend/auth/internal/usecases"
	sharedjwt "github.com/moneymate-2026/moneymate-backend/shared/pkg/jwt"
	sharedmailer "github.com/moneymate-2026/moneymate-backend/shared/pkg/mailer"
	sharedpgxtx "github.com/moneymate-2026/moneymate-backend/shared/pkg/pgxtx"
)

type App struct {
	Server      *fiber.App
	DB          *pgxpool.Pool
	RedisClient *redis.Client
	Config      *config.Config
}

func Build(cfg *config.Config) (*App, error) {
	
	pool, err := postgres.ConnectDB(context.Background(), cfg.Database.DSN)
	if err != nil {
		return nil, err
	}
	dsn := fmt.Sprintf(
    "postgres://%s:%s@%s:%s/%s?sslmode=disable",
    cfg.Database.User,
    cfg.Database.Password,
    cfg.Database.Host,
    cfg.Database.Port,
    cfg.Database.Name,
)
	err = postgres.RunMigrations(dsn, cfg.Database.MigrationsPath)
	if err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	redisClient, err := setupRedis(cfg)
	if err != nil {
		return nil, err
	}

	authHandler := setupDependencies(pool, redisClient, cfg)

	server := setupServer(cfg, authHandler)

	return &App{
		Server:      server,
		DB:          pool,
		RedisClient: redisClient,
		Config:      cfg,
	}, nil
}

func (a *App) Close() {
	if a.DB != nil {
		a.DB.Close()
	}
	if a.RedisClient != nil {
		a.RedisClient.Close()
	}
}


func setupRedis(cfg *config.Config) (*redis.Client, error) {
	return rediscard.NewClient(rediscard.Config{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       0,
	})
}

func setupDependencies(pool *pgxpool.Pool, redisClient *redis.Client, cfg *config.Config) *transporthttp.AuthHandler {
	jwtCfg := sharedjwt.Config{
		AccessSecret:     cfg.JWT.AccessSecret,
		RefreshSecret:    cfg.JWT.RefreshSecret,
		AccessExpiryMins: cfg.JWT.AccessExpiryMinutes,
		RefreshExpiryHrs: cfg.JWT.RefreshExpiryHours,
	}

	smtpCfg := sharedmailer.Config{
		Host:        cfg.SMTP.Host,
		Port:        cfg.SMTP.Port,
		Username:    cfg.SMTP.Username,
		Password:    cfg.SMTP.Password,
		FromAddress: cfg.SMTP.FromAddress,
		FromName:    cfg.SMTP.FromName,
	}

	h := hasher.New()
	g := idgen.New()
	issuer := tokenissuer.New(jwtCfg)
	mailerClient := sharedmailer.New(smtpCfg)
	otpMailer := mailer.NewOtpMail(mailerClient)


	userRepo := repo.NewUserRepo(pool)
	roleRepo := repo.NewRoleRepo(pool)
	refreshTokenRepo := repo.NewRefreshTokenRepo(pool)
	store := rediscard.NewStore(redisClient)
	txMgr := sharedpgxtx.New(pool)

	authUC := usecase.NewAuthUsecase(userRepo, roleRepo, refreshTokenRepo, store, txMgr, h, g, issuer, jwtCfg)

	otpMailerIface := usecase.EmailSender(otpMailer)
	if cfg.Env == "dev" {
		otpMailerIface = mailer.NewDevOtpMail()
		log.Println("[DEV MODE] OTP codes will be logged to console instead of sent via email")
	}
	otpUC := usecase.NewOTPUsecase(userRepo, store, otpMailerIface, cfg.OTP)


	return transporthttp.NewAuthHandler(authUC, otpUC, userRepo, cfg.JWT.AccessSecret)
}

func setupServer(cfg *config.Config, authHandler *transporthttp.AuthHandler) *fiber.App {
	server := fiber.New(fiber.Config{
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		AppName:      "auth-service",
	})

	server.Use(recover.New())
	server.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Device-Id"},
	}))

	server.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "auth"})
	})

	noopAuth := func(c fiber.Ctx) error { return c.Next() }
	transporthttp.RegisterRoutes(server, authHandler, noopAuth)

	return server
}



// package app

// import (
// 	"context"
// 	"fmt"
// 	"time"

// 	"github.com/gofiber/fiber/v3"
// 	"github.com/gofiber/fiber/v3/middleware/cors"
// 	"github.com/gofiber/fiber/v3/middleware/limiter"
// 	"github.com/gofiber/fiber/v3/middleware/logger"
// 	"github.com/gofiber/fiber/v3/middleware/recover"
// 	"github.com/jackc/pgx/v5/pgxpool"
// 	"github.com/redis/go-redis/v9"

// 	"github.com/moneymate-2026/moneymate-backend/auth/config"
// 	"github.com/moneymate-2026/moneymate-backend/auth/internal/adapter/postgres"
// 	"github.com/moneymate-2026/moneymate-backend/auth/internal/adapter/postgres/repo"
// 	rediscard "github.com/moneymate-2026/moneymate-backend/auth/internal/adapter/redis"
// 	"github.com/moneymate-2026/moneymate-backend/auth/internal/infra/hasher"
// 	"github.com/moneymate-2026/moneymate-backend/auth/internal/infra/idgen"
// 	"github.com/moneymate-2026/moneymate-backend/auth/internal/infra/mailer"
// 	"github.com/moneymate-2026/moneymate-backend/auth/internal/infra/tokenissuer"
// 	transporthttp "github.com/moneymate-2026/moneymate-backend/auth/internal/transport/http"
// 	usecase "github.com/moneymate-2026/moneymate-backend/auth/internal/usecases"
// 	sharedjwt "github.com/moneymate-2026/moneymate-backend/shared/pkg/jwt"
// 	sharedmailer "github.com/moneymate-2026/moneymate-backend/shared/pkg/mailer"
// 	sharedpgxtx "github.com/moneymate-2026/moneymate-backend/shared/pkg/pgxtx"
// )

// const dbConnectTimeout = 10 * time.Second

// type App struct {
// 	Server      *fiber.App
// 	DB          *pgxpool.Pool
// 	RedisClient *redis.Client
// 	Config      *config.Config
// }

// func Build(cfg *config.Config) (app *App, err error) {
// 	var pool *pgxpool.Pool
// 	var redisClient *redis.Client

// 	defer func() {
// 		if err != nil {
// 			if redisClient != nil {
// 				redisClient.Close()
// 			}
// 			if pool != nil {
// 				pool.Close()
// 			}
// 		}
// 	}()

	
// 	pool, err = connectDB(cfg)
// 	if err != nil {
// 		return nil, fmt.Errorf("connect db: %w", err)
// 	}

// 	redisClient, err = setupRedis(cfg)
// 	if err != nil {
// 		return nil, fmt.Errorf("setup redis: %w", err)
// 	}

// 	authHandler := setupDependencies(pool, redisClient, cfg)

// 	server := setupServer(cfg, authHandler, pool, redisClient)

// 	return &App{
// 		Server:      server,
// 		DB:          pool,
// 		RedisClient: redisClient,
// 		Config:      cfg,
// 	}, nil
// }

// func (a *App) Close() {
// 	if a.DB != nil {
// 		a.DB.Close()
// 	}
// 	if a.RedisClient != nil {
// 		a.RedisClient.Close()
// 	}
// }

// // ─── Private Setup Helpers ───────────────────────────────────────────────────

// func connectDB(cfg *config.Config) (*pgxpool.Pool, error) {
// 	ctx, cancel := context.WithTimeout(context.Background(), dbConnectTimeout)
// 	defer cancel()
// 	return postgres.ConnectDB(ctx, cfg.Database.DSN)
// }

// func setupRedis(cfg *config.Config) (*redis.Client, error) {
// 	return rediscard.NewClient(rediscard.Config{
// 		Addr:     cfg.Redis.Addr,
// 		Password: cfg.Redis.Password,
// 		DB:       0,
// 	})
// }

// func setupDependencies(pool *pgxpool.Pool, redisClient *redis.Client, cfg *config.Config) *transporthttp.AuthHandler {
// 	jwtCfg := sharedjwt.Config{
// 		AccessSecret:     cfg.JWT.AccessSecret,
// 		RefreshSecret:    cfg.JWT.RefreshSecret,
// 		AccessExpiryMins: cfg.JWT.AccessExpiryMinutes,
// 		RefreshExpiryHrs: cfg.JWT.RefreshExpiryHours,
// 	}

// 	smtpCfg := sharedmailer.Config{
// 		Host:        cfg.SMTP.Host,
// 		Port:        cfg.SMTP.Port,
// 		Username:    cfg.SMTP.Username,
// 		Password:    cfg.SMTP.Password,
// 		FromAddress: cfg.SMTP.FromAddress,
// 		FromName:    cfg.SMTP.FromName,
// 	}

// 	// Infra
// 	h := hasher.New()
// 	g := idgen.New()
// 	issuer := tokenissuer.New(jwtCfg)
// 	mailerClient := sharedmailer.New(smtpCfg)
// 	otpMailer := mailer.NewOtpMail(mailerClient)

// 	// Repositories
// 	userRepo := repo.NewUserRepo(pool)
// 	roleRepo := repo.NewRoleRepo(pool)
// 	refreshTokenRepo := repo.NewRefreshTokenRepo(pool)
// 	store := rediscard.NewStore(redisClient)
// 	txMgr := sharedpgxtx.New(pool)

// 	// Usecases
// 	authUC := usecase.NewAuthUsecase(userRepo, roleRepo, refreshTokenRepo, store, txMgr, h, g, issuer, jwtCfg)
// 	otpUC := usecase.NewOTPUsecase(userRepo, store, otpMailer, cfg.OTP)

// 	// Handlers — transport layer only ever talks to usecases, never
// 	// touches repos or secrets directly.
// 	return transporthttp.NewAuthHandler(authUC, otpUC,userRepo,jwtCfg.AccessSecret)
// }

// // setupServer configures the Fiber application, middlewares, and routing.
// func setupServer(cfg *config.Config, authHandler *transporthttp.AuthHandler, pool *pgxpool.Pool, redisClient *redis.Client) *fiber.App {
// 	server := fiber.New(fiber.Config{
// 		ReadTimeout:  cfg.Server.ReadTimeout,
// 		WriteTimeout: cfg.Server.WriteTimeout,
// 		BodyLimit:    1 * 1024 * 1024, // 1MB — this service only ever accepts small JSON auth payloads
// 		AppName:      "auth-service",
// 	})

// 	server.Use(recover.New())
// 	server.Use(logger.New(logger.Config{
// 		Format: "[${time}] ${status} - ${method} ${path} (${latency})\n",
// 	}))
// 	server.Use(cors.New(cors.Config{
// 		AllowOrigins: cfg.Server.AllowedOrigins,
// 		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
// 		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Device-Id"},
// 	}))

// 	registerHealthRoutes(server, pool, redisClient)

// 	// Coarse global rate limit — baseline protection for every route.
// 	// Tighter, endpoint-specific limits (login, OTP send/verify) belong
// 	// inside RegisterRoutes where per-path scoping is possible.
// 	server.Use(limiter.New(limiter.Config{
// 		Max:        60,
// 		Expiration: 1 * time.Minute,
// 		KeyGenerator: func(c fiber.Ctx) string {
// 			return c.IP()
// 		},
// 	}))

// 	// Temporary no-op auth middleware for route registration
// 	noopAuth := func(c fiber.Ctx) error { return c.Next() }
// 	transporthttp.RegisterRoutes(server, authHandler, noopAuth)

// 	return server
// }

// // registerHealthRoutes splits liveness (is the process up) from
// // readiness (can it actually serve a request right now) — an
// // orchestrator should stop routing traffic here if /ready fails,
// // even while /health still reports the process alive.
// func registerHealthRoutes(server *fiber.App, pool *pgxpool.Pool, redisClient *redis.Client) {
// 	server.Get("/health", func(c fiber.Ctx) error {
// 		return c.JSON(fiber.Map{"status": "ok", "service": "auth"})
// 	})

// 	server.Get("/ready", func(c fiber.Ctx) error {
// 		ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
// 		defer cancel()

// 		if err := pool.Ping(ctx); err != nil {
// 			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
// 				"status": "unavailable", "dependency": "postgres", "error": err.Error(),
// 			})
// 		}
// 		if err := redisClient.Ping(ctx).Err(); err != nil {
// 			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
// 				"status": "unavailable", "dependency": "redis", "error": err.Error(),
// 			})
// 		}
// 		return c.JSON(fiber.Map{"status": "ready", "service": "auth"})
// 	})
// }