package kafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
	topic  string
}

func NewProducer(brokers []string, topic string) *Producer {
	return &Producer{
		topic: topic,
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.Hash{},
			RequiredAcks: kafka.RequireOne,
		},
	}
}

func (p *Producer) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	return p.writer.Close()
}

type paymentCompletedEvent struct {
	Timestamp     time.Time `json:"timestamp"`
	Amount        int64     `json:"amount"`
	EventID       string    `json:"event_id"`
	Type          string    `json:"type"`
	AccountID     string    `json:"account_id"`
	TransactionID string    `json:"transaction_id"`
}

type transferCompletedEvent struct {
	Timestamp     time.Time `json:"timestamp"`
	Amount        int64     `json:"amount"`
	EventID       string    `json:"event_id"`
	Type          string    `json:"type"`
	FromAccountID string    `json:"from_account_id"`
	ToAccountID   string    `json:"to_account_id"`
	TransactionID string    `json:"transaction_id"`
}

type transferFailedEvent struct {
	Timestamp     time.Time `json:"timestamp"`
	Amount        int64     `json:"amount"`
	EventID       string    `json:"event_id"`
	Type          string    `json:"type"`
	FromAccountID string    `json:"from_account_id"`
	ToAccountID   string    `json:"to_account_id"`
	Reason        string    `json:"reason"`
}

func (p *Producer) PublishPaymentCompleted(ctx context.Context, accountID, txID string, amount int64) error {
	ev := paymentCompletedEvent{
		EventID:       newEventID(),
		Type:          "payment.completed",
		AccountID:     accountID,
		TransactionID: txID,
		Amount:        amount,
		Timestamp:     time.Now().UTC(),
	}
	return p.send(ctx, accountID, ev)
}

func (p *Producer) PublishTransferCompleted(ctx context.Context, fromAccountID, toAccountID, txID string, amount int64) error {
	ev := transferCompletedEvent{
		EventID:       newEventID(),
		Type:          "transfer.completed",
		FromAccountID: fromAccountID,
		ToAccountID:   toAccountID,
		TransactionID: txID,
		Amount:        amount,
		Timestamp:     time.Now().UTC(),
	}
	return p.send(ctx, fromAccountID, ev)
}

func (p *Producer) PublishTransferFailed(ctx context.Context, fromAccountID, toAccountID string, amount int64, reason string) error {
	ev := transferFailedEvent{
		EventID:       newEventID(),
		Type:          "transfer.failed",
		FromAccountID: fromAccountID,
		ToAccountID:   toAccountID,
		Amount:        amount,
		Reason:        reason,
		Timestamp:     time.Now().UTC(),
	}
	return p.send(ctx, fromAccountID, ev)
}

func (p *Producer) send(ctx context.Context, key string, payload any) error {
	if p == nil || p.writer == nil {
		return nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	msg := kafka.Message{
		Key:   []byte(key),
		Time:  time.Now().UTC(),
		Value: data,
	}
	return p.writer.WriteMessages(ctx, msg)
}

func newEventID() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
