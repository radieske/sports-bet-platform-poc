package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	kafkago "github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	bhttp "github.com/radieske/sports-bet-platform-poc/internal/bet-service/http"
	"github.com/radieske/sports-bet-platform-poc/internal/bet-service/odds"
	kpub "github.com/radieske/sports-bet-platform-poc/internal/bet-service/producer"
	"github.com/radieske/sports-bet-platform-poc/internal/bet-service/repo"
	"github.com/radieske/sports-bet-platform-poc/internal/bet-service/wallet"
	"github.com/radieske/sports-bet-platform-poc/internal/shared/config"
	"github.com/radieske/sports-bet-platform-poc/internal/shared/db"
	"github.com/radieske/sports-bet-platform-poc/internal/shared/logger"
)

func main() {
	cfg := config.Load()
	log, _ := logger.New(cfg.ServiceName, cfg.Env)
	defer log.Sync()

	// Postgres
	pg, err := db.ConnectPostgres(cfg.PostgresDSN)
	if err != nil {
		log.Fatal("pg", zap.Error(err))
	}
	defer pg.Close()

	// Redis
	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatal("redis", zap.Error(err))
	}

	// Kafka writer (topic bet_placed)
	writer := kafkago.NewWriter(kafkago.WriterConfig{
		Brokers:  strings.Split(cfg.KafkaBrokers, ","),
		Topic:    cfg.TopicBetPlaced,
		Balancer: &kafkago.LeastBytes{},
	})
	defer writer.Close()

	// deps
	repository := repo.NewPostgres(pg)
	ov := odds.NewValidator(rdb)

	walletURL := os.Getenv("WALLET_URL")
	if walletURL == "" {
		walletURL = "http://localhost:8082"
	}
	wcli := wallet.New(walletURL) // wallet-service
	publ := kpub.NewKafkaPublisher(writer, cfg.TopicBetPlaced)

	// HTTP público
	api := bhttp.NewServer(log, repository, ov, wcli, publ)
	apiSrv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.HTTPPort),
		Handler: api.Router(),
	}

	// metrics/health
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	metricsMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := pg.PingContext(r.Context()); err != nil {
			http.Error(w, "pg", http.StatusServiceUnavailable)
			return
		}
		if err := rdb.Ping(r.Context()).Err(); err != nil {
			http.Error(w, "redis", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	go func() {
		addr := fmt.Sprintf(":%s", cfg.MetricsPort)
		log.Info("metrics/health", zap.String("addr", addr))
		_ = http.ListenAndServe(addr, metricsMux)
	}()

	log.Info("bet-service listening", zap.String("addr", fmt.Sprintf(":%s", cfg.HTTPPort)))
	if err := apiSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("api", zap.Error(err))
	}
}
