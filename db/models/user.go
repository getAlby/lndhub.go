package models

import (
	"context"
	"database/sql"
	"time"

	"github.com/uptrace/bun"
)

// User : User Model
type User struct {
	ID          int64          `bun:",pk,autoincrement"`
	Email       sql.NullString `bun:",unique"`
	Login       string         `bun:",unique,notnull"`
	Password    string         `bun:",notnull"`
	CreatedAt   time.Time      `bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt   bun.NullTime
	Invoices    []*Invoice `bun:"rel:has-many,join:id=user_id"`
	Accounts    []*Account `bun:"rel:has-many,join:id=user_id"`
	Deactivated bool
}

func (u *User) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.UpdateQuery:
		u.UpdatedAt = bun.NullTime{Time: time.Now()}
	}
	return nil
}

var _ bun.BeforeAppendModelHook = (*User)(nil)
