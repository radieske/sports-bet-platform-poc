package main

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/radieske/sports-bet-platform-poc/internal/shared/config"
	"github.com/radieske/sports-bet-platform-poc/internal/shared/db"
	"github.com/radieske/sports-bet-platform-poc/internal/shared/logger"
	whttp "github.com/radieske/sports-bet-platform-poc/internal/wallet-service/http"
	wrepo "github.com/radieske/sports-bet-platform-poc/internal/wallet-service/repo"
)

func main() {
	cfg := config.Load()

	// Inicializa logger estruturado
	log, err := logger.New("wallet-service", cfg.Env)
	if err != nil {
		panic(err)
	}
	defer log.Sync()
	log.Info("starting service", zap.String("service", "wallet-service"), zap.String("env", cfg.Env))

	// Conexão com Postgres para operações de carteira
	pg, err := db.ConnectPostgres(cfg.PostgresDSN)
	if err != nil {
		log.Fatal("postgres connect", zap.Error(err))
	}
	defer pg.Close()

	// Instancia repositório e servidor HTTP da wallet
	repo := wrepo.NewPostgres(pg)
	api := whttp.NewServer(log, repo)

	// Servidor HTTP público (API de wallet)
	apiSrv := &http.Server{
		Addr:    ":" + cfg.HTTPPort, // ex: 8082
		Handler: api.Router(),
	}

	// Servidor de métricas e health check
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	metricsMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
		defer cancel()
		if err := pg.PingContext(ctx); err != nil {
			http.Error(w, "pg", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	metricsSrv := &http.Server{Addr: ":" + cfg.MetricsPort, Handler: metricsMux} // ex: 9098

	// Inicia servidor de métricas/health em goroutine separada
	go func() {
		log.Info("metrics/health listening", zap.String("addr", metricsSrv.Addr))
		if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("metrics srv", zap.Error(err))
		}
	}()

	// Inicia servidor principal da API de wallet
	log.Info("api listening", zap.String("addr", apiSrv.Addr))
	if err := apiSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("api srv", zap.Error(err))
	}
}
