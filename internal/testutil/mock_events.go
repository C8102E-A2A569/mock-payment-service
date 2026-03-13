package testutil

import (
	"context"
	"sync"
)

// MockEventProducer записывает вызовы событий для проверки в тестах (Kafka-слой).
type MockEventProducer struct {
	mu                   sync.Mutex
	PaymentCompleted     []PaymentCompletedCall
	TransferCompleted    []TransferCompletedCall
	TransferFailed       []TransferFailedCall
}

type PaymentCompletedCall struct {
	AccountID, TxID string
	Amount         int64
}

type TransferCompletedCall struct {
	FromAccountID, ToAccountID, TxID string
	Amount                           int64
}

type TransferFailedCall struct {
	FromAccountID, ToAccountID string
	Amount                     int64
	Reason                     string
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
