package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"sale-service/config"
)

// NewPostgresDB - PostgreSQL connection pool yaratish
func NewPostgresDB(cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	// Connection string yaratish
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	// Pool konfiguratsiyasi
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("konfiguratsiyani parse qilishda xatolik: %w", err)
	}

	// Connection pool sozlamalari
	poolConfig.MaxConns = int32(cfg.MaxOpenConns) // Maksimal connection lar
	poolConfig.MinConns = int32(cfg.MaxIdleConns) // Minimal bo'sh connection lar
	poolConfig.MaxConnLifetime = time.Duration(cfg.ConnMaxLifetime) * time.Minute
	poolConfig.MaxConnIdleTime = 30 * time.Minute  // Bo'sh connection ning maksimal vaqti
	poolConfig.HealthCheckPeriod = 1 * time.Minute // Health check oralig'i

	// Connection pool yaratish
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("connection pool yaratishda xatolik: %w", err)
	}

	// Connection ni tekshirish
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database ga ulanishda xatolik: %w", err)
	}

	return pool, nil
}
