package lib

import (
	"context"
	"fmt"

	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/lib/tokens"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"
)

type LndhubService struct {
	Config    *Config
	DB        *bun.DB
	LndClient *lnrpc.LightningClient
}
type Config struct {
	DatabaseUri    string `envconfig:"DATABASE_URI" required:"true"`
	SentryDSN      string `envconfig:"SENTRY_DSN"`
	LogFilePath    string `envconfig:"LOG_FILE_PATH"`
	JWTSecret      []byte `envconfig:"JWT_SECRET" required:"true"`
	JWTExpiry      int    `envconfig:"JWT_EXPIRY" default:"604800"` // in seconds
	LNDAddress     string `envconfig:"LND_ADDRESS" required:"true"`
	LNDMacaroonHex string `envconfig:"LND_MACAROON_HEX" required:"true"`
	LNDCertHex     string `envconfig:"LND_CERT_HEX"`
}

func (svc *LndhubService) CurrentBalance(ctx context.Context, userId int64) (int64, error) {
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

func (svc *LndhubService) AddInvoice(userID int64, amount uint, memo, descriptionHash string) (*models.Invoice, error) {
	invoice := &models.Invoice{
		Type:               "",
		UserID:             userID,
		TransactionEntryID: 0,
		Amount:             amount,
		Memo:               memo,
		DescriptionHash:    descriptionHash,
		PaymentRequest:     "",
		RHash:              "",
		State:              "",
	}

	// TODO: move this to a service layer and call a method
	_, err := svc.DB.NewInsert().Model(invoice).Exec(context.TODO())
	if err != nil {
		return nil, err
	}
	return invoice, nil
}
func (svc *LndhubService) GenerateToken(login, password, inRefreshToken string) (accessToken, refreshToken string, err error) {
	var user models.User

	switch {
	case login != "" || password != "":
		{
			if err := svc.DB.NewSelect().Model(&user).Where("login = ?", login).Scan(context.TODO()); err != nil {
				return "", "", fmt.Errorf("bad auth")
			}
			if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
				return "", "", fmt.Errorf("bad auth")

			}
		}
	case inRefreshToken != "":
		{
			// TODO: currently not supported
			// I'd love to remove this from the auth handler, as the refresh token
			// is usually a part of the JWT middleware: https://webdevstation.com/posts/user-authentication-with-go-using-jwt-token/
			// if the current client depends on that - we can incorporate the refresh JWT code into here
			return "", "", fmt.Errorf("bad auth")
		}
	default:
		{
			return "", "", fmt.Errorf("login and password or refresh token is required")
		}
	}

	accessToken, err = tokens.GenerateAccessToken(svc.Config.JWTSecret, svc.Config.JWTExpiry, &user)
	if err != nil {
		return "", "", err
	}

	refreshToken, err = tokens.GenerateRefreshToken(svc.Config.JWTSecret, svc.Config.JWTExpiry, &user)
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}

func (svc *LndhubService) Payinvoice(userId int64, invoice string) error {
	debitAccount, err := svc.AccountFor(context.TODO(), "current", userId)
	if err != nil {
		return err
	}
	creditAccount, err := svc.AccountFor(context.TODO(), "outgoing", userId)
	if err != nil {
		return err
	}

	entry := models.TransactionEntry{
		UserID:          userId,
		CreditAccountID: creditAccount.ID,
		DebitAccountID:  debitAccount.ID,
		Amount:          1000,
	}
	_, err = svc.DB.NewInsert().Model(&entry).Exec(context.TODO())
	return err

}
