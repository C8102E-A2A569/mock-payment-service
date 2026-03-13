package kafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/segmentio/kafka-go"
)

// Producer публикует события платежей в Kafka в JSON-формате.
type Producer struct {
	writer *kafka.Writer
	topic  string
}

// NewProducer создаёт продюсер для заданного топика и брокеров.
func NewProducer(brokers []string, topic string) *Producer {
	return &Producer{
		topic: topic,
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			RequiredAcks: kafka.RequireOne,
		},
	}
}

// Close закрывает writer.
func (p *Producer) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	return p.writer.Close()
}

type paymentCompletedEvent struct {
	EventID      string    `json:"event_id"`
	Type         string    `json:"type"`
	AccountID    string    `json:"account_id"`
	TransactionID string   `json:"transaction_id"`
	Amount       int64     `json:"amount"`
	Timestamp    time.Time `json:"timestamp"`
}

type transferCompletedEvent struct {
	EventID       string    `json:"event_id"`
	Type          string    `json:"type"`
	FromAccountID string    `json:"from_account_id"`
	ToAccountID   string    `json:"to_account_id"`
	TransactionID string    `json:"transaction_id"`
	Amount        int64     `json:"amount"`
	Timestamp     time.Time `json:"timestamp"`
}

type transferFailedEvent struct {
	EventID       string    `json:"event_id"`
	Type          string    `json:"type"`
	FromAccountID string    `json:"from_account_id"`
	ToAccountID   string    `json:"to_account_id"`
	Amount        int64     `json:"amount"`
	Reason        string    `json:"reason"`
	Timestamp     time.Time `json:"timestamp"`
}

// PublishPaymentCompleted публикует событие успешного пополнения.
func (p *Producer) PublishPaymentCompleted(ctx context.Context, accountID, txID string, amount int64) error {
	ev := paymentCompletedEvent{
		EventID:       newEventID(),
		Type:          "payment.completed",
		AccountID:     accountID,
		TransactionID: txID,
		Amount:        amount,
		Timestamp:     time.Now().UTC(),
	}
	return p.send(ctx, ev)
}

// PublishTransferCompleted публикует событие успешного перевода.
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
	return p.send(ctx, ev)
}

// PublishTransferFailed публикует событие неуспешного перевода (например, недостаточно средств).
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
	return p.send(ctx, ev)
}

func (p *Producer) send(ctx context.Context, payload any) error {
	if p == nil || p.writer == nil {
		return nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	msg := kafka.Message{
		Time:  time.Now().UTC(),
		Value: data,
	}
	return p.writer.WriteMessages(ctx, msg)
}

// newEventID генерирует простой уникальный идентификатор события.
func newEventID() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

