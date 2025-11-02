-- 0004_user_id_text_no_fk.up.sql
-- Objetivo:
-- - Remover FKs que amarram user_id -> users(id)
-- - Converter colunas user_id (wallets, bets) de UUID para TEXT
-- - Recriar índices necessários

-- 1) Remover FKs para users(id), se existirem
ALTER TABLE IF EXISTS wallets DROP CONSTRAINT IF EXISTS wallets_user_id_fkey;
ALTER TABLE IF EXISTS bets    DROP CONSTRAINT IF EXISTS bets_user_id_fkey;

-- 2) Remover índices dependentes de user_id para evitar conflito na mudança de tipo
DROP INDEX IF EXISTS idx_wallets_user_id;
DROP INDEX IF EXISTS idx_bets_user_id;

-- 3) Alterar wallets.user_id para TEXT (somente se for UUID)
DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema='public'
      AND table_name='wallets'
      AND column_name='user_id'
      AND data_type='uuid'
  ) THEN
    ALTER TABLE wallets
      ALTER COLUMN user_id TYPE TEXT USING user_id::text;
  END IF;
END$$;

-- 4) Alterar bets.user_id para TEXT (somente se for UUID)
DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema='public'
      AND table_name='bets'
      AND column_name='user_id'
      AND data_type='uuid'
  ) THEN
    ALTER TABLE bets
      ALTER COLUMN user_id TYPE TEXT USING user_id::text;
  END IF;
END$$;

-- 5) Recriar índices
-- wallets: 1 carteira por usuário -> unique
CREATE UNIQUE INDEX IF NOT EXISTS idx_wallets_user_id ON wallets(user_id);

-- bets: index por user_id para consultas
CREATE INDEX IF NOT EXISTS idx_bets_user_id ON bets(user_id);
