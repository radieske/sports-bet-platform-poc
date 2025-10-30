-- ============================================
-- V001__init_schema.sql
-- Schema inicial da plataforma de apostas
-- ============================================

-- Habilita extensões úteis
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================
-- TABELA: users
-- ============================================
CREATE TABLE IF NOT EXISTS users (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email           TEXT NOT NULL UNIQUE,
    full_name       TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================
-- TABELA: wallets
-- Cada usuário possui 1 carteira vinculada
-- ============================================
CREATE TABLE IF NOT EXISTS wallets (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    balance_cents   BIGINT NOT NULL DEFAULT 0,
    version         INT NOT NULL DEFAULT 1,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_balance_positive CHECK (balance_cents >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_wallets_user_id ON wallets(user_id);

-- ============================================
-- TABELA: bets
-- Armazena as apostas realizadas
-- ============================================
CREATE TABLE IF NOT EXISTS bets (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_id        TEXT NOT NULL,
    market          TEXT NOT NULL,
    selection       TEXT NOT NULL, -- "home" | "draw" | "away"
    stake_cents     BIGINT NOT NULL,
    odd_value       NUMERIC(8,3) NOT NULL,
    potential_win   BIGINT GENERATED ALWAYS AS ((stake_cents * odd_value)::BIGINT) STORED,
    status          TEXT NOT NULL DEFAULT 'PENDING_CONFIRMATION',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_stake_positive CHECK (stake_cents > 0)
);

CREATE INDEX IF NOT EXISTS idx_bets_user_id ON bets(user_id);
CREATE INDEX IF NOT EXISTS idx_bets_event_id ON bets(event_id);

-- ============================================
-- TABELA: bet_transactions (histórico)
-- Cada registro reflete uma mudança de estado ou valor na aposta
-- ============================================
CREATE TABLE IF NOT EXISTS bet_transactions (
    id              BIGSERIAL PRIMARY KEY,
    bet_id          UUID NOT NULL REFERENCES bets(id) ON DELETE CASCADE,
    old_status      TEXT,
    new_status      TEXT,
    reason          TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_bet_transactions_bet_id ON bet_transactions(bet_id);

-- ============================================
-- TABELA: wallet_ledger (livro contábil)
-- Cada operação financeira é registrada aqui, garantindo rastreabilidade
-- ============================================
CREATE TABLE IF NOT EXISTS wallet_ledger (
    id              BIGSERIAL PRIMARY KEY,
    wallet_id       UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
    operation_type  TEXT NOT NULL, -- "DEBIT" | "CREDIT" | "RESERVE" | "REFUND"
    amount_cents    BIGINT NOT NULL,
    description     TEXT,
    related_bet_id  UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wallet_ledger_wallet_id ON wallet_ledger(wallet_id);

-- ============================================
-- FUNÇÕES DE AUDITORIA (opcional)
-- Atualiza automaticamente o updated_at
-- ============================================
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_users_updated_at
BEFORE UPDATE ON users
FOR EACH ROW EXECUTE PROCEDURE set_updated_at();

CREATE TRIGGER trg_wallets_updated_at
BEFORE UPDATE ON wallets
FOR EACH ROW EXECUTE PROCEDURE set_updated_at();

CREATE TRIGGER trg_bets_updated_at
BEFORE UPDATE ON bets
FOR EACH ROW EXECUTE PROCEDURE set_updated_at();
