package pubsub

import (
	"context"

	"github.com/redis/go-redis/v9"
)

const ChannelOddsBroadcast = "odds_updates_broadcast"

type RedisBroadcaster struct {
	r *redis.Client
}

func NewRedisBroadcaster(r *redis.Client) *RedisBroadcaster {
	return &RedisBroadcaster{r: r}
}

func (b *RedisBroadcaster) Publish(ctx context.Context, channel string, payload []byte) error {
	return b.r.Publish(ctx, channel, payload).Err()
}

// Payload padrão para o WS do odds-service
type WSUpdate struct {
	EventID string      `json:"eventId"`
	Payload interface{} `json:"payload"`
}
