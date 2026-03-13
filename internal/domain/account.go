package domain

import (
	"time"

	"github.com/google/uuid"
)

// Account — счёт пользователя. Баланс в копейках (минимальные единицы).
type Account struct {
	ID        uuid.UUID
	UserID    string
	Balance   int64
	CreatedAt time.Time
}

// Transaction — запись об операции по счёту (пополнение или часть перевода).
type Transaction struct {
	ID         uuid.UUID
	AccountID  uuid.UUID
	Type       OperationType
	Amount     int64     // всегда положительное
	RelatedID  *uuid.UUID // для transfer — id парной операции
	CreatedAt  time.Time
}

// OperationType — тип операции в истории.
type OperationType string

const (
	OpDeposit     OperationType = "deposit"
	OpTransferOut OperationType = "transfer_out"
	OpTransferIn  OperationType = "transfer_in"
)
