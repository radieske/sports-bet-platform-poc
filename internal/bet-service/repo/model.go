package repo

import "time"

// Bet é o modelo persistido no Postgres.
type Bet struct {
	ID         string
	UserID     string
	EventID    string
	Market     string
	Selection  string
	StakeCents int64
	OddValue   float64
	Status     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
