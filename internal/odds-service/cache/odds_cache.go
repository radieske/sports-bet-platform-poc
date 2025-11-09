package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache encapsula operações de cache de odds no Redis
type Cache struct{ R *redis.Client }

// New cria uma nova instância de Cache
func New(r *redis.Client) *Cache { return &Cache{R: r} }

// keyEvent gera a chave Redis para odds de um evento
func keyEvent(eventID string) string { return "odds:event:" + eventID }

// GetOdds tenta obter odds do cache Redis. Retorna true se encontrou.
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

// SetOdds salva odds no cache Redis com TTL definido
func (c *Cache) SetOdds(ctx context.Context, eventID string, v any, ttl time.Duration) error {
	b, _ := json.Marshal(v)
	return c.R.Set(ctx, keyEvent(eventID), b, ttl).Err()
}
