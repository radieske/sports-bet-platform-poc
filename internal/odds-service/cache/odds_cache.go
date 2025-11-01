package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct{ R *redis.Client }

func New(r *redis.Client) *Cache { return &Cache{R: r} }

func keyEvent(eventID string) string { return "odds:event:" + eventID }

func (c *Cache) GetOdds(ctx context.Context, eventID string, dst any) (bool, error) {
	b, err := c.R.Get(ctx, keyEvent(eventID)).Bytes()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, json.Unmarshal(b, dst)
}

func (c *Cache) SetOdds(ctx context.Context, eventID string, v any, ttl time.Duration) error {
	b, _ := json.Marshal(v)
	return c.R.Set(ctx, keyEvent(eventID), b, ttl).Err()
}
