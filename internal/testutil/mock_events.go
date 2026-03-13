package testutil

import (
	"context"
	"sync"
)

// MockEventProducer записывает вызовы событий для проверки в тестах (Kafka-слой).
type MockEventProducer struct {
	PaymentCompleted  []PaymentCompletedCall
	TransferCompleted []TransferCompletedCall
	TransferFailed    []TransferFailedCall
	mu                sync.Mutex
}

type PaymentCompletedCall struct {
	Amount    int64
	AccountID string
	TxID      string
}

type TransferCompletedCall struct {
	Amount        int64
	FromAccountID string
	ToAccountID   string
	TxID          string
}

type TransferFailedCall struct {
	Amount        int64
	FromAccountID string
	ToAccountID   string
	Reason        string
}

func (m *MockEventProducer) PublishPaymentCompleted(ctx context.Context, accountID, txID string, amount int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PaymentCompleted = append(m.PaymentCompleted, PaymentCompletedCall{AccountID: accountID, TxID: txID, Amount: amount})
	return nil
}

func (m *MockEventProducer) PublishTransferCompleted(ctx context.Context, fromAccountID, toAccountID, txID string, amount int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TransferCompleted = append(m.TransferCompleted, TransferCompletedCall{
		FromAccountID: fromAccountID, ToAccountID: toAccountID, TxID: txID, Amount: amount,
	})
	return nil
}

func (m *MockEventProducer) PublishTransferFailed(ctx context.Context, fromAccountID, toAccountID string, amount int64, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TransferFailed = append(m.TransferFailed, TransferFailedCall{
		FromAccountID: fromAccountID, ToAccountID: toAccountID, Amount: amount, Reason: reason,
	})
	return nil
}
