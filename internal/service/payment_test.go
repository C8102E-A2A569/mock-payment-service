package service

import (
	"context"
	"errors"
	"testing"

	"new-project/internal/repository"
	"new-project/internal/testutil"

	"github.com/google/uuid"
)

func TestPaymentService_CreateAndGetBalance(t *testing.T) {
	ctx := context.Background()
	repo := testutil.NewFakeRepo()
	svc := NewPaymentService(repo, nil, nil)

	acc, err := svc.CreateAccount(ctx, "user-1")
	if err != nil {
		t.Fatalf("CreateAccount error: %v", err)
	}
	if acc.UserID != "user-1" {
		t.Fatalf("expected user_id=user-1, got %s", acc.UserID)
	}

	balance, err := svc.GetBalance(ctx, acc.ID)
	if err != nil {
		t.Fatalf("GetBalance error: %v", err)
	}
	if balance != 0 {
		t.Fatalf("expected balance=0, got %d", balance)
	}
}

func TestPaymentService_Deposit(t *testing.T) {
	ctx := context.Background()
	repo := testutil.NewFakeRepo()
	svc := NewPaymentService(repo, nil, nil)

	acc, err := svc.CreateAccount(ctx, "user-1")
	if err != nil {
		t.Fatalf("CreateAccount error: %v", err)
	}

	txID, balance, err := svc.Deposit(ctx, acc.ID, 100, "")
	if err != nil {
		t.Fatalf("Deposit error: %v", err)
	}
	if txID == uuid.Nil {
		t.Fatalf("expected non-nil transaction id")
	}
	if balance != 100 {
		t.Fatalf("expected balance=100, got %d", balance)
	}

	// Нельзя вносить неположительную сумму.
	if _, _, err := svc.Deposit(ctx, acc.ID, 0, ""); err == nil {
		t.Fatalf("expected error on zero amount")
	}
}

func TestPaymentService_Transfer_SuccessAndInsufficientFunds(t *testing.T) {
	ctx := context.Background()
	repo := testutil.NewFakeRepo()
	svc := NewPaymentService(repo, nil, nil)

	from, err := svc.CreateAccount(ctx, "from")
	if err != nil {
		t.Fatalf("CreateAccount(from) error: %v", err)
	}
	to, err := svc.CreateAccount(ctx, "to")
	if err != nil {
		t.Fatalf("CreateAccount(to) error: %v", err)
	}

	// Пополняем счёт отправителя.
	if _, _, depErr := svc.Deposit(ctx, from.ID, 200, ""); depErr != nil {
		t.Fatalf("Deposit error: %v", depErr)
	}

	// Успешный перевод.
	_, fromBal, toBal, err := svc.Transfer(ctx, from.ID, to.ID, 150, "")
	if err != nil {
		t.Fatalf("Transfer error: %v", err)
	}
	if fromBal != 50 || toBal != 150 {
		t.Fatalf("unexpected balances after transfer: from=%d to=%d", fromBal, toBal)
	}

	// Недостаточно средств.
	_, _, _, err = svc.Transfer(ctx, from.ID, to.ID, 1000, "")
	if err == nil {
		t.Fatalf("expected error on insufficient funds")
	}
	if !errors.Is(err, repository.ErrInsufficientFunds) {
		t.Fatalf("expected ErrInsufficientFunds, got %v", err)
	}
}

// ——— Kafka: проверка публикации событий при Deposit и Transfer ———

func TestPaymentService_Deposit_PublishesPaymentCompleted(t *testing.T) {
	ctx := context.Background()
	repo := testutil.NewFakeRepo()
	mockEvents := &testutil.MockEventProducer{}
	svc := NewPaymentService(repo, mockEvents, nil)

	acc, _ := svc.CreateAccount(ctx, "u1")
	txID, newBal, err := svc.Deposit(ctx, acc.ID, 100, "")
	if err != nil {
		t.Fatalf("Deposit: %v", err)
	}
	if len(mockEvents.PaymentCompleted) != 1 {
		t.Fatalf("expected 1 PaymentCompleted event, got %d", len(mockEvents.PaymentCompleted))
	}
	ev := mockEvents.PaymentCompleted[0]
	if ev.AccountID != acc.ID.String() || ev.TxID != txID.String() || ev.Amount != 100 {
		t.Errorf("event: accountID=%s txID=%s amount=%d", ev.AccountID, ev.TxID, ev.Amount)
	}
	if newBal != 100 {
		t.Errorf("newBalance=%d", newBal)
	}
}

func TestPaymentService_Transfer_PublishesCompletedOrFailed(t *testing.T) {
	ctx := context.Background()
	repo := testutil.NewFakeRepo()
	mockEvents := &testutil.MockEventProducer{}
	svc := NewPaymentService(repo, mockEvents, nil)

	from, _ := svc.CreateAccount(ctx, "from")
	to, _ := svc.CreateAccount(ctx, "to")
	_, _, _ = svc.Deposit(ctx, from.ID, 50, "")

	// Успешный перевод — ожидаем transfer.completed
	_, _, _, err := svc.Transfer(ctx, from.ID, to.ID, 30, "")
	if err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	if len(mockEvents.TransferCompleted) != 1 {
		t.Fatalf("expected 1 TransferCompleted, got %d", len(mockEvents.TransferCompleted))
	}
	if len(mockEvents.TransferFailed) != 0 {
		t.Fatalf("expected 0 TransferFailed, got %d", len(mockEvents.TransferFailed))
	}

	// Недостаточно средств — ожидаем transfer.failed
	_, _, _, err = svc.Transfer(ctx, from.ID, to.ID, 100, "")
	if err == nil {
		t.Fatal("expected error")
	}
	if len(mockEvents.TransferFailed) != 1 {
		t.Fatalf("expected 1 TransferFailed, got %d", len(mockEvents.TransferFailed))
	}
	fail := mockEvents.TransferFailed[0]
	if fail.FromAccountID != from.ID.String() || fail.ToAccountID != to.ID.String() || fail.Amount != 100 || fail.Reason != "insufficient funds" {
		t.Errorf("TransferFailed: %+v", fail)
	}
}

// ——— Redis (кэш/идемпотентность): проверка через FakeCache ———

func TestPaymentService_GetBalance_CacheHit(t *testing.T) {
	ctx := context.Background()
	repo := testutil.NewFakeRepo()
	cache := testutil.NewFakeCache()
	svc := NewPaymentService(repo, nil, cache)

	acc, _ := svc.CreateAccount(ctx, "u1")
	_, _, _ = svc.Deposit(ctx, acc.ID, 200, "")

	// Первый вызов — из репозитория, кэш заполняется.
	bal, err := svc.GetBalance(ctx, acc.ID)
	if err != nil || bal != 200 {
		t.Fatalf("first GetBalance: %v, %d", err, bal)
	}
	// Второй вызов — из кэша (проверяем, что кэш используется: подменяем баланс в репо и снова запрашиваем — без инвалидации кэш вернёт 200).
	repo.Accounts[acc.ID].Balance = 999
	bal2, err := svc.GetBalance(ctx, acc.ID)
	if err != nil || bal2 != 200 {
		t.Errorf("second GetBalance (expect cache=200): %v, %d", err, bal2)
	}
}

func TestPaymentService_Deposit_Idempotency(t *testing.T) {
	ctx := context.Background()
	repo := testutil.NewFakeRepo()
	cache := testutil.NewFakeCache()
	svc := NewPaymentService(repo, nil, cache)

	acc, _ := svc.CreateAccount(ctx, "u1")
	idemKey := "deposit-idem-1"

	tx1, bal1, err := svc.Deposit(ctx, acc.ID, 100, idemKey)
	if err != nil {
		t.Fatalf("first Deposit: %v", err)
	}
	tx2, bal2, err := svc.Deposit(ctx, acc.ID, 100, idemKey)
	if err != nil {
		t.Fatalf("second Deposit (idempotent): %v", err)
	}
	if tx1 != tx2 || bal1 != bal2 {
		t.Errorf("idempotent response must match: tx %v vs %v, bal %d vs %d", tx1, tx2, bal1, bal2)
	}
	if bal1 != 100 {
		t.Errorf("balance=%d", bal1)
	}
	// В репозитории должно быть одно пополнение (100), а не два.
	if repo.Accounts[acc.ID].Balance != 100 {
		t.Errorf("repo balance should be 100 (one deposit), got %d", repo.Accounts[acc.ID].Balance)
	}
}

func TestPaymentService_Transfer_Idempotency(t *testing.T) {
	ctx := context.Background()
	repo := testutil.NewFakeRepo()
	cache := testutil.NewFakeCache()
	svc := NewPaymentService(repo, nil, cache)

	from, _ := svc.CreateAccount(ctx, "from")
	to, _ := svc.CreateAccount(ctx, "to")
	_, _, _ = svc.Deposit(ctx, from.ID, 80, "")
	idemKey := "transfer-idem-1"

	tx1, from1, to1, err := svc.Transfer(ctx, from.ID, to.ID, 40, idemKey)
	if err != nil {
		t.Fatalf("first Transfer: %v", err)
	}
	tx2, from2, to2, err := svc.Transfer(ctx, from.ID, to.ID, 40, idemKey)
	if err != nil {
		t.Fatalf("second Transfer (idempotent): %v", err)
	}
	if tx1 != tx2 || from1 != from2 || to1 != to2 {
		t.Errorf("idempotent response must match")
	}
	if from1 != 40 || to1 != 40 {
		t.Errorf("from=%d to=%d", from1, to1)
	}
	if repo.Accounts[from.ID].Balance != 40 || repo.Accounts[to.ID].Balance != 40 {
		t.Errorf("repo: from=%d to=%d", repo.Accounts[from.ID].Balance, repo.Accounts[to.ID].Balance)
	}
}

func TestPaymentService_Transfer_FailedIdempotency(t *testing.T) {
	ctx := context.Background()
	repo := testutil.NewFakeRepo()
	cache := testutil.NewFakeCache()
	svc := NewPaymentService(repo, nil, cache)

	from, _ := svc.CreateAccount(ctx, "from")
	to, _ := svc.CreateAccount(ctx, "to")
	_, _, _ = svc.Deposit(ctx, from.ID, 10, "")
	idemKey := "transfer-fail-idem"

	_, _, _, err1 := svc.Transfer(ctx, from.ID, to.ID, 50, idemKey)
	if err1 == nil {
		t.Fatal("expected insufficient funds")
	}
	_, _, _, err2 := svc.Transfer(ctx, from.ID, to.ID, 50, idemKey)
	if err2 == nil {
		t.Fatal("expected error again (idempotent failure)")
	}
	// Оба вызова должны вернуть ошибку недостатка средств; второй — из кэша идемпотентности.
	if !errors.Is(err1, repository.ErrInsufficientFunds) || !errors.Is(err2, repository.ErrInsufficientFunds) {
		t.Errorf("both should be ErrInsufficientFunds: %v, %v", err1, err2)
	}
}
