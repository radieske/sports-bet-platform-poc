-- Snapshot atual (read-optimized)
CREATE TABLE IF NOT EXISTS odds_current (
  event_id    TEXT PRIMARY KEY,
  home_team   TEXT NOT NULL,
  away_team   TEXT NOT NULL,
  market      TEXT NOT NULL,
  home_odd    NUMERIC(8,3) NOT NULL,
  draw_odd    NUMERIC(8,3) NOT NULL,
  away_odd    NUMERIC(8,3) NOT NULL,
  version     INT NOT NULL,
  updated_at  TIMESTAMPTZ NOT NULL
);

-- Histórico (append-only)
CREATE TABLE IF NOT EXISTS odds_history (
  id         BIGSERIAL PRIMARY KEY,
  event_id   TEXT NOT NULL,
  home_odd   NUMERIC(8,3) NOT NULL,
  draw_odd   NUMERIC(8,3) NOT NULL,
  away_odd   NUMERIC(8,3) NOT NULL,
  version    INT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_odds_history_event_id ON odds_history(event_id);
