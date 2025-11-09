package ws

import (
	"context"
	"encoding/json"
	"log"

	"github.com/redis/go-redis/v9"
)

// PubSubChannel define o canal Redis Pub/Sub utilizado para broadcast de odds
const PubSubChannel = "odds_updates_broadcast"

// StartRedisSubscriber inicia uma goroutine que escuta o canal Redis Pub/Sub
// e repassa as atualizações recebidas para todos os clientes WebSocket conectados via Hub
//
// Funcionamento:
// - Recebe mensagens JSON do canal Redis
// - Desserializa para OddsUpdate
// - Chama hub.Broadcast para enviar aos clientes conectados
func StartRedisSubscriber(ctx context.Context, r *redis.Client, hub *Hub) {
	sub := r.Subscribe(ctx, PubSubChannel)
	ch := sub.Channel()
	go func() {
		for {
			select {
			case <-ctx.Done():
				_ = sub.Close() // encerra a inscrição ao finalizar o contexto
				return
			case msg := <-ch:
				if msg == nil {
					continue
				}
				var upd OddsUpdate
				if err := json.Unmarshal([]byte(msg.Payload), &upd); err != nil {
					log.Printf("ws subscriber unmarshal error: %v", err)
					continue
				}
				hub.Broadcast(upd) // envia atualização para todos os clientes inscritos
			}
		}
	}()
}
