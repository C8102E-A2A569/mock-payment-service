package domain

import (
	"time"

	"github.com/google/uuid"
)

type Account struct {
	Balance   int64
	CreatedAt time.Time
	ID        uuid.UUID
	UserID    string
}

type Transaction struct {
	Amount    int64
	CreatedAt time.Time
	RelatedID *uuid.UUID
	ID        uuid.UUID
	AccountID uuid.UUID
	Type      OperationType
}

type OperationType string

const (
	OpDeposit     OperationType = "deposit"
	OpTransferOut OperationType = "transfer_out"
	OpTransferIn  OperationType = "transfer_in"
)
