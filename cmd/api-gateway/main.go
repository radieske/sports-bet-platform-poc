package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"go.uber.org/zap"

	"github.com/radieske/sports-bet-platform-poc/internal/shared/config"
	"github.com/radieske/sports-bet-platform-poc/internal/shared/logger"
)

func rp(to string) *httputil.ReverseProxy {
	u, _ := url.Parse(to)
	return httputil.NewSingleHostReverseProxy(u)
}

func main() {
	cfg := config.Load()
	log, _ := logger.New(cfg.ServiceName, cfg.Env)
	defer log.Sync()

	// targets
	odds := rp("http://localhost:8080")
	wallet := rp("http://localhost:8082")
	bet := rp("http://localhost:8083")

	mux := http.NewServeMux()

	// odds (ex.: /api/odds/* -> odds-service)
	mux.Handle("/api/odds/", http.StripPrefix("/api/odds", odds))

	// wallet (ex.: /api/wallet/* -> wallet-service)
	mux.Handle("/api/wallet/", http.StripPrefix("/api/wallet", wallet))

	// bets (ex.: /api/bets/* -> bet-service)
	mux.Handle("/api/bets/", http.StripPrefix("/api/bets", bet))

	addr := ":" + cfg.HTTPPort
	log.Info("api-gateway listening", zap.String("addr", addr))
	if err := http.ListenAndServe(addr, mux); err != nil && err != http.ErrServerClosed {
		log.Fatal("gateway failed", zap.Error(err))
	}
}
