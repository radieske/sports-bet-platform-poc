package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

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
	oddsURL := os.Getenv("ODDS_URL")
	if oddsURL == "" {
		oddsURL = "http://localhost:8080"
	}
	walletURL := os.Getenv("WALLET_URL")
	if walletURL == "" {
		walletURL = "http://localhost:8082"
	}
	betURL := os.Getenv("BET_URL")
	if betURL == "" {
		betURL = "http://localhost:8083"
	}
	odds := rp(oddsURL)
	wallet := rp(walletURL)
	bet := rp(betURL)

	mux := http.NewServeMux()

	// odds (ex.: /api/odds/* -> odds-service)
	mux.Handle("/api/odds/", http.StripPrefix("/api/odds", odds))

	// wallet (ex.: /api/wallet/* -> wallet-service)
	mux.Handle("/api/wallet/", http.StripPrefix("/api/wallet", wallet))

	// bets (ex.: /api/bets/* -> bet-service)
	mux.Handle("/api/bets/", http.StripPrefix("/api/bets", bet))

	// Serve openapi-gateway.yaml em /swagger/openapi-gateway.yaml e /docs/swagger-ui/openapi-gateway.yaml
	mux.HandleFunc("/swagger/openapi-gateway.yaml", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "/app/docs/swagger-ui/openapi-gateway.yaml")
	})
	mux.HandleFunc("/docs/swagger-ui/openapi-gateway.yaml", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "/app/docs/swagger-ui/openapi-gateway.yaml")
	})

	// Serve Swagger UI em /swagger/ e /docs/swagger-ui/
	mux.HandleFunc("/swagger/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/swagger/" || r.URL.Path == "/swagger/index.html" {
			http.ServeFile(w, r, "/app/docs/swagger-ui/dist/index.html")
			return
		}
		http.StripPrefix("/swagger/", http.FileServer(http.Dir("/app/docs/swagger-ui/dist"))).ServeHTTP(w, r)
	})
	mux.HandleFunc("/docs/swagger-ui/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/docs/swagger-ui/" || r.URL.Path == "/docs/swagger-ui/index.html" {
			http.ServeFile(w, r, "/app/docs/swagger-ui/dist/index.html")
			return
		}
		http.StripPrefix("/docs/swagger-ui/", http.FileServer(http.Dir("/app/docs/swagger-ui/dist"))).ServeHTTP(w, r)
	})
	mux.Handle("/swagger", http.RedirectHandler("/swagger/", http.StatusMovedPermanently))
	mux.Handle("/docs/swagger-ui", http.RedirectHandler("/docs/swagger-ui/", http.StatusMovedPermanently))

	addr := ":" + cfg.HTTPPort
	log.Info("api-gateway listening", zap.String("addr", addr))
	if err := http.ListenAndServe(addr, withCORS(mux)); err != nil && err != http.ErrServerClosed {
		log.Fatal("gateway failed", zap.Error(err))
	}
}

func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}
