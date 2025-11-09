package ws

// ClientMsg representa uma mensagem recebida do cliente WebSocket
// Type: subscribe | unsubscribe | ping
// EventID: obrigatório para subscribe/unsubscribe
type ClientMsg struct {
	Type    string `json:"type"`    // subscribe | unsubscribe | ping
	EventID string `json:"eventId"` // requerido em subscribe/unsubscribe
}

// OddsUpdate representa uma atualização de odds enviada para clientes WebSocket
type OddsUpdate struct {
	EventID string      `json:"eventId"`
	Payload interface{} `json:"payload"`
}
