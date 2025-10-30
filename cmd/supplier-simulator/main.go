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

// Processo principal do simulador de fornecedor de odds
func main() {
	cfg := config.Load()
	log, err := logger.New("supplier-simulator", cfg.Env)
	if err != nil {
		panic(err)
	}
	defer log.Sync()
	rand.Seed(time.Now().UnixNano())

	prometheus.MustRegister(wsConnections, wsMessagesSent)

	h := newHub(log)

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
				u.Source = "supplier-simulator"
				u.Version = version
				updates[i] = u
			}
			version++
			// Envia um JSON por partida simulada
			for _, up := range updates {
				h.broadcast(up)
			}
		}
	}()

	mux := http.NewServeMux()

	// Endpoint WebSocket para clientes receberem odds simuladas
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
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
			_ = conn.SetReadDeadline(time.Time{}) // Sem deadline para leitura
			for {
				// Lê e descarta mensagens do cliente para manter o socket limpo
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
			}
		}()
	})

	// Endpoint de health check
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Endpoint de métricas Prometheus
	mux.Handle("/metrics", promhttp.Handler())

	addr := ":8081"
	log.Info("supplier simulator (WS) running", zap.String("addr", addr), zap.String("path", "/ws"))
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal("server error", zap.Error(err))
	}
}

// Gera um número float64 aleatório entre min e max
func rnd(min, max float64) float64 {
	return (rand.Float64() * (max - min)) + min
}
