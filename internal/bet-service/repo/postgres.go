package repo

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

// Postgres implementa operações de persistência de apostas em banco Postgres
type Postgres struct{ db *sql.DB }

// NewPostgres retorna uma instância do repositório de apostas
func NewPostgres(db *sql.DB) *Postgres { return &Postgres{db: db} }

// CreatePending insere uma nova aposta com status PENDING_CONFIRMATION
func (p *Postgres) CreatePending(ctx context.Context, b *Bet) (string, error) {
	id := uuid.NewString()
	_, err := p.db.ExecContext(ctx, `
		INSERT INTO bets (id,user_id,event_id,market,selection,stake_cents,odd_value,status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,'PENDING_CONFIRMATION')`,
		id, b.UserID, b.EventID, b.Market, b.Selection, b.StakeCents, b.OddValue,
	)
	if err != nil {
		return "", err
	}
	return id, nil
}

// GetStatus retorna o status atual de uma aposta pelo betID
func (p *Postgres) GetStatus(ctx context.Context, betID string) (string, error) {
	var s string
	err := p.db.QueryRowContext(ctx, `SELECT status FROM bets WHERE id=$1`, betID).Scan(&s)
	return s, err
}
