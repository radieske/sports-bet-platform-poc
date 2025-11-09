package ws

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// Hub gerencia conexões WebSocket e assinaturas de eventos de odds
// subs: mapeia eventID para o conjunto de conexões inscritas
type Hub struct {
	upgrader websocket.Upgrader
	mu       sync.RWMutex
	// eventID -> set of connections
	subs map[string]map[*websocket.Conn]struct{}
}

// NewHub cria uma instância de Hub com política customizada de origem (CORS)
func NewHub(allowOrigin func(r *http.Request) bool) *Hub {
	return &Hub{
		upgrader: websocket.Upgrader{CheckOrigin: allowOrigin},
		subs:     make(map[string]map[*websocket.Conn]struct{}),
	}
}

// HandleWS gerencia o ciclo de vida de uma conexão WebSocket
// Permite subscribe/unsubscribe em eventos e responde a pings
// Cada cliente pode se inscrever em múltiplos eventIDs
func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	for {
		var msg ClientMsg
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}
		switch msg.Type {
		case "subscribe":
			h.mu.Lock()
			if _, ok := h.subs[msg.EventID]; !ok {
				h.subs[msg.EventID] = make(map[*websocket.Conn]struct{})
			}
			h.subs[msg.EventID][conn] = struct{}{}
			h.mu.Unlock()
		case "unsubscribe":
			h.mu.Lock()
			if m, ok := h.subs[msg.EventID]; ok {
				delete(m, conn)
				if len(m) == 0 {
					delete(h.subs, msg.EventID)
				}
			}
			h.mu.Unlock()
		case "ping":
			_ = conn.WriteJSON(map[string]string{"type": "pong"})
		}
	}
	// Remove a conexão de todas as assinaturas ao desconectar
	h.mu.Lock()
	for _, set := range h.subs {
		delete(set, conn)
	}
	h.mu.Unlock()
}

// Broadcast envia uma atualização de odds para todos os clientes inscritos no eventID correspondente
func (h *Hub) Broadcast(update OddsUpdate) {
	h.mu.RLock()
	conns := h.subs[update.EventID]
	h.mu.RUnlock()
	if len(conns) == 0 {
		return
	}

	b, _ := json.Marshal(update)
	for c := range conns {
		_ = c.WriteMessage(websocket.TextMessage, b)
	}
}
