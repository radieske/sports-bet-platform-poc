package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/radieske/sports-bet-platform-poc/internal/shared/config"
	"github.com/radieske/sports-bet-platform-poc/internal/shared/logger"
	"github.com/radieske/sports-bet-platform-poc/pkg/contracts/events"

	sdto "github.com/radieske/sports-bet-platform-poc/internal/supplier-simulator/dto"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}

	// Catálogo fixo de partidas simuladas para geração de odds
	eventCatalog = []events.OddsUpdate{
		{EventID: "MATCH_001", HomeTeam: "Flamengo", AwayTeam: "Palmeiras", Market: "1x2"},
		{EventID: "MATCH_002", HomeTeam: "Grêmio", AwayTeam: "Internacional", Market: "1x2"},
		{EventID: "MATCH_003", HomeTeam: "Corinthians", AwayTeam: "Santos", Market: "1x2"},
		{EventID: "MATCH_004", HomeTeam: "São Paulo", AwayTeam: "Vasco", Market: "1x2"},
	}

	// Métricas Prometheus para monitoramento de conexões e mensagens
	wsConnections = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "supplier_ws_connections",
		Help: "Clientes WebSocket conectados",
	})
	wsMessagesSent = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "supplier_ws_messages_sent_total",
		Help: "Total de mensagens WS enviadas",
	})
)

// Representa uma conexão de cliente WebSocket
// id: identificador único da conexão
// conn: ponteiro para a conexão WebSocket
type clientConn struct {
	id   string
	conn *websocket.Conn
}

// Estrutura responsável por gerenciar os clientes conectados via WebSocket
// e realizar broadcast de mensagens para todos eles.
type hub struct {
	mu      sync.RWMutex
	clients map[string]*clientConn
	log     *zap.Logger
}

// Cria uma nova instância de hub para gerenciar conexões
func newHub(log *zap.Logger) *hub {
	return &hub{
		clients: make(map[string]*clientConn),
		log:     log,
	}
}

// Adiciona um novo cliente ao hub e incrementa a métrica de conexões
func (h *hub) add(c *clientConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c.id] = c
	wsConnections.Inc()
	h.log.Info("ws client connected", zap.String("client_id", c.id))
}

// Remove um cliente do hub e decrementa a métrica de conexões
func (h *hub) remove(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[id]; ok {
		delete(h.clients, id)
		wsConnections.Dec()
		h.log.Info("ws client disconnected", zap.String("client_id", id))
	}
}

// Envia uma mensagem para todos os clientes conectados
func (h *hub) broadcast(v any) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	msg, _ := json.Marshal(v)
	for id, c := range h.clients {
		c.conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			h.log.Warn("ws write failed", zap.String("client_id", id), zap.Error(err))
			_ = c.conn.Close()
		} else {
			wsMessagesSent.Inc()
		}
	}
}

// Server estrutura principal do serviço
type server struct {
	log *zap.Logger
}

func newServer(log *zap.Logger) *server { return &server{log: log} }

// Handler para confirmar aposta (mock)
func (s *server) confirmHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req sdto.ConfirmReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	ok := rand.Intn(100) < 80 // 80% sucesso

	resp := sdto.ConfirmResp{
		Status:      sdto.StatusConfirmed,
		ProviderRef: "SUP-" + safePrefix(req.BetID, 8),
	}
	if !ok {
		resp.Status = sdto.StatusRejected
		resp.Reason = "supplier_reject_mock"
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// evita panic se o BetID for menor que 8 caracteres
func safePrefix(s string, n int) string {
	if len(s) < n {
		return s
	}
	return s[:n]
}

// gera número aleatório entre min e max
func rnd(min, max float64) float64 {
	return (rand.Float64() * (max - min)) + min
}

func main() {
	cfg := config.Load()
	log, err := logger.New(cfg.ServiceName, cfg.Env)
	if err != nil {
		panic(err)
	}
	defer log.Sync()
	rand.Seed(time.Now().UnixNano())

	prometheus.MustRegister(wsConnections, wsMessagesSent)

	h := newHub(log)
	s := newServer(log)

	// Gera e envia odds simuladas para todos os clientes conectados a cada 3 segundos
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		version := 1
		for range ticker.C {
			updates := make([]events.OddsUpdate, len(eventCatalog))
			for i := range eventCatalog {
				u := eventCatalog[i]
				u.Odds = events.Odds{
					Home: rnd(1.40, 3.50),
					Draw: rnd(2.50, 4.50),
					Away: rnd(2.00, 5.00),
				}
				u.UpdatedAt = time.Now().UTC()
				u.Source = cfg.ServiceName
				u.Version = version
				updates[i] = u
			}
			version++
			for _, up := range updates {
				h.broadcast(up)
			}
		}
	}()

	// ==== MUX PÚBLICO (HTTP principal): /ws e /supplier/confirm
	appMux := http.NewServeMux()

	appMux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Warn("ws upgrade failed", zap.Error(err))
			return
		}
		id := fmt.Sprintf("%d", time.Now().UnixNano())
		c := &clientConn{id: id, conn: conn}
		h.add(c)

		// Goroutine para manter a conexão viva e remover cliente ao desconectar
		go func() {
			defer func() {
				h.remove(id)
				_ = conn.Close()
			}()
			_ = conn.SetReadDeadline(time.Time{})
			for {
				// Lê e descarta mensagens do cliente para manter o socket limpo
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
			}
		}()
	})

	appMux.HandleFunc("/supplier/confirm", s.confirmHandler)

	// ==== MUX DE MÉTRICAS (/healthz, /metrics)
	metricsMux := http.NewServeMux()
	metricsMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	metricsMux.Handle("/metrics", promhttp.Handler())

	// Servidor de métricas em goroutine
	go func() {
		metricsAddr := fmt.Sprintf(":%s", cfg.MetricsPort)
		log.Info("supplier simulator (metrics) running",
			zap.String("addr", metricsAddr),
			zap.String("paths", "/healthz,/metrics"),
		)
		if err := http.ListenAndServe(metricsAddr, metricsMux); err != nil {
			log.Fatal("metrics server error", zap.Error(err))
		}
	}()

	// Servidor público (WS + supplier confirm)
	publicAddr := fmt.Sprintf(":%s", cfg.HTTPPort)
	log.Info("supplier simulator (public) running",
		zap.String("addr", publicAddr),
		zap.String("paths", "/ws,/supplier/confirm"),
	)
	if err := http.ListenAndServe(publicAddr, appMux); err != nil {
		log.Fatal("public server error", zap.Error(err))
	}
}
