package repo

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
)

// Postgres implementa operações de carteira em banco
type Postgres struct{ db *sql.DB }

func NewPostgres(db *sql.DB) *Postgres { return &Postgres{db: db} }

var (
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrNotFound          = errors.New("not found")
)

// GetOrCreateWallet retorna o walletId e saldo de um usuário, criando a carteira se não existir
// Usa transação para garantir atomicidade
func (p *Postgres) GetOrCreateWallet(ctx context.Context, userID string) (walletID string, balance int64, err error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return "", 0, err
	}
	defer tx.Rollback()

	var id string
	var bal int64
	err = tx.QueryRowContext(ctx, `SELECT id, balance_cents FROM wallets WHERE user_id=$1`, userID).Scan(&id, &bal)
	if err == sql.ErrNoRows {
		id = uuid.New().String()
		if _, err = tx.ExecContext(ctx,
			`INSERT INTO wallets(id, user_id, balance_cents, version) VALUES($1,$2,0,1)`,
			id, userID); err != nil {
			return "", 0, err
		}
		bal = 0
	} else if err != nil {
		return "", 0, err
	}

	if err = tx.Commit(); err != nil {
		return "", 0, err
	}

	return id, bal, nil
}

// Deposit incrementa o saldo da carteira e registra a operação no ledger
// Garante lock pessimista na linha da carteira
func (p *Postgres) Deposit(ctx context.Context, userID string, amount int64, externalRef string) (walletID string, newBalance int64, err error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return "", 0, err
	}
	defer tx.Rollback()

	var id string
	if err = tx.QueryRowContext(ctx, `SELECT id FROM wallets WHERE user_id=$1 FOR UPDATE`, userID).Scan(&id); err != nil {
		return "", 0, err
	}

	if _, err = tx.ExecContext(ctx, `UPDATE wallets SET balance_cents = balance_cents + $1, version = version + 1 WHERE id=$2`, amount, id); err != nil {
		return "", 0, err
	}

	if _, err = tx.ExecContext(ctx, `INSERT INTO wallet_ledger(wallet_id, operation_type, amount_cents, description) VALUES($1,'CREDIT',$2,$3)`,
		id, amount, "deposit:"+externalRef); err != nil {
		return "", 0, err
	}

	if err = tx.QueryRowContext(ctx, `SELECT balance_cents FROM wallets WHERE id=$1`, id).Scan(&newBalance); err != nil {
		return "", 0, err
	}

	if err = tx.Commit(); err != nil {
		return "", 0, err
	}
	return id, newBalance, nil
}

// Reserve cria uma reserva PENDING e debita saldo (bloqueio)
// Garante idempotência por (wallet_id, external_ref)
func (p *Postgres) Reserve(ctx context.Context, userID string, amount int64, externalRef string) (reservationID string, err error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	var walletID string
	if err = tx.QueryRowContext(ctx, `SELECT id FROM wallets WHERE user_id=$1 FOR UPDATE`, userID).Scan(&walletID); err != nil {
		return "", err
	}

	var balance int64
	if err = tx.QueryRowContext(ctx, `SELECT balance_cents FROM wallets WHERE id=$1`, walletID).Scan(&balance); err != nil {
		return "", err
	}

	if balance < amount {
		return "", ErrInsufficientFunds
	}

	// Idempotência: verifica se já existe reserva para o mesmo external_ref
	var exists string
	err = tx.QueryRowContext(ctx, `SELECT id FROM wallet_reservations WHERE wallet_id=$1 AND external_ref=$2`, walletID, externalRef).Scan(&exists)

	if err == nil {
		return exists, nil // já existe
	} else if err != sql.ErrNoRows {
		return "", err
	}

	// Debita saldo (bloqueio)
	if _, err = tx.ExecContext(ctx, `UPDATE wallets SET balance_cents = balance_cents - $1, version = version + 1 WHERE id=$2`, amount, walletID); err != nil {
		return "", err
	}

	reservationID = uuid.New().String()
	if _, err = tx.ExecContext(ctx, `INSERT INTO wallet_reservations(id, wallet_id, external_ref, amount_cents, status) VALUES($1,$2,$3,$4,'PENDING')`,
		reservationID, walletID, externalRef, amount); err != nil {
		return "", err
	}

	if _, err = tx.ExecContext(ctx, `INSERT INTO wallet_ledger(wallet_id, operation_type, amount_cents, description, related_bet_id)
		VALUES($1,'RESERVE',$2,$3,$4)`,
		walletID, amount, "reserve:"+externalRef, nil); err != nil {
		return "", err
	}

	if err = tx.Commit(); err != nil {
		return "", err
	}

	return reservationID, nil
}

// Commit efetiva uma reserva, marcando como COMMITTED e registrando débito no ledger
// Idempotente: se já estiver committed, não faz nada
func (p *Postgres) Commit(ctx context.Context, userID, externalRef string) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var walletID, resID string
	var status string
	var amount int64

	if err = tx.QueryRowContext(ctx, `
		SELECT wr.id, wr.wallet_id, wr.amount_cents, wr.status
		FROM wallet_reservations wr
		JOIN wallets w ON w.id = wr.wallet_id
		WHERE w.user_id=$1 AND wr.external_ref=$2
		FOR UPDATE`, userID, externalRef).Scan(&resID, &walletID, &amount, &status); err != nil {
		return ErrNotFound
	}

	if status != "PENDING" {
		return nil
	} // idempotente

	if _, err = tx.ExecContext(ctx, `UPDATE wallet_reservations SET status='COMMITTED' WHERE id=$1`, resID); err != nil {
		return err
	}

	if _, err = tx.ExecContext(ctx, `INSERT INTO wallet_ledger(wallet_id, operation_type, amount_cents, description)
		VALUES($1,'DEBIT',$2,$3)`, walletID, amount, "commit:"+externalRef); err != nil {
		return err
	}

	return tx.Commit()
}

// Refund desfaz uma reserva PENDING, devolvendo saldo e registrando no ledger
// Idempotente: se já estiver REFUNDED, não faz nada
func (p *Postgres) Refund(ctx context.Context, userID, externalRef string) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var walletID, resID string
	var status string
	var amount int64

	if err = tx.QueryRowContext(ctx, `
		SELECT wr.id, wr.wallet_id, wr.amount_cents, wr.status
		FROM wallet_reservations wr
		JOIN wallets w ON w.id = wr.wallet_id
		WHERE w.user_id=$1 AND wr.external_ref=$2
		FOR UPDATE`, userID, externalRef).Scan(&resID, &walletID, &amount, &status); err != nil {
		return ErrNotFound
	}

	if status != "PENDING" {
		return nil
	} // já tratado

	// Devolve saldo
	if _, err = tx.ExecContext(ctx, `UPDATE wallets SET balance_cents = balance_cents + $1, version = version + 1 WHERE id=$2`, amount, walletID); err != nil {
		return err
	}

	if _, err = tx.ExecContext(ctx, `UPDATE wallet_reservations SET status='REFUNDED' WHERE id=$1`, resID); err != nil {
		return err
	}

	if _, err = tx.ExecContext(ctx, `INSERT INTO wallet_ledger(wallet_id, operation_type, amount_cents, description)
		VALUES($1,'REFUND',$2,$3)`, walletID, amount, "refund:"+externalRef); err != nil {
		return err
	}

	return tx.Commit()
}
