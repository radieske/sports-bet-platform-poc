package ws

type ClientMsg struct {
	Type    string `json:"type"`    // subscribe | unsubscribe | ping
	EventID string `json:"eventId"` // requerido em subscribe/unsubscribe
}

type OddsUpdate struct {
	EventID string      `json:"eventId"`
	Payload interface{} `json:"payload"`
}
