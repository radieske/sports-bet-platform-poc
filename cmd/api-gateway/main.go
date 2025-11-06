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

	// Serve openapi-gateway.yaml em /swagger/openapi-gateway.yaml
	mux.HandleFunc("/swagger/openapi-gateway.yaml", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./docs/swagger-ui/openapi-gateway.yaml")
	})

	// Serve Swagger UI em /swagger/
	mux.HandleFunc("/swagger/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/swagger/" || r.URL.Path == "/swagger/index.html" {
			http.ServeFile(w, r, "./docs/swagger-ui/dist/index.html")
			return
		}
		http.StripPrefix("/swagger/", http.FileServer(http.Dir("./docs/swagger-ui/dist"))).ServeHTTP(w, r)
	})
	mux.Handle("/swagger", http.RedirectHandler("/swagger/", http.StatusMovedPermanently))

	addr := ":" + cfg.HTTPPort
	log.Info("api-gateway listening", zap.String("addr", addr))
	if err := http.ListenAndServe(addr, withCORS(mux)); err != nil && err != http.ErrServerClosed {
		log.Fatal("gateway failed", zap.Error(err))
	}
}

func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}
