package dto

type BetPlaced struct {
	BetID       string  `json:"betId"`
	UserID      string  `json:"userId"`
	EventID     string  `json:"eventId"`
	Market      string  `json:"market"`
	Selection   string  `json:"selection"`
	StakeCents  int64   `json:"stakeCents"`
	OddValue    float64 `json:"oddValue"`
	ReservedRef string  `json:"reservedRef"`
	TsUnixMs    int64   `json:"tsUnixMs"`
}

type SupplierConfirmResp struct {
	Status      string `json:"status"`
	ProviderRef string `json:"providerRef"`
	Reason      string `json:"reason,omitempty"`
}
