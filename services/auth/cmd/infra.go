package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/moneymate-2026/moneymate-backend/auth/config"
	"github.com/moneymate-2026/moneymate-backend/auth/internal/adapter/postgres"
	redisadapter "github.com/moneymate-2026/moneymate-backend/auth/internal/adapter/redis"
)

func connectPostgres(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	return postgres.ConnectDB(ctx, &postgres.Config{
		DSN:             cfg.Database.DSN,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MinOpenConns:    cfg.Database.MinOpenConns,
		MaxConnLifetime: cfg.Database.MaxConnLifetime,
		MaxIdleTime:     cfg.Database.MaxIdleTime,
	})
}

func connectRedis(cfg *config.Config) (*redis.Client, error) {
	return redisadapter.NewClient(redisadapter.Config{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       0, 
	})
}