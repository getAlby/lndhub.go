package models

import (
	"context"
	"time"

	"github.com/uptrace/bun"
)
type User struct {
	ID          int64          `bun:",pk,autoincrement"`
	Pubkey      string         `bun:",unique,notnull"`
	Accounts    []*Account `bun:"rel:has-many,join:id=user_id"`
	Invoices    []*Invoice `bun:"rel:has-many,join:id=user_id"`
	Deactivated bool
	Deleted     bool
	CreatedAt   time.Time      `bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt   bun.NullTime
}

func (u *User) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.UpdateQuery:
		u.UpdatedAt = bun.NullTime{Time: time.Now()}
	}
	return nil
}

var _ bun.BeforeAppendModelHook = (*User)(nil)
