-- V003__wallet_reservations.sql

CREATE TYPE reservation_status AS ENUM ('PENDING', 'COMMITTED', 'REFUNDED', 'EXPIRED');

CREATE TABLE IF NOT EXISTS wallet_reservations (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  wallet_id       UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
  external_ref    TEXT NOT NULL, -- ex: betId ou requestId do serviço de apostas
  amount_cents    BIGINT NOT NULL CHECK (amount_cents > 0),
  status          reservation_status NOT NULL DEFAULT 'PENDING',
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (wallet_id, external_ref)
);

-- ajuda a localizar reservas recentes
CREATE INDEX IF NOT EXISTS idx_wallet_reservations_wallet_id ON wallet_reservations(wallet_id);

-- gatilho de updated_at
CREATE OR REPLACE FUNCTION set_updated_at_wallet_reservations()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_wallet_reservations_updated_at
BEFORE UPDATE ON wallet_reservations
FOR EACH ROW EXECUTE PROCEDURE set_updated_at_wallet_reservations();
