package wallet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	walletdto "github.com/radieske/sports-bet-platform-poc/internal/bet-service/wallet/dto"
)

type Client struct {
	BaseURL string
	HTTP    *http.Client
}

func New(base string) *Client {
	return &Client{
		BaseURL: base,
		HTTP:    &http.Client{Timeout: 2 * time.Second},
	}
}

func (c *Client) Reserve(ctx context.Context, userID string, cents int64, externalRef string) (string, error) {
	body, _ := json.Marshal(walletdto.ReserveRequest{UserID: userID, AmountCents: cents, ExternalRef: externalRef})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/wallet/reserve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return "", fmt.Errorf("wallet reserve http %d", res.StatusCode)
	}
	var out walletdto.ReserveResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.ReservationID, nil
}
