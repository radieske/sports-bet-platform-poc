package events

type BetPlaced struct {
	BetID       string  `json:"bet_id"`
	UserID      string  `json:"user_id"`
	EventID     string  `json:"event_id"`
	Market      string  `json:"market"`
	Selection   string  `json:"selection"`
	StakeCents  int64   `json:"stake_cents"`
	OddValue    float64 `json:"odd_value"`
	ReservedRef string  `json:"reserved_ref"` // external_ref usado na reserva da carteira (betID)
	TsUnixMs    int64   `json:"ts_unix_ms"`
}
