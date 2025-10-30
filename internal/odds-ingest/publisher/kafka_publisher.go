package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"github.com/radieske/sports-bet-platform-poc/pkg/contracts/events"
)

// KafkaPublisher encapsula a lógica de publicação de mensagens de odds em um tópico Kafka
// writer: produtor Kafka
// log: logger estruturado
type KafkaPublisher struct {
	writer *kafka.Writer
	log    *zap.Logger
}

// NewKafkaPublisher inicializa um publisher Kafka para o tópico informado.
// Garante que o tópico existe, criando-o se necessário e retorna o publisher pronto para uso.
func NewKafkaPublisher(brokers []string, topic string, log *zap.Logger) *KafkaPublisher {
	// Conecta ao primeiro broker para operações administrativas
	conn, err := kafka.Dial("tcp", brokers[0])
	if err != nil {
		log.Fatal("failed to connect to kafka", zap.Error(err))
	}
	defer conn.Close()

	// Obtém informações do controller do cluster Kafka
	controller, err := conn.Controller()
	if err != nil {
		log.Fatal("failed to get kafka controller", zap.Error(err))
	}

	// Conecta ao controller para criar o tópico
	controllerConn, err := kafka.Dial("tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	if err != nil {
		log.Fatal("failed to dial controller", zap.Error(err))
	}
	defer controllerConn.Close()

	topicConfigs := []kafka.TopicConfig{{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	}}

	// Cria o tópico se ele ainda não existir
	if err := controllerConn.CreateTopics(topicConfigs...); err != nil {
		// Ignora erro se o tópico já existir
		if !strings.Contains(err.Error(), "Topic with this name already exists") {
			log.Warn("failed to create kafka topic", zap.String("topic", topic), zap.Error(err))
		}
	} else {
		log.Info("kafka topic created", zap.String("topic", topic))
	}

	return &KafkaPublisher{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			RequiredAcks: kafka.RequireAll,
		},
		log: log,
	}
}

// Publish publica um evento OddsUpdate serializado em JSON no tópico Kafka
func (p *KafkaPublisher) Publish(ctx context.Context, e events.OddsUpdate) error {
	value, err := json.Marshal(e)
	if err != nil {
		return err
	}
	msg := kafka.Message{
		Key:   []byte(e.EventID), // Usa o EventID como chave da partição
		Value: value,
		Time:  time.Now(),
	}
	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		p.log.Error("failed to publish odds update", zap.Error(err))
		return err
	}
	p.log.Debug("published odds update", zap.String("event_id", e.EventID))
	return nil
}

// Close encerra o writer Kafka liberando recursos
func (p *KafkaPublisher) Close() error {
	return p.writer.Close()
}
