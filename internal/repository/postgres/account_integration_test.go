//go:build integration

package postgres

import (
	"context"
	"testing"

	"new-project/internal/repository"

	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestAccountRepo_Integration(t *testing.T) {
	ctx := context.Background()

	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("payments_test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("terminate container: %v", err)
		}
	}()

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	if err := RunMigrations(ctx, dsn); err != nil {
		t.Fatalf("migrations: %v", err)
	}

	pool, err := NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	repo := NewAccountRepo(pool)

	// CreateAccount + GetAccount
	acc, err := repo.CreateAccount(ctx, "user-integration")
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	if acc.Balance != 0 || acc.UserID != "user-integration" {
		t.Errorf("unexpected account: %+v", acc)
	}

	got, err := repo.GetAccount(ctx, acc.ID)
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if got == nil || got.ID != acc.ID || got.Balance != 0 {
		t.Errorf("GetAccount: %+v", got)
	}

	// Deposit
	txID, newBal, err := repo.Deposit(ctx, acc.ID, 1000)
	if err != nil {
		t.Fatalf("Deposit: %v", err)
	}
	if txID == uuid.Nil || newBal != 1000 {
		t.Errorf("Deposit: txID=%v newBal=%d", txID, newBal)
	}

	// Второй счёт и перевод
	acc2, _ := repo.CreateAccount(ctx, "user2")
	txID2, fromBal, toBal, err := repo.Transfer(ctx, acc.ID, acc2.ID, 300)
	if err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	if txID2 == uuid.Nil || fromBal != 700 || toBal != 300 {
		t.Errorf("Transfer: fromBal=%d toBal=%d", fromBal, toBal)
	}

	// Недостаточно средств
	_, _, _, err = repo.Transfer(ctx, acc.ID, acc2.ID, 10000)
	if err == nil {
		t.Fatal("expected error on insufficient funds")
	}
	if err != repository.ErrInsufficientFunds {
		t.Errorf("expected ErrInsufficientFunds, got %v", err)
	}
}
