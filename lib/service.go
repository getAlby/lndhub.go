package lib

import (
	"context"

	"github.com/getAlby/lndhub.go/db/models"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/uptrace/bun"
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
