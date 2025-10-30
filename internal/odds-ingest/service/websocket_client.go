package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/radieske/sports-bet-platform-poc/internal/odds-ingest/publisher"
	"github.com/radieske/sports-bet-platform-poc/pkg/contracts/events"
)

// WSClient representa um cliente WebSocket responsável por consumir odds de um fornecedor
// e publicar as atualizações recebidas em um tópico Kafka.
type WSClient struct {
	URL       string                    // URL do endpoint WebSocket do fornecedor
	Log       *zap.Logger               // Logger estruturado
	Publisher *publisher.KafkaPublisher // Publisher Kafka para envio das odds
}

// Start inicia o loop de conexão e escuta do WebSocket.
// Em caso de desconexão, tenta reconectar automaticamente com backoff.
func (c *WSClient) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			c.Log.Info("context canceled, stopping WS client")
			return
		default:
			if err := c.connectAndListen(ctx); err != nil {
				c.Log.Warn("connection closed", zap.Error(err))
				time.Sleep(3 * time.Second) // Aguarda antes de tentar reconectar
			}
		}
	}
}

// connectAndListen estabelece a conexão WebSocket e processa mensagens recebidas.
// Cada mensagem é desserializada e publicada no Kafka.
func (c *WSClient) connectAndListen(ctx context.Context) error {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, c.URL, nil)
	if err != nil {
		return err
	}
	defer conn.Close()
	c.Log.Info("connected to supplier WS", zap.String("url", c.URL))

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) || errors.Is(err, context.Canceled) {
				return nil
			}
			c.Log.Error("read message failed", zap.Error(err))
			return err
		}

		var update events.OddsUpdate
		if err := json.Unmarshal(message, &update); err != nil {
			c.Log.Warn("invalid message", zap.Error(err))
			continue
		}

		// Publica a atualização recebida no Kafka
		if err := c.Publisher.Publish(ctx, update); err != nil {
			c.Log.Error("failed to publish to Kafka", zap.Error(err))
		}
	}
}
