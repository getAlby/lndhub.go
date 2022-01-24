package service

import (
	"context"
	"database/sql"
	"math/rand"

	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/lib/security"
	"github.com/uptrace/bun"
)

func (svc *LndhubService) CreateUser() (user *models.User, err error) {

	user = &models.User{}

	// generate user login/password (TODO: allow the user to choose a login/password?)
	user.Login = randStringBytes(20)
	password := randStringBytes(20)
	// we only store the hashed password but return the initial plain text password in the HTTP response
	hashedPassword := security.HashPassword(password)
	user.Password = hashedPassword

	// Create user and the user's accounts
	// We use double-entry bookkeeping so we use 4 accounts: incoming, current, outgoing and fees
	// Wrapping this in a transaction in case something fails
	err = svc.DB.RunInTx(context.TODO(), &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(user).Exec(ctx); err != nil {
			return err
		}
		accountTypes := []string{"incoming", "current", "outgoing", "fees"}
		for _, accountType := range accountTypes {
			account := models.Account{UserID: user.ID, Type: accountType}
			if _, err := tx.NewInsert().Model(&account).Exec(ctx); err != nil {
				return err
			}
		}
		return nil
	})
	//return actual password in the response, not the hashed one
	user.Password = password
	return user, err
}

func (svc *LndhubService) FindUser(ctx context.Context, userId int64) (*models.User, error) {
	var user models.User

	err := svc.DB.NewSelect().Model(&user).Where("id = ?", userId).Limit(1).Scan(ctx)
	if err != nil {
		return &user, err
	}
	return &user, nil
}

func (svc *LndhubService) CurrentUserBalance(ctx context.Context, userId int64) (int64, error) {
	var balance int64

	account, err := svc.AccountFor(ctx, "current", userId)
	if err != nil {
		return balance, err
	}
	err = svc.DB.NewSelect().Table("account_ledgers").ColumnExpr("sum(account_ledgers.amount) as balance").Where("account_ledgers.account_id = ?", account.ID).Scan(context.TODO(), &balance)
	return balance, err
}

func (svc *LndhubService) AccountFor(ctx context.Context, accountType string, userId int64) (models.Account, error) {
	account := models.Account{}
	err := svc.DB.NewSelect().Model(&account).Where("user_id = ? AND type= ?", userId, accountType).Limit(1).Scan(ctx)
	return account, err
}

func (svc *LndhubService) InvoicesFor(ctx context.Context, userId int64, invoiceType string) ([]models.Invoice, error) {
	var invoices []models.Invoice

	query := svc.DB.NewSelect().Model(&invoices).Where("user_id = ?", userId)
	if invoiceType != "" {
		query.Where("type = ? AND state <> ?", invoiceType, "initialized")
	}
	query.OrderExpr("id DESC").Limit(100)
	err := query.Scan(ctx)
	if err != nil {
		return nil, err
	}
	return invoices, nil
}

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = alphaNumBytes[rand.Intn(len(alphaNumBytes))]
	}
	return string(b)
}
