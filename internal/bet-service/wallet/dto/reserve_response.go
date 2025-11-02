package dto

// ReserveResponse representa a resposta do endpoint de reserva do wallet-service.
type ReserveResponse struct {
	ReservationID string `json:"reservation_id"`
	Status        string `json:"status"`
}
