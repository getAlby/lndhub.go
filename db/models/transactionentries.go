package models

import (
	"time"
)

// TransactionEntry : Transaction Entries Model
type TransactionEntry struct {
	UserID          uint
	InvoiceID       uint
	CreditAccountID uint
	DebitAccountID  uint
	Amount          uint64
	CreatedAt       time.Time `bun:",nullzero,notnull,default:current_timestamp"`
}
