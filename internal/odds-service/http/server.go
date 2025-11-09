package httpapi

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/radieske/sports-bet-platform-poc/internal/odds-service/cache"
	"github.com/radieske/sports-bet-platform-poc/internal/odds-service/dto"
	"github.com/radieske/sports-bet-platform-poc/internal/odds-service/repo"
)

// API expõe os endpoints REST de consulta de odds esportivas
// Utiliza um repositório de leitura (Postgres) e cache (Redis)
type API struct {
	ReadRepo *repo.ReadRepo // acesso ao banco de dados
	Cache    *cache.Cache   // cache de odds
}

// Router retorna o roteador HTTP com os endpoints REST
func (a *API) Router() http.Handler {
	r := chi.NewRouter()
	r.Get("/v1/events", a.listEvents)               // Lista eventos esportivos
	r.Get("/v1/events/{id}/markets", a.listMarkets) // Lista mercados de um evento
	r.Get("/v1/events/{id}/odds", a.getOdds)        // Lista odds de um evento
	return r
}

// writeJSON serializa a resposta em JSON e define o status HTTP
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// listEvents retorna todos os eventos esportivos disponíveis
func (a *API) listEvents(w http.ResponseWriter, r *http.Request) {
	ev, err := a.ReadRepo.ListEvents(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, ev)
}

// listMarkets retorna todos os mercados de um evento
func (a *API) listMarkets(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	mk, err := a.ReadRepo.ListMarkets(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, mk)
}

// getOdds retorna as odds de um evento, preferencialmente do cache
func (a *API) getOdds(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var fromCache []dto.Odds
	if ok, _ := a.Cache.GetOdds(r.Context(), id, &fromCache); ok {
		writeJSON(w, http.StatusOK, fromCache)
		return
	}

	od, err := a.ReadRepo.GetOddsByEvent(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	_ = a.Cache.SetOdds(r.Context(), id, od, 30*time.Second) // salva no cache por 30s
	writeJSON(w, http.StatusOK, od)
}
