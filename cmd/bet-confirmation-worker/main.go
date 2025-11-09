package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	kafkago "github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"github.com/radieske/sports-bet-platform-poc/internal/shared/config"
	"github.com/radieske/sports-bet-platform-poc/internal/shared/db"
	"github.com/radieske/sports-bet-platform-poc/internal/shared/kafka"
	"github.com/radieske/sports-bet-platform-poc/internal/shared/logger"
	ev "github.com/radieske/sports-bet-platform-poc/pkg/contracts/events"
)

type betPlaced struct {
	BetID      string  `json:"betId"`
	UserID     string  `json:"userId"`
	EventID    string  `json:"eventId"`
	Market     string  `json:"market"`
	Selection  string  `json:"selection"`
	StakeCents int64   `json:"stake_cents"`
	OddValue   float64 `json:"odd_value"`
	ReservedID string  `json:"reserved_id,omitempty"`
	CreatedAt  int64   `json:"created_at,omitempty"`
}

type supplierConfirmResp struct {
	Status      string `json:"status"`
	ProviderRef string `json:"providerRef"`
	Reason      string `json:"reason,omitempty"`
}

func main() {
	cfg := config.Load()
	log, err := logger.New(cfg.ServiceName, cfg.Env)
	if err != nil {
		panic(err)
	}
	defer log.Sync()

	pg, err := db.ConnectPostgres(cfg.PostgresDSN)
	if err != nil {
		log.Fatal("pg connect", zap.Error(err))
	}
	defer pg.Close()

	// Dialer do Kafka com timeouts e identificação do cliente.
	kDialer := &kafkago.Dialer{
		Timeout:   10 * time.Second,
		DualStack: true,
		ClientID:  cfg.ServiceName,
	}

	// Consumer do Kafka para o tópico bet_placed usando brokers vindos da config.
	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:         strings.Split(cfg.KafkaBrokers, ","),
		GroupID:         "bet-confirmation",
		Topic:           cfg.TopicBetPlaced,
		MinBytes:        1e3,
		MaxBytes:        10e6,
		MaxWait:         500 * time.Millisecond,
		StartOffset:     kafkago.FirstOffset,
		ReadLagInterval: 2 * time.Second,
		Dialer:          kDialer,
	})
	defer reader.Close()

	// Producer para bet_confirmed e, opcionalmente, DLQ para bet_placed.
	confirmedWriter := kafka.NewWriter(cfg.KafkaBrokers, cfg.TopicBetConfirmed)
	defer confirmedWriter.Close()

	var dlqWriter *kafkago.Writer
	if cfg.TopicBetPlacedDLQ != "" {
		dlqWriter = kafka.NewWriter(cfg.KafkaBrokers, cfg.TopicBetPlacedDLQ)
		defer dlqWriter.Close()
	}

	// Servidor HTTP para métricas e health.
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()
			if err := pg.PingContext(ctx); err != nil {
				http.Error(w, "pg", http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		})
		addr := ":" + cfg.MetricsPort
		log.Info("metrics/health", zap.String("addr", addr))
		_ = http.ListenAndServe(addr, mux)
	}()

	log.Info("bet-confirmation-worker started",
		zap.String("consume", cfg.TopicBetPlaced),
		zap.String("publish", cfg.TopicBetConfirmed),
	)

	ctx := context.Background()

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			log.Warn("kafka read", zap.Error(err))
			time.Sleep(time.Second)
			continue
		}

		var placed betPlaced
		if jerr := json.Unmarshal(msg.Value, &placed); jerr != nil {
			log.Error("unmarshal bet_placed", zap.Error(jerr))
			continue
		}

		if err := processOne(ctx, log, pg, cfg, confirmedWriter, dlqWriter, &placed); err != nil {
			log.Error("process bet", zap.String("betId", placed.BetID), zap.Error(err))
			time.Sleep(500 * time.Millisecond)
		}
	}
}

func processOne(
	ctx context.Context,
	log *zap.Logger,
	pg *sql.DB,
	cfg config.Config,
	confirmedWriter *kafkago.Writer,
	dlqWriter *kafkago.Writer,
	placed *betPlaced,
) error {
	// Chamada ao supplier com retries simples.
	sresp, err := callSupplierConfirm(ctx, cfg, placed)
	if err != nil {
		const retries = 3
		for i := 0; i < retries; i++ {
			time.Sleep(time.Duration(300*(i+1)) * time.Millisecond)
			if sresp, err = callSupplierConfirm(ctx, cfg, placed); err == nil {
				break
			}
		}
		if err != nil {
			if dlqWriter != nil {
				_ = kafka.WriteJSON(ctx, dlqWriter, placed.BetID, mustJSON(placed))
			}
			return err
		}
	}

	// Atualização do status da aposta.
	newStatus := strings.ToUpper(sresp.Status)
	if newStatus != "CONFIRMED" && newStatus != "REJECTED" {
		newStatus = "REJECTED"
	}
	if err := updateBetStatus(ctx, pg, placed.BetID, newStatus); err != nil {
		return err
	}
	if err := insertBetTransaction(ctx, pg, placed.BetID, "PENDING_CONFIRMATION", newStatus, sresp.Reason); err != nil {
		log.Warn("bet_tx insert", zap.Error(err))
	}

	// Estorno de saldo em caso de rejeição.
	if newStatus == "REJECTED" {
		if err := walletRefund(ctx, cfg, placed.UserID, placed.StakeCents, "bet-reject:"+placed.BetID); err != nil {
			log.Error("wallet refund", zap.Error(err))
		}
	}

	// Publicação do evento bet_confirmed.
	evc := ev.BetConfirmed{
		BetID:       placed.BetID,
		UserID:      placed.UserID,
		Status:      newStatus,
		Reason:      sresp.Reason,
		ProviderRef: sresp.ProviderRef,
		Ts:          time.Now(),
	}
	return kafka.WriteJSON(ctx, confirmedWriter, placed.BetID, mustJSON(evc))
}

func callSupplierConfirm(ctx context.Context, cfg config.Config, p *betPlaced) (*supplierConfirmResp, error) {
	body, _ := json.Marshal(map[string]any{
		"betId":       p.BetID,
		"userId":      p.UserID,
		"eventId":     p.EventID,
		"stake_cents": p.StakeCents,
		"odd_value":   p.OddValue,
	})

	// Derivação da base HTTP a partir da URL de WS do supplier.
	base := cfg.SupplierWSURL
	base = strings.Replace(base, "ws://", "http://", 1)
	base = strings.TrimSuffix(base, "/ws")
	url := base + "/supplier/confirm"

	// Cliente HTTP com timeout.
	httpClient := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, errors.New("supplier http " + resp.Status)
	}

	var out supplierConfirmResp
	if jerr := json.NewDecoder(resp.Body).Decode(&out); jerr != nil {
		return nil, jerr
	}
	return &out, nil
}

func updateBetStatus(ctx context.Context, pg *sql.DB, betID, status string) error {
	_, err := pg.ExecContext(ctx, `UPDATE bets SET status=$1, updated_at=NOW() WHERE id=$2`, status, betID)
	return err
}

func insertBetTransaction(ctx context.Context, pg *sql.DB, betID, oldStatus, newStatus, reason string) error {
	_, err := pg.ExecContext(ctx, `
		INSERT INTO bet_transactions (bet_id, old_status, new_status, reason, created_at)
		VALUES ($1,$2,$3,$4,NOW())`, betID, oldStatus, newStatus, reason)
	return err
}

func walletRefund(ctx context.Context, cfg config.Config, userID string, amount int64, ext string) error {
	payload, _ := json.Marshal(map[string]any{
		"userId":       userID,
		"amount_cents": amount,
		"external_ref": ext,
	})

	// URL do wallet parametrizável por config; padrão para ambiente Docker.
	walletBase := cfg.WalletBaseURL
	if walletBase == "" {
		walletBase = "http://wallet-service:8082"
	}
	url := strings.TrimRight(walletBase, "/") + "/wallet/refund"

	httpClient := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return errors.New("wallet refund http " + resp.Status)
	}
	return nil
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
