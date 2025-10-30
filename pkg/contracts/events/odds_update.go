package events

import "time"

// Evento publicado no tópico "odds_updates"
type Odds struct {
	Home float64 `json:"home"`
	Draw float64 `json:"draw"`
	Away float64 `json:"away"`
}

type OddsUpdate struct {
	EventID   string    `json:"event_id"`
	HomeTeam  string    `json:"home_team"`
	AwayTeam  string    `json:"away_team"`
	Market    string    `json:"market"` // "1x2"
	Odds      Odds      `json:"odds"`
	UpdatedAt time.Time `json:"updated_at"`
	Source    string    `json:"source"`  // "supplier-simulator"
	Version   int       `json:"version"` // incrementado a cada atualização
}
