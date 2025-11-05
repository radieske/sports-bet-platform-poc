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

	"github.com/radieske/sports-bet-platform-poc/internal/odds-ingest/publisher"
	"github.com/radieske/sports-bet-platform-poc/internal/odds-ingest/service"
	"github.com/radieske/sports-bet-platform-poc/internal/shared/config"
	"github.com/radieske/sports-bet-platform-poc/internal/shared/logger"
)

func main() {
	cfg := config.Load()
	log, err := logger.New(cfg.ServiceName, cfg.Env)
	if err != nil {
		panic(err)
	}
	defer log.Sync()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Kafka Publisher
	pub := publisher.NewKafkaPublisher(
		[]string{cfg.KafkaBrokers},
		cfg.TopicOddsUpdates,
		log,
	)
	defer pub.Close()

	// WS Client
	wsClient := &service.WSClient{
		URL:       cfg.SupplierWSURL,
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

		addr := fmt.Sprintf(":%s", cfg.MetricsPort)
		log.Info("metrics/health listening", zap.String("addr", addr))
		_ = http.ListenAndServe(addr, mux)
	}()

	// graceful shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Info("shutdown signal received")
	cancel()
	time.Sleep(2 * time.Second)
}
