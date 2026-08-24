package database

import (
	"context"
	"fmt"

	"backend/internal/config"

	"github.com/redis/go-redis/v9"
)

func ConnectValkey(cfg *config.Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Valkey.Addr,
		Username: cfg.Valkey.Username,
		Password: cfg.Valkey.Password,
		DB:       cfg.Valkey.DB,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to valkey: %w", err)
	}

	return client, nil
}
