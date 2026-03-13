package domain

import (
	"time"

	"github.com/google/uuid"
)

// Account — счёт пользователя. Баланс в копейках (минимальные единицы).
type Account struct {
	Balance   int64
	CreatedAt time.Time
	ID        uuid.UUID
	UserID    string
}

// Transaction — запись об операции по счёту (пополнение или часть перевода).
type Transaction struct {
	Amount    int64 // всегда положительное
	CreatedAt time.Time
	RelatedID *uuid.UUID // для transfer — id парной операции
	ID        uuid.UUID
	AccountID uuid.UUID
	Type      OperationType
}

// OperationType — тип операции в истории.
type OperationType string

const (
	OpDeposit     OperationType = "deposit"
	OpTransferOut OperationType = "transfer_out"
	OpTransferIn  OperationType = "transfer_in"
)
