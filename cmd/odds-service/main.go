package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	svcCache "github.com/radieske/sports-bet-platform-poc/internal/odds-service/cache"
	httpapi "github.com/radieske/sports-bet-platform-poc/internal/odds-service/http"
	"github.com/radieske/sports-bet-platform-poc/internal/odds-service/repo"
	"github.com/radieske/sports-bet-platform-poc/internal/odds-service/ws"

	"github.com/radieske/sports-bet-platform-poc/internal/shared/cache"
	"github.com/radieske/sports-bet-platform-poc/internal/shared/config"
	"github.com/radieske/sports-bet-platform-poc/internal/shared/db"
	"github.com/radieske/sports-bet-platform-poc/internal/shared/kafka"
	"github.com/radieske/sports-bet-platform-poc/internal/shared/logger"
)

func main() {
	// Cria contexto base para shutdown controlado via sinal do SO
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()

	// Inicializa logger estruturado
	log, err := logger.New(cfg.ServiceName, cfg.Env)
	if err != nil {
		panic(fmt.Errorf("logger init: %w", err))
	}
	defer log.Sync()

	log.Info("starting service", zap.String("service", cfg.ServiceName), zap.String("env", cfg.Env))

	// Conexão com Postgres para leitura de odds
	pg, err := db.ConnectPostgres(cfg.PostgresDSN)
	if err != nil {
		log.Fatal("failed to connect postgres", zap.Error(err))
	}
	defer pg.Close()
	log.Info("postgres connected")

	// Conexão com Redis compartilhado
	redisClient, err := cache.ConnectRedis(cfg.RedisAddr)
	if err != nil {
		log.Fatal("failed to connect redis", zap.Error(err))
	}
	defer redisClient.Close()
	log.Info("redis connected")

	// Writer Kafka utilizado apenas para healthcheck
	writer := kafka.NewWriter(cfg.KafkaBrokers, "odds_updates")
	defer writer.Close()
	log.Info("kafka writer ready", zap.String("topic", "odds_updates"))

	// ========= Servidor de métricas e health check =========
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	metricsMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		hctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		// Valida dependências críticas
		if err := pg.PingContext(hctx); err != nil {
			http.Error(w, "postgres not healthy: "+err.Error(), http.StatusServiceUnavailable)
			return
		}

		if err := redisClient.Ping(hctx).Err(); err != nil {
			http.Error(w, "redis not healthy: "+err.Error(), http.StatusServiceUnavailable)
			return
		}

		kctx, kcancel := context.WithTimeout(hctx, 2*time.Second)
		defer kcancel()

		start := time.Now()
		if err := kafkaWriteTest(kctx, writer); err != nil {
			log.Warn("kafka health check failed", zap.Error(err), zap.Duration("latency", time.Since(start)))
			http.Error(w, "kafka not healthy: "+err.Error(), http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	metricsSrv := &http.Server{
		Addr:    ":" + cfg.MetricsPort,
		Handler: metricsMux,
	}

	// ========= Servidor principal (REST + WS) =========
	// Repositório de leitura e cache de odds
	readRepo := &repo.ReadRepo{DB: pg}
	oddsCache := svcCache.New(redisClient)
	api := &httpapi.API{ReadRepo: readRepo, Cache: oddsCache}

	// Hub WebSocket e inscrição no Redis Pub/Sub para broadcast de odds
	hub := ws.NewHub(func(r *http.Request) bool { return true /* TODO: restringir origem */ })
	ws.StartRedisSubscriber(ctx, redisClient, hub)

	appMux := http.NewServeMux()
	appMux.Handle("/", api.Router())            // REST: endpoints de consulta de odds
	appMux.HandleFunc("/ws/odds", hub.HandleWS) // WS: protocolo subscribe/unsubscribe

	appSrv := &http.Server{
		Addr:    ":" + cfg.HTTPPort,
		Handler: appMux,
	}

	// ========= Inicializa servidores =========
	go func() {
		log.Info("metrics/health server starting", zap.String("addr", metricsSrv.Addr))
		if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("metrics server failed", zap.Error(err))
		}
	}()

	go func() {
		log.Info("app server starting", zap.String("addr", appSrv.Addr))
		if err := appSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("app server failed", zap.Error(err))
		}
	}()

	// ========= Aguarda sinal de shutdown =========
	<-ctx.Done()
	log.Info("shutdown signal received")

	shCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = appSrv.Shutdown(shCtx)
	_ = metricsSrv.Shutdown(shCtx)

	log.Info("service stopped")
}

// kafkaWriteTest envia uma mensagem de teste para o Kafka para validar a saúde da conexão
func kafkaWriteTest(ctx context.Context, w *kafka.Writer) error {
	payload := []byte(`{"ping":"ok"}`)
	return kafka.WriteJSON(ctx, w, "healthcheck", payload)
}
