package command

import (
	"context"
	"crypto-opportunities-bot/internal/config"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient creates a new Redis client
func NewRedisClient(cfg config.RedisConfig) (*redis.Client, error) {
	if cfg.Host == "" {
		return nil, nil // Redis is optional
	}

	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 2,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return client, nil
}

// CloseRedisClient closes Redis client connection
func CloseRedisClient(client *redis.Client) error {
	if client == nil {
		return nil
	}
	return client.Close()
}
