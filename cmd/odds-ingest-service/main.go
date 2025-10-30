package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/radieske/sports-bet-platform-poc/internal/odds-ingest/publisher"
	"github.com/radieske/sports-bet-platform-poc/internal/odds-ingest/service"
	"github.com/radieske/sports-bet-platform-poc/internal/shared/config"
	"github.com/radieske/sports-bet-platform-poc/internal/shared/logger"
)

func main() {
	cfg := config.Load()
	log, err := logger.New("odds-ingest-service", cfg.Env)
	if err != nil {
		panic(err)
	}
	defer log.Sync()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Kafka Publisher
	pub := publisher.NewKafkaPublisher(
		[]string{cfg.KafkaBrokers},
		"odds_updates",
		log,
	)
	defer pub.Close()

	// WS Client
	wsClient := &service.WSClient{
		URL:       "ws://localhost:8081/ws", // ajustar pra nome do container em Docker
		Log:       log,
		Publisher: pub,
	}
	go wsClient.Start(ctx)

	// Metrics e health
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		})
		log.Info("metrics/health listening", zap.String("addr", ":9096"))
		_ = http.ListenAndServe(":9096", mux)
	}()

	// graceful shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Info("shutdown signal received")
	cancel()
	time.Sleep(2 * time.Second)
}
