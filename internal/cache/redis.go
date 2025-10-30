package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis is the Redis client
type Redis struct{ c *redis.Client }

// NewRedis creates a new Redis client
func NewRedis(addr string, db int) *Redis {
	return &Redis{c: redis.NewClient(&redis.Options{Addr: addr, DB: db})}
}

// Close closes the Redis client
func (r *Redis) Close() error { return r.c.Close() }

// SetSent sets the sent timestamp for a message
func (r *Redis) SetSent(ctx context.Context, id string, sentAt time.Time, ttl time.Duration) error {
	return r.c.Set(ctx, "message:"+id, sentAt.UTC().Format(time.RFC3339Nano), ttl).Err()
}
