package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/radieske/sports-bet-platform-poc/internal/shared/cache"
	"github.com/radieske/sports-bet-platform-poc/internal/shared/config"
	"github.com/radieske/sports-bet-platform-poc/internal/shared/db"
	"github.com/radieske/sports-bet-platform-poc/internal/shared/kafka"
	"github.com/radieske/sports-bet-platform-poc/internal/shared/logger"
)

func main() {
	// Carrega as configurações do serviço
	cfg := config.Load()

	// Inicializa o logger estruturado
	log, err := logger.New(cfg.ServiceName, cfg.Env)
	if err != nil {
		panic(fmt.Errorf("logger init: %w", err))
	}
	defer log.Sync()

	log.Info("starting service", zap.String("service", cfg.ServiceName), zap.String("env", cfg.Env))

	// Estabelece conexão com o banco de dados Postgres
	pg, err := db.ConnectPostgres(cfg.PostgresDSN)
	if err != nil {
		log.Fatal("failed to connect postgres", zap.Error(err))
	}
	defer pg.Close()
	log.Info("postgres connected")

	// Estabelece conexão com o cache Redis
	redisClient, err := cache.ConnectRedis(cfg.RedisAddr)
	if err != nil {
		log.Fatal("failed to connect redis", zap.Error(err))
	}
	defer redisClient.Close()
	log.Info("redis connected")

	// Inicializa writer Kafka para o tópico de odds (validação de conectividade)
	writer := kafka.NewWriter(cfg.KafkaBrokers, "odds_updates")
	defer writer.Close()
	log.Info("kafka writer ready", zap.String("topic", "odds_updates"))

	// Inicializa servidor HTTP para métricas e health check
	mux := http.NewServeMux()

	// Exposição de métricas Prometheus
	mux.Handle("/metrics", promhttp.Handler())

	// Endpoint de health check: valida a saúde das dependências críticas (Postgres, Redis, Kafka)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		// Valida conexão com Postgres
		if err := pg.PingContext(ctx); err != nil {
			http.Error(w, "postgres not healthy: "+err.Error(), http.StatusServiceUnavailable)
			return
		}

		// Valida conexão com Redis
		if err := redisClient.Ping(ctx).Err(); err != nil {
			http.Error(w, "redis not healthy: "+err.Error(), http.StatusServiceUnavailable)
			return
		}

		// Valida conexão com Kafka enviando mensagem de teste
		kctx, kcancel := context.WithTimeout(ctx, 2*time.Second)
		defer kcancel()

		start := time.Now()
		if err := kafkaWriteTest(kctx, writer); err != nil {
			log.Warn("kafka health check failed",
				zap.Error(err),
				zap.Duration("latency", time.Since(start)),
			)
			http.Error(w, "kafka not healthy: "+err.Error(), http.StatusServiceUnavailable)
			return
		}

		log.Debug("kafka health ok", zap.Duration("latency", time.Since(start)))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:    ":" + cfg.MetricsPort,
		Handler: mux,
	}

	log.Info("metrics/health server starting", zap.String("addr", srv.Addr))

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("metrics server failed", zap.Error(err))
	}
}

// kafkaWriteTest envia uma mensagem de teste para o Kafka para validar a saúde da conexão
func kafkaWriteTest(ctx context.Context, w *kafka.Writer) error {
	payload := []byte(`{"ping":"ok"}`)
	return kafka.WriteJSON(ctx, w, "healthcheck", payload)
}
