package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"go.uber.org/zap"

	"github.com/radieske/sports-bet-platform-poc/internal/bet-service/dto"
	"github.com/radieske/sports-bet-platform-poc/internal/bet-service/odds"
	"github.com/radieske/sports-bet-platform-poc/internal/bet-service/repo"
	"github.com/radieske/sports-bet-platform-poc/internal/bet-service/wallet"
	"github.com/radieske/sports-bet-platform-poc/pkg/contracts/events"
)

type Server struct {
	log  *zap.Logger
	repo *repo.Postgres
	odds *odds.Validator
	wcli *wallet.Client
	publ interface {
		PublishBetPlaced(context.Context, events.BetPlaced) error
	}
}

func NewServer(log *zap.Logger, r *repo.Postgres, v *odds.Validator, w *wallet.Client, p interface {
	PublishBetPlaced(context.Context, events.BetPlaced) error
}) *Server {
	return &Server{log: log, repo: r, odds: v, wcli: w, publ: p}
}

func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/bets", s.placeBet)      // POST
	mux.HandleFunc("/bets/", s.getBetStatus) // GET /bets/{id}
	return mux
}

func (s *Server) placeBet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req dto.PlaceBetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if req.UserID == "" || req.EventID == "" || req.Market == "" || req.Selection == "" || req.StakeCents <= 0 || req.OddValue <= 0 {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	// 1) Valida odd atual no cache
	curOddStr, err := s.odds.CurrentOdd(r.Context(), req.EventID, req.Market, req.Selection)
	if err == nil {
		// compara como string simples; se quiser tolerância, parse float e compare delta
		if curOddStr != "" {
			// se divergir muito, retorne 409 e a odd corrente
			if curOddStr != strconv.FormatFloat(req.OddValue, 'f', -1, 64) {
				http.Error(w, "odd changed; current="+curOddStr, http.StatusConflict)
				return
			}
		}
	}

	// 2) Cria aposta local PENDING
	betID, err := s.repo.CreatePending(r.Context(), &repo.Bet{
		UserID:     req.UserID,
		EventID:    req.EventID,
		Market:     req.Market,
		Selection:  req.Selection,
		StakeCents: req.StakeCents,
		OddValue:   req.OddValue,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 3) Reserva saldo via wallet (external_ref = betID)
	if _, err := s.wcli.Reserve(r.Context(), req.UserID, req.StakeCents, betID); err != nil {
		http.Error(w, "wallet reserve failed", http.StatusConflict)
		return
	}

	// 4) Publica evento bet_placed
	_ = s.publ.PublishBetPlaced(r.Context(), events.BetPlaced{
		BetID:       betID,
		UserID:      req.UserID,
		EventID:     req.EventID,
		Market:      req.Market,
		Selection:   req.Selection,
		StakeCents:  req.StakeCents,
		OddValue:    req.OddValue,
		ReservedRef: betID,
	})

	writeJSON(w, dto.PlaceBetResponse{
		BetID:  betID,
		Status: "PENDING_CONFIRMATION",
	})
}

func (s *Server) getBetStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// path: /bets/{id}
	id := r.URL.Path[len("/bets/"):]
	if id == "" {
		http.Error(w, "betId required", http.StatusBadRequest)
		return
	}

	st, err := s.repo.GetStatus(r.Context(), id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	writeJSON(w, dto.BetStatusResponse{BetID: id, Status: st})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
