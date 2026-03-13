package postgres

import (
	"context"
	"errors"

	"new-project/internal/domain"
	"new-project/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ repository.AccountRepository = (*AccountRepo)(nil)

// AccountRepo — реализация AccountRepository для PostgreSQL.
type AccountRepo struct {
	pool *pgxpool.Pool
}

// NewAccountRepo создаёт репозиторий счетов.
func NewAccountRepo(pool *pgxpool.Pool) *AccountRepo {
	return &AccountRepo{pool: pool}
}

// CreateAccount создаёт счёт с нулевым балансом.
func (r *AccountRepo) CreateAccount(ctx context.Context, userID string) (*domain.Account, error) {
	id := uuid.Must(uuid.NewV7())
	row := r.pool.QueryRow(ctx,
		`INSERT INTO accounts (id, user_id, balance) VALUES ($1, $2, 0) RETURNING id, user_id, balance, created_at`,
		id, userID,
	)
	var acc domain.Account
	err := row.Scan(&acc.ID, &acc.UserID, &acc.Balance, &acc.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &acc, nil
}

// GetAccount возвращает счёт по ID.
func (r *AccountRepo) GetAccount(ctx context.Context, accountID uuid.UUID) (*domain.Account, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, user_id, balance, created_at FROM accounts WHERE id = $1`,
		accountID,
	)
	var acc domain.Account
	err := row.Scan(&acc.ID, &acc.UserID, &acc.Balance, &acc.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &acc, nil
}

// Deposit пополняет счёт и создаёт запись в transactions. Возвращает ID транзакции и новый баланс.
func (r *AccountRepo) Deposit(ctx context.Context, accountID uuid.UUID, amount int64) (txID uuid.UUID, newBalance int64, err error) {
	txID = uuid.Must(uuid.NewV7())
	row := r.pool.QueryRow(ctx,
		`WITH upd AS (
			UPDATE accounts SET balance = balance + $2 WHERE id = $1 RETURNING balance
		),
		ins AS (
			INSERT INTO transactions (id, account_id, type, amount) VALUES ($3, $1, 'deposit', $2)
		)
		SELECT balance FROM upd`,
		accountID, amount, txID,
	)
	err = row.Scan(&newBalance)
	return txID, newBalance, err
}

// Transfer списывает с одного счёта и зачисляет на другой в одной транзакции.
// При недостатке средств возвращает ErrInsufficientFunds.
func (r *AccountRepo) Transfer(ctx context.Context, fromID, toID uuid.UUID, amount int64) (txID uuid.UUID, fromBalance, toBalance int64, err error) {
	txID = uuid.Must(uuid.NewV7())
	relatedID := uuid.Must(uuid.NewV7())

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, 0, 0, err
	}
	defer tx.Rollback(ctx)

	// Блокируем оба счёта в предсказуемом порядке (по id), чтобы избежать deadlock.
	_, err = tx.Exec(ctx,
		`SELECT id FROM accounts WHERE id = $1 OR id = $2 ORDER BY id FOR UPDATE`,
		fromID, toID,
	)
	if err != nil {
		return uuid.Nil, 0, 0, err
	}
	// Проверяем баланс до списания: иначе CHECK (balance >= 0) в БД отклонит UPDATE и вернёт ошибку вместо ErrInsufficientFunds.
	row := tx.QueryRow(ctx, `SELECT balance FROM accounts WHERE id = $1`, fromID)
	var fromBalanceBefore int64
	if err = row.Scan(&fromBalanceBefore); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, 0, 0, errors.New("from account not found")
		}
		return uuid.Nil, 0, 0, err
	}
	if fromBalanceBefore < amount {
		return uuid.Nil, 0, 0, repository.ErrInsufficientFunds
	}

	row = tx.QueryRow(ctx, `UPDATE accounts SET balance = balance - $2 WHERE id = $1 RETURNING balance`, fromID, amount)
	if err = row.Scan(&fromBalance); err != nil {
		return uuid.Nil, 0, 0, err
	}

	_, err = tx.Exec(ctx, `UPDATE accounts SET balance = balance + $2 WHERE id = $1`, toID, amount)
	if err != nil {
		return uuid.Nil, 0, 0, err
	}
	row = tx.QueryRow(ctx, `SELECT balance FROM accounts WHERE id = $1`, toID)
	if err = row.Scan(&toBalance); err != nil {
		return uuid.Nil, 0, 0, err
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO transactions (id, account_id, type, amount, related_id) VALUES ($1, $2, 'transfer_out', $3, $4)`,
		txID, fromID, amount, relatedID,
	)
	if err != nil {
		return uuid.Nil, 0, 0, err
	}
	_, err = tx.Exec(ctx,
		`INSERT INTO transactions (id, account_id, type, amount, related_id) VALUES ($1, $2, 'transfer_in', $3, $4)`,
		uuid.Must(uuid.NewV7()), toID, amount, relatedID,
	)
	if err != nil {
		return uuid.Nil, 0, 0, err
	}

	if err = tx.Commit(ctx); err != nil {
		return uuid.Nil, 0, 0, err
	}
	return txID, fromBalance, toBalance, nil
}
