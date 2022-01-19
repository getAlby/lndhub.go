package models

import (
	"time"
)

// TransactionEntry : Transaction Entries Model
type TransactionEntry struct {
	ID              int64    `bun:",pk,autoincrement"`
	UserID          int64    `bun:",notnull"`
	User            *User    `bun:"rel:belongs-to,join:user_id=id"`
	InvoiceID       int64    `bun:",notnull"`
	Invoice         *Invoice `bun:"rel:belongs-to,join:invoice_id=id"`
	ParentID        int64
	Parent          *TransactionEntry `bun:"rel:belongs-to"`
	CreditAccountID int64             `bun:",notnull"`
	CreditAccount   *Account          `bun:"rel:belongs-to,join:credit_account_id=id"`
	DebitAccountID  int64             `bun:",notnull"`
	DebitAccount    *Account          `bun:"rel:belongs-to,join:debit_account_id=id"`
	Amount          int64             `bun:",notnull"`
	CreatedAt       time.Time         `bun:",nullzero,notnull,default:current_timestamp"`
}
