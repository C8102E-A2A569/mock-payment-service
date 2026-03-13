package testutil

import (
	"context"
	"errors"

	"new-project/internal/domain"
	"new-project/internal/repository"

	"github.com/google/uuid"
)

// FakeRepo — in-memory реализация AccountRepository для тестов (unit и handler).
type FakeRepo struct {
	Accounts map[uuid.UUID]*domain.Account
}

// NewFakeRepo создаёт репозиторий для тестов.
func NewFakeRepo() *FakeRepo {
	return &FakeRepo{Accounts: make(map[uuid.UUID]*domain.Account)}
}

func (r *FakeRepo) CreateAccount(ctx context.Context, userID string) (*domain.Account, error) {
	id := uuid.New()
	acc := &domain.Account{ID: id, UserID: userID, Balance: 0}
	r.Accounts[id] = acc
	return acc, nil
}

func (r *FakeRepo) GetAccount(ctx context.Context, accountID uuid.UUID) (*domain.Account, error) {
	acc, ok := r.Accounts[accountID]
	if !ok {
		return nil, nil
	}
	return acc, nil
}

func (r *FakeRepo) Deposit(ctx context.Context, accountID uuid.UUID, amount int64) (txID uuid.UUID, newBalance int64, err error) {
	acc, ok := r.Accounts[accountID]
	if !ok {
		return uuid.Nil, 0, errors.New("account not found")
	}
	acc.Balance += amount
	return uuid.New(), acc.Balance, nil
}

func (r *FakeRepo) Transfer(ctx context.Context, fromID, toID uuid.UUID, amount int64) (txID uuid.UUID, fromBalance, toBalance int64, err error) {
	from, ok := r.Accounts[fromID]
	if !ok {
		return uuid.Nil, 0, 0, errors.New("from account not found")
	}
	to, ok := r.Accounts[toID]
	if !ok {
		return uuid.Nil, 0, 0, errors.New("to account not found")
	}
	if from.Balance < amount {
		return uuid.Nil, from.Balance, to.Balance, repository.ErrInsufficientFunds
	}
	from.Balance -= amount
	to.Balance += amount
	return uuid.New(), from.Balance, to.Balance, nil
}
