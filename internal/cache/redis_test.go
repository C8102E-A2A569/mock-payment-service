package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
)

func TestRedisCache_BalanceAndIdempotency(t *testing.T) {
	mr := miniredis.RunT(t)
	defer mr.Close()

	c, err := NewRedisCache(mr.Addr(), "", 5*time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("NewRedisCache: %v", err)
	}
	defer c.Close()

	ctx := context.Background()
	accID := uuid.New()

	// GetBalance — пустой кэш
	bal, ok, err := c.GetBalance(ctx, accID)
	if err != nil || ok || bal != 0 {
		t.Errorf("GetBalance empty: bal=%d ok=%v err=%v", bal, ok, err)
	}

	// SetBalance + GetBalance
	if setErr := c.SetBalance(ctx, accID, 150); setErr != nil {
		t.Fatalf("SetBalance: %v", setErr)
	}
	bal, ok, err = c.GetBalance(ctx, accID)
	if err != nil || !ok || bal != 150 {
		t.Errorf("GetBalance after set: bal=%d ok=%v err=%v", bal, ok, err)
	}

	// InvalidateBalance
	if invErr := c.InvalidateBalance(ctx, accID); invErr != nil {
		t.Fatalf("InvalidateBalance: %v", invErr)
	}
	bal, ok, err = c.GetBalance(ctx, accID)
	if err != nil || ok {
		t.Errorf("GetBalance after invalidate: should be miss: bal=%d ok=%v", bal, ok)
	}

	// GetIdempotency — пустой
	raw, ok, err := c.GetIdempotency(ctx, "deposit", "key1")
	if err != nil || ok || raw != nil {
		t.Errorf("GetIdempotency empty: ok=%v err=%v", ok, err)
	}

	// SetIdempotency + GetIdempotency
	payload := []byte(`{"transaction_id":"abc","new_balance":200}`)
	if setErr := c.SetIdempotency(ctx, "deposit", "key1", payload); setErr != nil {
		t.Fatalf("SetIdempotency: %v", setErr)
	}
	raw, ok, err = c.GetIdempotency(ctx, "deposit", "key1")
	if err != nil || !ok || string(raw) != string(payload) {
		t.Errorf("GetIdempotency after set: ok=%v raw=%s err=%v", ok, raw, err)
	}

	// Разные prefix — разные ключи
	raw, ok, _ = c.GetIdempotency(ctx, "transfer", "key1")
	if ok {
		t.Error("transfer:key1 should be empty")
	}
	if setErr := c.SetIdempotency(ctx, "transfer", "key1", []byte("transfer-payload")); setErr != nil {
		t.Fatalf("SetIdempotency transfer: %v", setErr)
	}
	raw, ok, _ = c.GetIdempotency(ctx, "transfer", "key1")
	if !ok || string(raw) != "transfer-payload" {
		t.Errorf("transfer key1: ok=%v raw=%s", ok, raw)
	}
}

func TestRedisCache_IdempotencyEmptyKey(t *testing.T) {
	mr := miniredis.RunT(t)
	defer mr.Close()

	c, err := NewRedisCache(mr.Addr(), "", time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("NewRedisCache: %v", err)
	}
	defer c.Close()

	ctx := context.Background()

	// Пустой idemKey — Get возвращает false, Set не падает
	raw, ok, err := c.GetIdempotency(ctx, "deposit", "")
	if err != nil || ok || raw != nil {
		t.Errorf("GetIdempotency empty key: ok=%v", ok)
	}
	if setErr := c.SetIdempotency(ctx, "deposit", "", []byte("x")); setErr != nil {
		t.Fatalf("SetIdempotency empty key: %v", setErr)
	}
}
