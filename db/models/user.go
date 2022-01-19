package models

import (
	"context"
	"database/sql"
	"time"

	"github.com/uptrace/bun"
)

// User : User Model
type User struct {
	ID        int64          `bun:",pk,autoincrement"`
	Email     sql.NullString `bun:",unique"`
	Login     string         `bun:",unique,notnull"`
	Password  string         `bun:",notnull"`
	CreatedAt time.Time      `bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt bun.NullTime
	Invoices  []*Invoice `bun:"rel:has-many,join:id=user_id"`
	Accounts  []*Account `bun:"rel:has-many,join:id=user_id"`
}

func (u *User) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.UpdateQuery:
		u.UpdatedAt = bun.NullTime{Time: time.Now()}
	}
	return nil
}

func (u *User) AccountFor(accountType string, ctx context.Context, db bun.IDB) (Account, error) {
	account := Account{}
	err := db.NewSelect().Model(&account).Where("user_id = ? AND type= ?", u.ID, accountType).Limit(1).Scan(ctx)
	return account, err
}

func (u *User) CurrentBalance(ctx context.Context, db bun.IDB) (int64, error) {
	var balance int64

	account, err := u.AccountFor("current", ctx, db)
	if err != nil {
		return balance, err
	}
	err = db.NewSelect().Table("account_ledgers").ColumnExpr("sum(account_ledgers.amount) as balance").Where("account_ledgers.account_id = ?", account.ID).Scan(context.TODO(), &balance)
	return balance, err
}

var _ bun.BeforeAppendModelHook = (*User)(nil)
