package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"miniot/apps/api/internal/config"
)

const (
	ConnectionTimeout = 5 * time.Second
)

type Clients struct {
	Pool  *pgxpool.Pool
	Redis *redis.Client
}

// InitDatabase initializes connections to PostgreSQL/TimescaleDB and Redis.
func InitDatabase(cfg *config.Config) (*Clients, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ConnectionTimeout)
	defer cancel()

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.DatabaseUser,
		cfg.DatabasePassword,
		cfg.DatabaseHost,
		cfg.DatabasePort,
		cfg.DatabaseName,
		cfg.DatabaseSSLMode,
	)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database dsn: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	redisAddr := fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort)
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: cfg.RedisPassword,
		DB:       0,
	})

	return &Clients{
		Pool:  pool,
		Redis: rdb,
	}, nil
}
