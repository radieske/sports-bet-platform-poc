package dto

type PlaceBetResponse struct {
	BetID      string `json:"betId"`
	Status     string `json:"status"` // PENDING_CONFIRMATION
	NewBalance *int64 `json:"new_balance,omitempty"`
	Message    string `json:"message,omitempty"`
}

type BetStatusResponse struct {
	BetID  string `json:"betId"`
	Status string `json:"status"`
}
