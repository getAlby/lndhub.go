package models

import "time"

// Invoice : Invoice Model
type Invoice struct {
	ID                 uint `gorm:"primary_key"`
	Type               string
	UserID             uint
	TransactionEntryID uint
	Amount             int64
	Memo               string
	DescriptionHash    string
	PaymentRequest     string
	RHash              string
	State              string
	CreatedAt          time.Time
	UpdatedAt          time.Time
	SettledAt          time.Time
}
