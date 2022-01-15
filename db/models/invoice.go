package models

import (
	"time"

	"github.com/uptrace/bun"
)

// Invoice : Invoice Model
type Invoice struct {
	bun.BaseModel `bun:"invoice"`

	ID                 uint      `gorm:"primary_key" json:"id"`
	Type               string    `json:"type"`
	UserID             uint      `json:"user_id"`
	TransactionEntryID uint      `json:"transaction_entry_id"`
	Amount             uint      `json:"amount"`
	Memo               string    `json:"memo"`
	DescriptionHash    string    `json:"description_hash"`
	PaymentRequest     string    `json:"payment_request"`
	RHash              string    `json:"r_hash"`
	State              string    `json:"state"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	SettledAt          time.Time `json:"settled_at"`
}
