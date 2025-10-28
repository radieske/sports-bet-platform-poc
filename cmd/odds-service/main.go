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
	// carrega config
	cfg := config.Load()

	// inicia logger
	log, err := logger.New(cfg.ServiceName, cfg.Env)
	if err != nil {
		panic(fmt.Errorf("logger init: %w", err))
	}
	defer log.Sync()

	log.Info("starting service", zap.String("service", cfg.ServiceName), zap.String("env", cfg.Env))

	// conecta com db Postgres
	pg, err := db.ConnectPostgres(cfg.PostgresDSN)
	if err != nil {
		log.Fatal("failed to connect postgres", zap.Error(err))
	}
	defer pg.Close()
	log.Info("postgres connected")

	// conecta com cache Redis
	redisClient, err := cache.ConnectRedis(cfg.RedisAddr)
	if err != nil {
		log.Fatal("failed to connect redis", zap.Error(err))
	}
	defer redisClient.Close()
	log.Info("redis connected")

	// cria writer Kafka só pra validar conexão de início
	writer := kafka.NewWriter(cfg.KafkaBrokers, "odds_updates")
	defer writer.Close()
	log.Info("kafka writer ready", zap.String("topic", "odds_updates"))

	// sobe servidor de métricas e health
	mux := http.NewServeMux()

	// métricas
	mux.Handle("/metrics", promhttp.Handler())

	// healthz: valida dependências críticas
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		// ping postgres
		if err := pg.PingContext(ctx); err != nil {
			http.Error(w, "postgres not healthy: "+err.Error(), http.StatusServiceUnavailable)
			return
		}

		// ping redis
		if err := redisClient.Ping(ctx).Err(); err != nil {
			http.Error(w, "redis not healthy: "+err.Error(), http.StatusServiceUnavailable)
			return
		}

		// Kafka check (melhorado)
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

func kafkaWriteTest(ctx context.Context, w *kafka.Writer) error {
	payload := []byte(`{"ping":"ok"}`)
	return kafka.WriteJSON(ctx, w, "healthcheck", payload)
}
