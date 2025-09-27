package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"sale-service/config"
)

// Cache - Cache interface
type Cache interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string, dest interface{}) error
	Delete(ctx context.Context, key string) error
	Close() error
}

type redisCache struct {
	client *redis.Client // Redis client
	logger *zap.Logger   // Logger
}

// NewRedisCache - Yangi Redis cache yaratish
func NewRedisCache(cfg config.RedisConfig) (Cache, error) {
	// Redis client yaratish
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  5 * time.Second, // Connection timeout
		ReadTimeout:  3 * time.Second, // Read timeout
		WriteTimeout: 3 * time.Second, // Write timeout
		PoolSize:     10,              // Connection pool size
		MinIdleConns: 5,               // Minimal bo'sh connection lar
	})

	// Redis connection ni tekshirish
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ga ulanishda xatolik: %w", err)
	}

	return &redisCache{
		client: client,
	}, nil
}

// Set - Ma'lumotni cache ga saqlash
func (c *redisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	// Obyektni JSON ga o'girish
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal qilishda xatolik: %w", err)
	}

	// Redis ga saqlash
	if err := c.client.Set(ctx, key, data, expiration).Err(); err != nil {
		return fmt.Errorf("cache ga saqlashda xatolik: %w", err)
	}

	return nil
}

// Get - Ma'lumotni cache dan olish
func (c *redisCache) Get(ctx context.Context, key string, dest interface{}) error {
	// Redis dan o'qish
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("cache da topilmadi")
		}
		return fmt.Errorf("cache dan o'qishda xatolik: %w", err)
	}

	// JSON dan obyektga o'girish
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("unmarshal qilishda xatolik: %w", err)
	}

	return nil
}

// Delete - Ma'lumotni cache dan o'chirish
func (c *redisCache) Delete(ctx context.Context, key string) error {
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("cache dan o'chirishda xatolik: %w", err)
	}
	return nil
}

// Close - Redis connection ni yopish
func (c *redisCache) Close() error {
	return c.client.Close()
}
