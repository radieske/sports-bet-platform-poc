package repo

import (
	"context"
	"database/sql"

	"github.com/radieske/sports-bet-platform-poc/internal/odds-service/dto"
)

type ReadRepo struct {
	DB *sql.DB
}

func (r *ReadRepo) ListEvents(ctx context.Context) ([]dto.Event, error) {
	const q = `
		SELECT event_id, MAX(home_team) AS home_team, MAX(away_team) AS away_team
		FROM odds_current
		GROUP BY event_id
		ORDER BY event_id;
	`
	rows, err := r.DB.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []dto.Event
	for rows.Next() {
		var e dto.Event
		if err := rows.Scan(&e.EventID, &e.HomeTeam, &e.AwayTeam); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *ReadRepo) ListMarkets(ctx context.Context, eventID string) ([]dto.Market, error) {
	const q = `
		SELECT DISTINCT market
		FROM odds_current
		WHERE event_id = $1
		ORDER BY market;
	`
	rows, err := r.DB.QueryContext(ctx, q, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []dto.Market
	for rows.Next() {
		var m dto.Market
		if err := rows.Scan(&m.Market); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *ReadRepo) GetOddsByEvent(ctx context.Context, eventID string) ([]dto.Odds, error) {
	const q = `
		SELECT event_id, market, home_odd, draw_odd, away_odd, version, to_char(updated_at, 'YYYY-MM-DD"T"HH24:MI:SSZ')
		FROM odds_current
		WHERE event_id = $1
		ORDER BY market;
	`
	rows, err := r.DB.QueryContext(ctx, q, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []dto.Odds
	for rows.Next() {
		var o dto.Odds
		if err := rows.Scan(&o.EventID, &o.Market, &o.HomeOdd, &o.DrawOdd, &o.AwayOdd, &o.Version, &o.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}
