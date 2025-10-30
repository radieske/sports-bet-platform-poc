package repository

import (
	"context"
	"database/sql"

	"github.com/radieske/sports-bet-platform-poc/pkg/contracts/events"
)

// PostgresRepo implementa operações de persistência de odds em um banco Postgres
// DB: conexão com o banco de dados
type PostgresRepo struct {
	DB *sql.DB
}

// NewPostgresRepo retorna uma instância de repositório Postgres
func NewPostgresRepo(db *sql.DB) *PostgresRepo {
	return &PostgresRepo{DB: db}
}

// UpsertCurrent insere ou atualiza a odd corrente de um evento na tabela odds_current
// Utiliza ON CONFLICT para garantir atomicidade e evitar duplicidade por event_id
func (r *PostgresRepo) UpsertCurrent(ctx context.Context, e events.OddsUpdate) error {
	const q = `
		INSERT INTO odds_current
		  (event_id, home_team, away_team, market, home_odd, draw_odd, away_odd, version, updated_at)
		VALUES
		  ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (event_id) DO UPDATE SET
		  home_team = EXCLUDED.home_team,
		  away_team = EXCLUDED.away_team,
		  market    = EXCLUDED.market,
		  home_odd  = EXCLUDED.home_odd,
		  draw_odd  = EXCLUDED.draw_odd,
		  away_odd  = EXCLUDED.away_odd,
		  version   = EXCLUDED.version,
		  updated_at= EXCLUDED.updated_at
	`
	_, err := r.DB.ExecContext(ctx, q,
		e.EventID, e.HomeTeam, e.AwayTeam, e.Market,
		e.Odds.Home, e.Odds.Draw, e.Odds.Away,
		e.Version, e.UpdatedAt,
	)
	return err
}

// InsertHistory insere uma nova odd no histórico de odds (odds_history)
func (r *PostgresRepo) InsertHistory(ctx context.Context, e events.OddsUpdate) error {
	const q = `
		INSERT INTO odds_history
		  (event_id, home_odd, draw_odd, away_odd, version, updated_at)
		VALUES
		  ($1,$2,$3,$4,$5,$6)
	`
	_, err := r.DB.ExecContext(ctx, q,
		e.EventID, e.Odds.Home, e.Odds.Draw, e.Odds.Away, e.Version, e.UpdatedAt,
	)
	return err
}
