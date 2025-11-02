package dto

type WalletResponse struct {
	UserID       string `json:"userId"`
	WalletID     string `json:"walletId"`
	BalanceCents int64  `json:"balance_cents"`
}

type ReservationResponse struct {
	ReservationID string `json:"reservation_id"`
	Status        string `json:"status"`
}
