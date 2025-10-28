package metrics

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type HealthFunc func(ctx context.Context) error

// StartMetricsServer sobe um servidor HTTP leve só pra /metrics e /healthz.
// executável em numa goroutine no main de cada serviço.
func StartMetricsServer(port string, healthFn HealthFunc) *http.Server {
	mux := http.NewServeMux()

	mux.Handle("/metrics", promhttp.Handler())

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
		defer cancel()

		if err := healthFn(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(fmt.Sprintf("unhealthy: %v", err)))
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		_ = srv.ListenAndServe()
	}()

	return srv
}
