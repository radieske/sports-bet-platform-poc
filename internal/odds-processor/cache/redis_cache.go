package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/radieske/sports-bet-platform-poc/pkg/contracts/events"
)

// RedisCache encapsula operações de cache de odds no Redis
// Client: cliente Redis
// TTL: tempo de expiração dos registros
type RedisCache struct {
	Client *redis.Client
	TTL    time.Duration
}

// NewRedisCache cria uma instância de cache Redis com TTL configurável
func NewRedisCache(c *redis.Client, ttl time.Duration) *RedisCache {
	return &RedisCache{Client: c, TTL: ttl}
}

// key gera a chave Redis para odds atuais de um evento
func key(eventID string) string { return "odds:current:" + eventID }

// SetCurrent armazena a odd atual de um evento no Redis com TTL definido
func (r *RedisCache) SetCurrent(ctx context.Context, e events.OddsUpdate) error {
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	return r.Client.Set(ctx, key(e.EventID), b, r.TTL).Err()
}
