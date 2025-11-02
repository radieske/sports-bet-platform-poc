package dto

type DepositRequest struct {
	UserID      string `json:"userId"`
	AmountCents int64  `json:"amount_cents"`
	ExternalRef string `json:"external_ref,omitempty"` // opcional p/ idempotência simples
}

type ReserveRequest struct {
	UserID      string `json:"userId"`
	AmountCents int64  `json:"amount_cents"`
	ExternalRef string `json:"external_ref"` // ex: betId
}

type CommitRequest struct {
	UserID      string `json:"userId"`
	ExternalRef string `json:"external_ref"`
}

type RefundRequest struct {
	UserID      string `json:"userId"`
	ExternalRef string `json:"external_ref"`
}
