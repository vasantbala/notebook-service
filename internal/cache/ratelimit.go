package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter interface {
	Allow(ctx context.Context, userID string, limit int, window time.Duration) (bool, error)
}

type redisRateLimiter struct{ client *redis.Client }

func NewRedisRateLimiter(client *redis.Client) RateLimiter {
	return &redisRateLimiter{client: client}
}

func (c *redisRateLimiter) Allow(ctx context.Context, userID string, limit int, window time.Duration) (bool, error) {
	key := fmt.Sprintf("rl:%s", userID)
	now := time.Now().UnixMilli()
	windowMs := window.Milliseconds()

	pipe := c.client.Pipeline()
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", now-windowMs)) // remove old entries
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: now})
	pipe.ZCard(ctx, key)
	pipe.Expire(ctx, key, window)
	cmds, err := pipe.Exec(ctx)
	if err != nil {
		return true, fmt.Errorf("rate limit check: %w", err) // fail open
	}
	count := cmds[2].(*redis.IntCmd).Val()
	return count <= int64(limit), nil
}
