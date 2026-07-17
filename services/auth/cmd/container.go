package main

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/moneymate-2026/moneymate-backend/auth/config"
	postgresrepo "github.com/moneymate-2026/moneymate-backend/auth/internal/adapter/postgres/repo"
	redisadapter "github.com/moneymate-2026/moneymate-backend/auth/internal/adapter/redis"
	"github.com/moneymate-2026/moneymate-backend/auth/internal/infra/hasher"
	"github.com/moneymate-2026/moneymate-backend/auth/internal/infra/idgen"
	authmailer "github.com/moneymate-2026/moneymate-backend/auth/internal/infra/mailer"
	"github.com/moneymate-2026/moneymate-backend/auth/internal/infra/tokenissuer"
	httptransport "github.com/moneymate-2026/moneymate-backend/auth/internal/transport/http"
	usecase "github.com/moneymate-2026/moneymate-backend/auth/internal/usecases"

	jwtutil "github.com/moneymate-2026/moneymate-backend/shared/pkg/jwt"
	sharedmailer "github.com/moneymate-2026/moneymate-backend/shared/pkg/mailer"
	"github.com/moneymate-2026/moneymate-backend/shared/pkg/pgxtx"
)

type container struct {
	authHandler *httptransport.AuthHandler
}

func buildContainer(cfg *config.Config, pool *pgxpool.Pool, redisClient *redis.Client) *container {
	// ── Repos / Adapters ────────────────────────────────────
	userRepo := postgresrepo.NewUserRepo(pool)
	roleRepo := postgresrepo.NewRoleRepo(pool)
	refreshTokenRepo := postgresrepo.NewRefreshTokenRepo(pool)
	store := redisadapter.NewStore(redisClient)
	txManager := pgxtx.New(pool)

	// ── Infra: hasher, idgen, tokenissuer, mailer ──────────
	passwordHasher := hasher.New()
	idGenerator := idgen.New()

	jwtCfg := jwtutil.Config{
		AccessSecret:      cfg.JWT.AccessSecret,
		RefreshSecret:     cfg.JWT.RefreshSecret,
		AccessExpiryMins:  cfg.JWT.AccessExpiryMinutes,
		RefreshExpiryHrs:  cfg.JWT.RefreshExpiryHours,
		TxTokenExpirySecs: 60,
	}
	tokenIssuer := tokenissuer.New(jwtCfg)

	smtpClient := sharedmailer.New(sharedmailer.Config{
		Host:        cfg.SMTP.Host,
		Port:        cfg.SMTP.Port,
		Username:    cfg.SMTP.Username,
		Password:    cfg.SMTP.Password,
		FromAddress: cfg.SMTP.FromAddress,
		FromName:    cfg.SMTP.FromName,
	})
	otpMailer := authmailer.NewOtpMail(smtpClient)

	// ── Usecases ────────────────────────────────────────────
	otpUsecase := usecase.NewOTPUsecase(userRepo, store, otpMailer, cfg.OTP)
	authUsecase := usecase.NewAuthUsecase(
		userRepo, roleRepo, refreshTokenRepo, store, txManager,
		passwordHasher, idGenerator, tokenIssuer, jwtCfg,
	)

	// ── Handlers ────────────────────────────────────────────
	authHandler := httptransport.NewAuthHandler(authUsecase, otpUsecase)

	return &container{
		authHandler: authHandler,
	}
}