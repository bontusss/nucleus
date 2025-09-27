package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Redis struct {
	client *redis.Client
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

func NewRedis(config RedisConfig) (*Redis, error) {
	// Sanitize the Host to remove "redis://" if present
	host := config.Host
	if len(host) > 8 && host[:8] == "redis://" {
		host = host[8:] // Remove the "redis://" prefix
	}

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", host, config.Port),
		Password: config.Password,
		DB:       config.DB,
	})

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}

	return &Redis{
		client: client,
	}, nil
}

func (r *Redis) RawClient() *redis.Client {
    return r.client
}

func (r *Redis) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

func (r *Redis) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

func (r *Redis) Delete(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

func (r *Redis) Exists(ctx context.Context, keys ...string) (bool, error) {
	count, err := r.client.Exists(ctx, keys...).Result()
	return count > 0, err
}

func (r *Redis) Incr(ctx context.Context, key string) error {
	return r.client.Incr(ctx, key).Err()
}

func (r *Redis) Decr(ctx context.Context, key string) error {
	return r.client.Decr(ctx, key).Err()
}

func (c *Redis) Close() error {
	return c.client.Close()
}
