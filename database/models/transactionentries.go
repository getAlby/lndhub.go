package models

import "time"

// TransactionEntries : Transaction Entries Model
type TransactionEntries struct {
	UserID          uint
	InvoiceID       uint
	CreditAccountID uint
	DebitAccountID  uint
	Amount          uint64
	CreatedAt       time.Time
}
