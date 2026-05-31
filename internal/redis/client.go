package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisRateLimiter implements a Fixed Window rate limiting algorithm using Redis.
type RedisRateLimiter struct {
	client *redis.Client
}

// NewRedisClient initializes a new connection pool to the Redis server.
func NewRedisClient(addr string, password string) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
		// Username: "default", //cloud redis
		Password: password,
		DB:       0,   // Default DB
		PoolSize: 100, // Important for high-throughput proxies
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return client, nil
}

// NewRedisRateLimiter creates a new rate limiter instance wrapping the Redis client.
func NewRedisRateLimiter(client *redis.Client) *RedisRateLimiter {
	return &RedisRateLimiter{client: client}
}

// Allow checks if the given key has exceeded the limit within the rolling window.
// It uses a Redis Pipeline for atomicity and to minimize network roundtrips.
func (r *RedisRateLimiter) Allow(ctx context.Context, key string, limit int, windowSec int) (bool, error) {
	// Calculate the current window bucket (e.g., floors to the current minute if windowSec=60)
	window := time.Now().Unix() / int64(windowSec)
	redisKey := fmt.Sprintf("ratelimit:%s:%d", key, window)

	// Use a pipeline to INCR and EXPIRE in a single network trip
	pipe := r.client.Pipeline()
	incr := pipe.Incr(ctx, redisKey)
	pipe.Expire(ctx, redisKey, time.Duration(windowSec)*time.Second)

	if _, err := pipe.Exec(ctx); err != nil {
		return false, err
	}

	// Check if the incremented value exceeds our limit
	if incr.Val() > int64(limit) {
		return false, nil // Rate limited!
	}

	return true, nil // Allowed
}
