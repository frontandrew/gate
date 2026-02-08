package database

import (
	"context"
	"fmt"
	"time"

	"github.com/frontandrew/gate/internal/pkg/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect создает пул подключений к PostgreSQL
func Connect(ctx context.Context, cfg *config.DatabaseConfig) (*pgxpool.Pool, error) {
	// Формируем строку подключения
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
		cfg.SSLMode,
	)

	// Настраиваем конфигурацию пула
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}

	// Настройки пула соединений
	poolConfig.MaxConns = int32(cfg.MaxOpenConns)
	poolConfig.MinConns = int32(cfg.MaxIdleConns)
	poolConfig.MaxConnLifetime = cfg.ConnMaxLifetime
	poolConfig.MaxConnIdleTime = 5 * time.Minute
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	// Создаем пул подключений
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Проверяем подключение
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	return pool, nil
}

// Close закрывает пул подключений
func Close(pool *pgxpool.Pool) {
	if pool != nil {
		pool.Close()
	}
}
