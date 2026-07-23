package postgres

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	DSN             string
	MaxOpenConns    int
	MinOpenConns    int
	MaxConnLifetime time.Duration
	MaxIdleTime     time.Duration
}

func ConnectDB(ctx context.Context,dsn string) (*pgxpool.Pool, error) {

poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to parse DSN: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	log.Println("Database connected ✅")
	return pool, nil
}


func RunMigrations(dsn string, migrationsPath string) error {
	log.Println("Running database migrations...")

	m, err := migrate.New(
		fmt.Sprintf("file://%s", migrationsPath),
		dsn,
	)
	if err != nil {
		return fmt.Errorf("could not create migrate instance: %w", err)
	}
	defer m.Close()

	err = m.Up()
	if err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Println("Migrations are already up to date ✅")
			return nil
		}
		return fmt.Errorf("could not run up migrations: %w", err)
	}

	log.Println("Migrations applied successfully ✅")
	return nil
}