package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	client *redis.Client
}

func NewRateLimit(client *redis.Client) *RateLimiter {
	return &RateLimiter{client: client}
}

func (r *RateLimiter) Check(ctx context.Context, key string, limit int, window time.Duration) (bool, int, time.Duration, error) {
	now := time.Now()
	windowStart := now.Add(-window)

	// Remove old entries
	r.client.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart.UnixMilli()))

	// Count current entries
	count, err := r.client.ZCard(ctx, key).Result()
	if err != nil {
		return false, 0, 0, err
	}

	// Check if limit exceeded
	if int(count) > limit {
		// Get oldest entry to calculate reset time
		oldest, err := r.client.ZRangeWithScores(ctx, key, 0, 0).Result()
		if err != nil {
			return false, 0, 0, err
		}

		if len(oldest) > 0 {
			oldestTime := time.Unix(0, int64(oldest[0].Score)*int64(time.Millisecond))
			resetTime := oldestTime.Add(window)
			timeUntilReset := time.Until(resetTime)
			return true, int(count), timeUntilReset, nil
		}
		return true, int(count), 0, nil
	}
	return false, int(count), 0, nil
}

func (r *RateLimiter) Increment(ctx context.Context, key string, window time.Duration) error {
	now := time.Now()
	member := fmt.Sprintf("%d", now.UnixNano())
	score := float64(now.UnixMilli())

	// Add new entry
	_, err := r.client.ZAdd(ctx, key, redis.Z{
		Score:  score,
		Member: member,
	}).Result()
	if err != nil {
		return err
	}

	// Set expiration to clean up automatically
	_, err = r.client.Expire(ctx, key, window+time.Minute).Result()
	return err
}

// BlockKey blocks a key for a specific duration
func (r *RateLimiter) BlockKey(ctx context.Context, key string, duration time.Duration) error {
	_, err := r.client.Set(ctx, key, "blocked", duration).Result()
	return err
}

// IsKeyBlocked checks if a key is currently blocked
func (r *RateLimiter) IsKeyBlocked(ctx context.Context, key string) (bool, time.Duration, error) {
	ttl, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		return false, 0, err
	}

	if ttl > 0 {
		return true, ttl, nil
	}

	return false, 0, nil
}

// GetRemainingAttempts gets remaining attempts for a key
func (r *RateLimiter) GetRemainingAttempts(ctx context.Context, key string, limit int, window time.Duration) (int, error) {
	now := time.Now()
	windowStart := now.Add(-window)

	// Remove old entries
	r.client.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart.UnixMilli()))

	// Count current entries
	count, err := r.client.ZCard(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	remaining := limit - int(count)
	if remaining < 0 {
		return 0, nil
	}

	return remaining, nil
}