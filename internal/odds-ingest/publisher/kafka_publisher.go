package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"github.com/radieske/sports-bet-platform-poc/pkg/contracts/events"
)

// KafkaPublisher encapsula o writer Kafka e o logger.
type KafkaPublisher struct {
	writer *kafka.Writer
	log    *zap.Logger
}

// NewKafkaPublisher cria um publisher para um tópico Kafka.
// A função lê a lista de brokers, opcionalmente garante a existência do tópico
// em ambientes de desenvolvimento, e inicializa o writer com timeouts.
func NewKafkaPublisher(brokers []string, topic string, log *zap.Logger) *KafkaPublisher {
	if len(brokers) == 0 {
		log.Fatal("kafka brokers not provided")
	}

	// Contextos com timeout curto para operações de controle (quando aplicáveis).
	ctrlCtx, ctrlCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer ctrlCancel()

	// Criação de tópico apenas quando APP_ENV indica ambiente local ou dev.
	// Esse trecho usa o controller do cluster para emitir o CreateTopics.
	if env := os.Getenv("APP_ENV"); env == "local" || env == "dev" {
		// Conexão com o primeiro broker para obter o controller.
		conn, err := kafka.DialContext(ctrlCtx, "tcp", brokers[0])
		if err != nil {
			log.Fatal("failed to connect to kafka", zap.Error(err))
		}
		defer conn.Close()

		// Descoberta do controller do cluster.
		controller, err := conn.Controller()
		if err != nil {
			log.Fatal("failed to get kafka controller", zap.Error(err))
		}

		// Conexão direta com o controller para operações administrativas.
		controllerAddr := fmt.Sprintf("%s:%d", controller.Host, controller.Port)
		cconn, err := kafka.DialContext(ctrlCtx, "tcp", controllerAddr)
		if err != nil {
			log.Fatal("failed to dial controller", zap.Error(err))
		}
		defer cconn.Close()

		// Configuração de tópico com particionamento e fator de replicação compatíveis com single-broker.
		cfg := kafka.TopicConfig{
			Topic:             topic,
			NumPartitions:     1,
			ReplicationFactor: 1,
		}

		// Tentativa de criação do tópico; em caso de existência prévia, mantém execução.
		if err := cconn.CreateTopics(cfg); err != nil && !strings.Contains(err.Error(), "already exists") {
			log.Warn("failed to create kafka topic", zap.String("topic", topic), zap.Error(err))
		} else if err == nil {
			log.Info("kafka topic created", zap.String("topic", topic))
		}
	}

	// Inicialização do writer com timeouts e balanceamento por menor carga.
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  topic,
		Balancer:               &kafka.LeastBytes{},
		RequiredAcks:           kafka.RequireAll,
		AllowAutoTopicCreation: true,
		BatchTimeout:           10 * time.Millisecond,
		ReadTimeout:            10 * time.Second,
		WriteTimeout:           10 * time.Second,
	}

	return &KafkaPublisher{
		writer: writer,
		log:    log,
	}
}

// Publish serializa o evento em JSON e envia uma mensagem para o tópico configurado.
// A chave da mensagem utiliza o EventID para garantir distribuição consistente por partição.
func (p *KafkaPublisher) Publish(ctx context.Context, e events.OddsUpdate) error {
	value, err := json.Marshal(e)
	if err != nil {
		return err
	}

	msg := kafka.Message{
		Key:   []byte(e.EventID),
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

// Close finaliza o writer e libera recursos associados.
func (p *KafkaPublisher) Close() error {
	return p.writer.Close()
}
