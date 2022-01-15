package models

import (
	"time"

	"github.com/uptrace/bun"
)

// TransactionEntry : Transaction Entries Model
type TransactionEntry struct {
	bun.BaseModel `bun:"transaction_entry"`

	UserID          uint
	InvoiceID       uint
	CreditAccountID uint
	DebitAccountID  uint
	Amount          uint64
	CreatedAt       time.Time
}
