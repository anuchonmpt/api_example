package database

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func NewRedis(ctx context.Context, address, password string, db int) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{Addr: address, Password: password, DB: db})
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping Redis: %w", err)
	}
	return client, nil
}
