package http

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"

	"go.uber.org/zap"

	"github.com/radieske/sports-bet-platform-poc/internal/wallet-service/dto"
)

// Repo define a interface de operações de carteira usadas pelo handler HTTP
type Repo interface {
	GetOrCreateWallet(ctx context.Context, userID string) (walletID string, balance int64, err error)
	Deposit(ctx context.Context, userID string, amount int64, externalRef string) (walletID string, newBalance int64, err error)
	Reserve(ctx context.Context, userID string, amount int64, externalRef string) (reservationID string, err error)
	Commit(ctx context.Context, userID, externalRef string) error
	Refund(ctx context.Context, userID, externalRef string) error
}

// Server expõe endpoints HTTP para operações de carteira (wallet)
type Server struct {
	log  *zap.Logger
	repo Repo
}

// NewServer instancia o servidor HTTP de wallet
func NewServer(log *zap.Logger, repo Repo) *Server { return &Server{log: log, repo: repo} }

// Router retorna o mux HTTP com as rotas da API de wallet
func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/wallet", s.getWallet)       // GET ?userId=...
	mux.HandleFunc("/wallet/deposit", s.deposit) // POST
	mux.HandleFunc("/wallet/reserve", s.reserve) // POST
	mux.HandleFunc("/wallet/commit", s.commit)   // POST
	mux.HandleFunc("/wallet/refund", s.refund)   // POST
	return mux
}

// getWallet retorna (ou cria) a carteira e saldo do usuário
func (s *Server) getWallet(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		http.Error(w, "userId required", http.StatusBadRequest)
		return
	}
	walletID, bal, err := s.repo.GetOrCreateWallet(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, dto.WalletResponse{UserID: userID, WalletID: walletID, BalanceCents: bal})
}

// deposit adiciona saldo à carteira do usuário
func (s *Server) deposit(w http.ResponseWriter, r *http.Request) {
	var req dto.DepositRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if req.UserID == "" || req.AmountCents <= 0 {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	walletID, bal, err := s.repo.Deposit(r.Context(), req.UserID, req.AmountCents, req.ExternalRef)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, dto.WalletResponse{UserID: req.UserID, WalletID: walletID, BalanceCents: bal})
}

// reserve cria uma reserva de saldo (bloqueio) para o usuário
func (s *Server) reserve(w http.ResponseWriter, r *http.Request) {
	var req dto.ReserveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if req.UserID == "" || req.AmountCents <= 0 || req.ExternalRef == "" {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	resID, err := s.repo.Reserve(r.Context(), req.UserID, req.AmountCents, req.ExternalRef)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "wallet not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	writeJSON(w, dto.ReservationResponse{ReservationID: resID, Status: "PENDING"})
}

// commit efetiva uma reserva de saldo
func (s *Server) commit(w http.ResponseWriter, r *http.Request) {
	var req dto.CommitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if req.UserID == "" || req.ExternalRef == "" {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	if err := s.repo.Commit(r.Context(), req.UserID, req.ExternalRef); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"COMMITTED"}`))
}

// refund desfaz uma reserva de saldo, devolvendo o valor ao usuário
func (s *Server) refund(w http.ResponseWriter, r *http.Request) {
	var req dto.RefundRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if req.UserID == "" || req.ExternalRef == "" {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	if err := s.repo.Refund(r.Context(), req.UserID, req.ExternalRef); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"REFUNDED"}`))
}

// writeJSON serializa e envia resposta JSON
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
