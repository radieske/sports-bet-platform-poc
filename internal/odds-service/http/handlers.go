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

type API struct {
	ReadRepo *repo.ReadRepo
	Cache    *cache.Cache
}

func (a *API) Router() http.Handler {
	r := chi.NewRouter()
	r.Get("/v1/events", a.listEvents)
	r.Get("/v1/events/{id}/markets", a.listMarkets)
	r.Get("/v1/events/{id}/odds", a.getOdds)
	return r
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (a *API) listEvents(w http.ResponseWriter, r *http.Request) {
	ev, err := a.ReadRepo.ListEvents(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, ev)
}

func (a *API) listMarkets(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	mk, err := a.ReadRepo.ListMarkets(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, mk)
}

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

	_ = a.Cache.SetOdds(r.Context(), id, od, 30*time.Second)
	writeJSON(w, http.StatusOK, od)
}
