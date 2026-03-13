package repository

import (
	"context"
	"errors"

	"new-project/internal/domain"

	"github.com/google/uuid"
)

// ErrInsufficientFunds — недостаточно средств для перевода.
var ErrInsufficientFunds = errors.New("insufficient funds")

// AccountRepository — операции с счетами и транзакциями в БД.
type AccountRepository interface {
	CreateAccount(ctx context.Context, userID string) (*domain.Account, error)
	GetAccount(ctx context.Context, accountID uuid.UUID) (*domain.Account, error)
	Deposit(ctx context.Context, accountID uuid.UUID, amount int64) (txID uuid.UUID, newBalance int64, err error)
	Transfer(ctx context.Context, fromID, toID uuid.UUID, amount int64) (txID uuid.UUID, fromBalance, toBalance int64, err error)
}
