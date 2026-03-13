package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"new-project/internal/domain"
	"new-project/internal/repository"
	"new-project/pkg/apperror"

	"github.com/google/uuid"
)

// PaymentService — бизнес-логика: счета, пополнение, переводы.
type PaymentService struct {
	repo   repository.AccountRepository
	events EventProducer
	cache  PaymentCache
}

// EventProducer описывает продюсер событий платежей (Kafka или другой транспорт).
type EventProducer interface {
	PublishPaymentCompleted(ctx context.Context, accountID, txID string, amount int64) error
	PublishTransferCompleted(ctx context.Context, fromAccountID, toAccountID, txID string, amount int64) error
	PublishTransferFailed(ctx context.Context, fromAccountID, toAccountID string, amount int64, reason string) error
}

// PaymentCache — кэш балансов и идемпотентности (Redis и т.п.).
type PaymentCache interface {
	GetBalance(ctx context.Context, accountID uuid.UUID) (int64, bool, error)
	SetBalance(ctx context.Context, accountID uuid.UUID, balance int64) error
	InvalidateBalance(ctx context.Context, accountID uuid.UUID) error
	GetIdempotency(ctx context.Context, prefix, idemKey string) ([]byte, bool, error)
	SetIdempotency(ctx context.Context, prefix, idemKey string, value []byte) error
}

// NewPaymentService создаёт сервис платежей (events и cache опциональны, можно nil).
func NewPaymentService(repo repository.AccountRepository, events EventProducer, cache PaymentCache) *PaymentService {
	return &PaymentService{repo: repo, events: events, cache: cache}
}

// CreateAccount создаёт счёт с нулевым балансом.
func (s *PaymentService) CreateAccount(ctx context.Context, userID string) (*domain.Account, error) {
	acc, err := s.repo.CreateAccount(ctx, userID)
	if err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, fmt.Sprintf("create account for user %s", userID), err)
	}
	return acc, nil
}

// GetBalance возвращает баланс счёта.
// При отсутствии счёта возвращает ошибку с кодом NOT_FOUND.
func (s *PaymentService) GetBalance(ctx context.Context, accountID uuid.UUID) (int64, error) {
	if s.cache != nil {
		if balance, ok, err := s.cache.GetBalance(ctx, accountID); err == nil && ok {
			return balance, nil
		}
	}
	acc, err := s.repo.GetAccount(ctx, accountID)
	if err != nil {
		return 0, apperror.Wrap(apperror.CodeInternal, fmt.Sprintf("get account %s", accountID.String()), err)
	}
	if acc == nil {
		return 0, apperror.New(apperror.CodeNotFound, "account not found")
	}
	if s.cache != nil {
		_ = s.cache.SetBalance(ctx, accountID, acc.Balance)
	}
	return acc.Balance, nil
}

// Deposit пополняет счёт. Возвращает ID транзакции и новый баланс.
// idemKey — ключ идемпотентности; при повторном запросе с тем же ключом возвращается закэшированный ответ.
func (s *PaymentService) Deposit(ctx context.Context, accountID uuid.UUID, amount int64, idemKey string) (txID uuid.UUID, newBalance int64, err error) {
	if amount <= 0 {
		return uuid.Nil, 0, apperror.New(apperror.CodeInvalidArgument, "amount must be positive")
	}
	if s.cache != nil && idemKey != "" {
		if raw, ok, _ := s.cache.GetIdempotency(ctx, "deposit", idemKey); ok {
			var cached struct {
				TransactionID string `json:"transaction_id"`
				NewBalance    int64  `json:"new_balance"`
			}
			if json.Unmarshal(raw, &cached) == nil {
				if id, parseErr := uuid.Parse(cached.TransactionID); parseErr == nil {
					return id, cached.NewBalance, nil
				}
			}
		}
	}
	txID, newBalance, err = s.repo.Deposit(ctx, accountID, amount)
	if err != nil {
		return uuid.Nil, 0, apperror.Wrap(apperror.CodeInternal, fmt.Sprintf("deposit to account %s", accountID.String()), err)
	}
	if s.cache != nil {
		_ = s.cache.InvalidateBalance(ctx, accountID)
		if idemKey != "" {
			payload, _ := json.Marshal(map[string]interface{}{"transaction_id": txID.String(), "new_balance": newBalance})
			_ = s.cache.SetIdempotency(ctx, "deposit", idemKey, payload)
		}
	}
	if s.events != nil {
		if err := s.events.PublishPaymentCompleted(ctx, accountID.String(), txID.String(), amount); err != nil {
			log.Printf("kafka publish payment.completed failed: %v", err)
		}
	}
	return txID, newBalance, nil
}

// Transfer переводит сумму между счетами.
// При недостатке средств возвращает ошибку с кодом INSUFFICIENT_FUNDS.
// idemKey — ключ идемпотентности; при повторном запросе с тем же ключом возвращается закэшированный ответ.
func (s *PaymentService) Transfer(ctx context.Context, fromID, toID uuid.UUID, amount int64, idemKey string) (txID uuid.UUID, fromBalance, toBalance int64, err error) {
	if amount <= 0 {
		return uuid.Nil, 0, 0, apperror.New(apperror.CodeInvalidArgument, "amount must be positive")
	}
	if fromID == toID {
		return uuid.Nil, 0, 0, apperror.New(apperror.CodeInvalidArgument, "from and to account are the same")
	}
	if s.cache != nil && idemKey != "" {
		if raw, ok, _ := s.cache.GetIdempotency(ctx, "transfer", idemKey); ok {
			var cached struct {
				TransactionID  string `json:"transaction_id"`
				FromNewBalance int64  `json:"from_new_balance"`
				ToNewBalance   int64  `json:"to_new_balance"`
				Success        bool   `json:"success"`
			}
			if json.Unmarshal(raw, &cached) == nil {
				if !cached.Success {
					return uuid.Nil, 0, 0, apperror.Wrap(apperror.CodeInsufficientFunds, "insufficient funds for transfer", repository.ErrInsufficientFunds)
				}
				id, _ := uuid.Parse(cached.TransactionID)
				return id, cached.FromNewBalance, cached.ToNewBalance, nil
			}
		}
	}
	txID, fromBalance, toBalance, err = s.repo.Transfer(ctx, fromID, toID, amount)
	if err != nil {
		if errors.Is(err, repository.ErrInsufficientFunds) {
			if s.cache != nil && idemKey != "" {
				payload, _ := json.Marshal(map[string]interface{}{
					"transaction_id":   uuid.Nil.String(),
					"from_new_balance": int64(0),
					"to_new_balance":   int64(0),
					"success":          false,
				})
				_ = s.cache.SetIdempotency(ctx, "transfer", idemKey, payload)
			}
			if s.events != nil {
				if pubErr := s.events.PublishTransferFailed(ctx, fromID.String(), toID.String(), amount, "insufficient funds"); pubErr != nil {
					log.Printf("kafka publish transfer.failed failed: %v", pubErr)
				}
			}
			return uuid.Nil, 0, 0, apperror.Wrap(apperror.CodeInsufficientFunds, "insufficient funds for transfer", err)
		}
		return uuid.Nil, 0, 0, apperror.Wrap(apperror.CodeInternal, "transfer failed", err)
	}
	if s.cache != nil {
		_ = s.cache.InvalidateBalance(ctx, fromID)
		_ = s.cache.InvalidateBalance(ctx, toID)
		if idemKey != "" {
			payload, _ := json.Marshal(map[string]interface{}{
				"transaction_id":   txID.String(),
				"from_new_balance": fromBalance,
				"to_new_balance":   toBalance,
				"success":          true,
			})
			_ = s.cache.SetIdempotency(ctx, "transfer", idemKey, payload)
		}
	}
	if s.events != nil {
		if err := s.events.PublishTransferCompleted(ctx, fromID.String(), toID.String(), txID.String(), amount); err != nil {
			log.Printf("kafka publish transfer.completed failed: %v", err)
		}
	}
	return txID, fromBalance, toBalance, nil
}
