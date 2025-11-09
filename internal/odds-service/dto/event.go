package dto

// Event representa um evento esportivo (ex: partida de futebol)
type Event struct {
	EventID  string `json:"eventId"`
	HomeTeam string `json:"homeTeam"`
	AwayTeam string `json:"awayTeam"`
}

// Market representa um mercado de aposta (ex: resultado final)
type Market struct {
	Market string `json:"market"`
}

// Odds representa as odds de um mercado para um evento esportivo
type Odds struct {
	EventID   string  `json:"eventId"`
	Market    string  `json:"market"`
	HomeOdd   float64 `json:"homeOdd"`
	DrawOdd   float64 `json:"drawOdd"`
	AwayOdd   float64 `json:"awayOdd"`
	Version   int     `json:"version"`
	UpdatedAt string  `json:"updatedAt"`
}
