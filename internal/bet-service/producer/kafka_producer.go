package producer

import (
	"context"
	"encoding/json"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/radieske/sports-bet-platform-poc/pkg/contracts/events"
)

type KafkaPublisher struct {
	Writer *kafka.Writer
	Topic  string
}

func NewKafkaPublisher(w *kafka.Writer, topic string) *KafkaPublisher {
	return &KafkaPublisher{Writer: w, Topic: topic}
}

func (p *KafkaPublisher) PublishBetPlaced(ctx context.Context, e events.BetPlaced) error {
	e.TsUnixMs = time.Now().UnixMilli()
	b, _ := json.Marshal(e)
	return p.Writer.WriteMessages(ctx, kafka.Message{Value: b})
}
