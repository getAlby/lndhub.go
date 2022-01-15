package models

import "github.com/uptrace/bun"

// Account : Account Model
type Account struct {
	bun.BaseModel `bun:"account"`
	
	UserID uint
	Type   string
}
