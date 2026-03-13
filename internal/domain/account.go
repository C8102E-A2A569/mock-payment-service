package domain

import (
	"time"

	"github.com/google/uuid"
)

// Account — счёт пользователя. Баланс в копейках (минимальные единицы).
type Account struct {
	CreatedAt time.Time
	ID        uuid.UUID
	UserID    string
	Balance   int64
}

// Transaction — запись об операции по счёту (пополнение или часть перевода).
type Transaction struct {
	CreatedAt time.Time
	Amount    int64 // всегда положительное
	ID        uuid.UUID
	AccountID uuid.UUID
	Type      OperationType
	RelatedID *uuid.UUID // для transfer — id парной операции
}

// OperationType — тип операции в истории.
type OperationType string

const (
	OpDeposit     OperationType = "deposit"
	OpTransferOut OperationType = "transfer_out"
	OpTransferIn  OperationType = "transfer_in"
)
