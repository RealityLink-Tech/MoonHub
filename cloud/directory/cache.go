// cloud/directory/cache.go
package directory

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// OnlineCache tracks agent online status.
type OnlineCache interface {
	MarkOnline(ctx context.Context, agentID, endpoint string) error
	IsOnline(ctx context.Context, agentID string) (bool, string, error)
	MarkOffline(ctx context.Context, agentID string) error
}

// DefaultCacheTTL is the Redis key TTL for online status.
const DefaultCacheTTL = 90 * time.Second

// NewRedisCache creates a Redis-backed OnlineCache.
func NewRedisCache(redisURL string) OnlineCache {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		opts = &redis.Options{Addr: "localhost:6379"}
	}
	client := redis.NewClient(opts)
	return &redisCache{client: client, ttl: DefaultCacheTTL}
}

type redisCache struct {
	client *redis.Client
	ttl    time.Duration
}

func (c *redisCache) MarkOnline(ctx context.Context, agentID, endpoint string) error {
	key := fmt.Sprintf("agent:%s:online", agentID)
	return c.client.Set(ctx, key, endpoint, c.ttl).Err()
}

func (c *redisCache) IsOnline(ctx context.Context, agentID string) (bool, string, error) {
	key := fmt.Sprintf("agent:%s:online", agentID)
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	return true, val, nil
}

func (c *redisCache) MarkOffline(ctx context.Context, agentID string) error {
	key := fmt.Sprintf("agent:%s:online", agentID)
	return c.client.Del(ctx, key).Err()
}
