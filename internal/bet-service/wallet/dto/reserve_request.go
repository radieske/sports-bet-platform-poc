package dto

// ReserveRequest representa o payload para reservar saldo no wallet-service.
type ReserveRequest struct {
	UserID      string `json:"userId"`
	AmountCents int64  `json:"amount_cents"`
	ExternalRef string `json:"external_ref"`
}
