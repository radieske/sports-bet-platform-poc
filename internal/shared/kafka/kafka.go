package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

type Writer = kafka.Writer

func NewWriter(brokers string, topic string) *kafka.Writer {
	return &kafka.Writer{
		Addr:                   kafka.TCP(brokers),
		Topic:                  topic,
		Balancer:               &kafka.LeastBytes{},
		AllowAutoTopicCreation: true,
	}
}

func NewReader(brokers string, topic string, groupID string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{brokers},
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
	})
}

// helper pra enviar mensagem simples
func WriteJSON(ctx context.Context, w *kafka.Writer, key string, payload []byte) error {
	msg := kafka.Message{
		Key:   []byte(key),
		Value: payload,
		Time:  time.Now(),
	}

	return w.WriteMessages(ctx, msg)
}

func ReadNext(ctx context.Context, r *kafka.Reader) (key []byte, value []byte, err error) {
	m, err := r.ReadMessage(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("read kafka message: %w", err)
	}
	return m.Key, m.Value, nil
}
