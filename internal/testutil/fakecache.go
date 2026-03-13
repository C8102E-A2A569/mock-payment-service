package testutil

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

// FakeCache — in-memory реализация PaymentCache для тестов (Redis-логика: кэш баланса и идемпотентность).
type FakeCache struct {
	balance     map[uuid.UUID]int64
	idempotency map[string][]byte
	mu          sync.Mutex
}

// NewFakeCache создаёт кэш для тестов.
func NewFakeCache() *FakeCache {
	return &FakeCache{
		balance:     make(map[uuid.UUID]int64),
		idempotency: make(map[string][]byte),
	}
}

func (f *FakeCache) GetBalance(ctx context.Context, accountID uuid.UUID) (int64, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.balance[accountID]
	return v, ok, nil
}

func (f *FakeCache) SetBalance(ctx context.Context, accountID uuid.UUID, balance int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.balance[accountID] = balance
	return nil
}

func (f *FakeCache) InvalidateBalance(ctx context.Context, accountID uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.balance, accountID)
	return nil
}

func idemCacheKey(prefix, key string) string { return prefix + ":" + key }

func (f *FakeCache) GetIdempotency(ctx context.Context, prefix, idemKey string) ([]byte, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.idempotency[idemCacheKey(prefix, idemKey)]
	if !ok {
		return nil, false, nil
	}
	return v, true, nil
}

func (f *FakeCache) SetIdempotency(ctx context.Context, prefix, idemKey string, value []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.idempotency[idemCacheKey(prefix, idemKey)] = value
	return nil
}
