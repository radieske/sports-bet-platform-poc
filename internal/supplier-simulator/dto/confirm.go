package dto

type ConfirmReq struct {
	BetID      string  `json:"betId"`
	UserID     string  `json:"userId"`
	EventID    string  `json:"eventId"`
	StakeCents int64   `json:"stake_cents"`
	OddValue   float64 `json:"odd_value"`
}

type ConfirmResp struct {
	Status      string `json:"status"` // CONFIRMED | REJECTED
	ProviderRef string `json:"providerRef"`
	Reason      string `json:"reason,omitempty"`
}

const (
	StatusConfirmed = "CONFIRMED"
	StatusRejected  = "REJECTED"
)
