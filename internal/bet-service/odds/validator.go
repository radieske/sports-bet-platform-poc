package odds

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type Validator struct {
	Rdb *redis.Client
}

func NewValidator(r *redis.Client) *Validator { return &Validator{Rdb: r} }

// Espera chave "odds:{eventID}:{market}:{selection}" => valor string com odd, ex: "1.85"
func (v *Validator) CurrentOdd(ctx context.Context, eventID, market, selection string) (string, error) {
	key := fmt.Sprintf("odds:%s:%s:%s", eventID, market, selection)
	val, err := v.Rdb.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}
	return val, nil
}
