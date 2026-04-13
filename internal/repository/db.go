package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"tax-module/internal/config"
)

// NewPostgresPool creates a pgxpool connected to PostgreSQL.
func NewPostgresPool(ctx context.Context, cfg config.DatabaseConfig, log *zerolog.Logger) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("parsing db config: %w", err)
	}

	poolCfg.MaxConns = int32(cfg.MaxOpenConns)
	poolCfg.MinConns = int32(cfg.MaxIdleConns)

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	log.Info().
		Str("host", cfg.Host).
		Int("port", cfg.Port).
		Str("database", cfg.DBName).
		Msg("Database connected")

	return pool, nil
}

// RunMigrations applies all up migrations from the given directory.
func RunMigrations(dsn string, migrationsPath string, log *zerolog.Logger) error {
	// Import side-effects for migrate drivers
	// Actual migration runner is called from main using golang-migrate CLI or programmatically.
	// This is a placeholder — we run migrations via Makefile `make migrate-up`.
	log.Info().Str("path", migrationsPath).Msg("Migrations should be run via: make migrate-up")
	return nil
}
