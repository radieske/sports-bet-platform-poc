package dto

type PlaceBetRequest struct {
	UserID     string  `json:"userId"`
	EventID    string  `json:"eventId"`
	Market     string  `json:"market"`    // ex: "MATCH_ODDS"
	Selection  string  `json:"selection"` // "home" | "draw" | "away"
	StakeCents int64   `json:"stake_cents"`
	OddValue   float64 `json:"odd_value"` // odd que o cliente viu
}
