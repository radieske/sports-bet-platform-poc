package events

import "time"

// Evento emitido pelo bet-confirmation-worker após processar uma aposta.
type BetConfirmed struct {
	BetID       string    `json:"betId"`
	UserID      string    `json:"userId"`
	Status      string    `json:"status"` // "CONFIRMED" | "REJECTED"
	Reason      string    `json:"reason,omitempty"`
	ProviderRef string    `json:"providerRef,omitempty"`
	Ts          time.Time `json:"ts"`
}
