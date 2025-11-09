package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"github.com/radieske/sports-bet-platform-poc/pkg/contracts/events"

	"github.com/radieske/sports-bet-platform-poc/internal/odds-processor/cache"
	"github.com/radieske/sports-bet-platform-poc/internal/odds-processor/consumer"
	"github.com/radieske/sports-bet-platform-poc/internal/odds-processor/pubsub"
	"github.com/radieske/sports-bet-platform-poc/internal/odds-processor/repository"
	sharedcache "github.com/radieske/sports-bet-platform-poc/internal/shared/cache"
	"github.com/radieske/sports-bet-platform-poc/internal/shared/config"
	"github.com/radieske/sports-bet-platform-poc/internal/shared/db"
	"github.com/radieske/sports-bet-platform-poc/internal/shared/logger"
)

func main() {
	cfg := config.Load()
	log, err := logger.New(cfg.ServiceName, cfg.Env)
	if err != nil {
		panic(err)
	}
	defer log.Sync()

	// Conexão com Postgres usando DSN configurado.
	pg, err := db.ConnectPostgres(cfg.PostgresDSN)
	if err != nil {
		log.Fatal("postgres connect", zap.Error(err))
	}
	defer pg.Close()

	// Conexão com Redis usando endereço configurado.
	redisClient, err := sharedcache.ConnectRedis(cfg.RedisAddr)
	if err != nil {
		log.Fatal("redis connect", zap.Error(err))
	}
	defer redisClient.Close()

	// Instâncias de cache e repositório para o processamento de odds.
	ttl := 60 * time.Second
	rcache := cache.NewRedisCache(redisClient, ttl)
	repo := repository.NewPostgresRepo(pg)

	// Configuração do dialer do Kafka com timeouts e suporte IPv4/IPv6.
	kDialer := &kafka.Dialer{
		Timeout:   10 * time.Second,
		DualStack: true,
		// ClientID definido para identificação nas métricas do broker.
		ClientID: cfg.ServiceName,
	}

	// Configuração do reader do Kafka usando brokers do ambiente.
	// StartOffset define o ponto inicial quando não há offset comprometido.
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     splitCSV(cfg.KafkaBrokers),
		GroupID:     "odds-processor",
		Topic:       cfg.TopicOddsUpdates,
		MinBytes:    10e3, // ~10KB
		MaxBytes:    10e6, // ~10MB
		MaxWait:     500 * time.Millisecond,
		StartOffset: kafka.FirstOffset,
		Dialer:      kDialer,
	})
	defer reader.Close()

	// Métricas Prometheus para contagem de consumo, cache, persistência e erros.
	consumed := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "odds_proc_messages_consumed_total",
		Help: "mensagens consumidas",
	})
	cached := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "odds_proc_cache_sets_total",
		Help: "sets no cache",
	})
	persist := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "odds_proc_db_writes_total",
		Help: "escritas no banco (upsert+history)",
	})
	errorsBy := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "odds_proc_errors_total",
		Help: "erros por estágio",
	}, []string{"stage"})
	prometheus.MustRegister(consumed, cached, persist, errorsBy)

	// Broadcaster para enviar atualizações via Redis Pub/Sub ao serviço de WebSocket.
	broadcaster := pubsub.NewRedisBroadcaster(redisClient)

	// Processador que coordena leitura do Kafka, cache, persistência e broadcast.
	proc := &consumer.Processor{
		Log:    log,
		Reader: reader,
		Repo:   repo,
		Cache:  rcache,

		OnConsumed: func() { consumed.Inc() },
		OnCached:   func() { cached.Inc() },
		OnPersist:  func() { persist.Inc() },
		OnError:    func(stage string) { errorsBy.WithLabelValues(stage).Inc() },

		// Publicação de atualização no canal do WebSocket após persistência bem-sucedida.
		OnAfterPersist: func(ev events.OddsUpdate) {
			msg := pubsub.WSUpdate{EventID: ev.EventID, Payload: ev}
			b, _ := json.Marshal(msg)

			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()

			if err := broadcaster.Publish(ctx, pubsub.ChannelOddsBroadcast, b); err != nil {
				log.Warn("ws broadcast publish failed", zap.Error(err))
			}
		},
	}

	// Servidor HTTP para métricas e health check.
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
			defer cancel()

			// Verificação leve de Postgres e Redis.
			if err := pg.PingContext(ctx); err != nil {
				http.Error(w, "pg", http.StatusServiceUnavailable)
				return
			}
			if err := redisClient.Ping(ctx).Err(); err != nil {
				http.Error(w, "redis", http.StatusServiceUnavailable)
				return
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		})

		addr := fmt.Sprintf(":%s", cfg.MetricsPort)
		log.Info("metrics/health listening", zap.String("addr", addr))
		_ = http.ListenAndServe(addr, mux)
	}()

	// Contexto para encerramento gracioso por sinais de sistema.
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log.Info("odds-processor started")
	if err := proc.Run(ctx); err != nil && ctx.Err() == nil {
		log.Fatal("processor stopped with error", zap.Error(err))
	}
	log.Info("odds-processor stopped")
}

// splitCSV separa uma lista de brokers no formato "host1:9092,host2:9092".
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			if start < i {
				out = append(out, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}
