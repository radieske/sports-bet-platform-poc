package dto

type Event struct {
	EventID  string `json:"eventId"`
	HomeTeam string `json:"homeTeam"`
	AwayTeam string `json:"awayTeam"`
}

type Market struct {
	Market string `json:"market"`
}

type Odds struct {
	EventID   string  `json:"eventId"`
	Market    string  `json:"market"`
	HomeOdd   float64 `json:"homeOdd"`
	DrawOdd   float64 `json:"drawOdd"`
	AwayOdd   float64 `json:"awayOdd"`
	Version   int     `json:"version"`
	UpdatedAt string  `json:"updatedAt"`
}
