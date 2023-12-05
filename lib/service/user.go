package service

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	"github.com/getAlby/lndhub.go/common"
	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/security"
	"github.com/getAlby/lndhub.go/lnd"
	"github.com/getsentry/sentry-go"
	"github.com/labstack/gommon/log"
	"github.com/uptrace/bun"
	passwordvalidator "github.com/wagslane/go-password-validator"
)

func (svc *LndhubService) CreateUser(ctx context.Context, login string, password string) (user *models.User, err error) {

	user = &models.User{}

	// generate user login/password if not provided
	user.Login = login
	if login == "" {
		randLoginBytes, err := randBytesFromStr(20, alphaNumBytes)
		if err != nil {
			return nil, err
		}
		user.Login = string(randLoginBytes)
	}

	if password == "" {
		randPasswordBytes, err := randBytesFromStr(20, alphaNumBytes)
		if err != nil {
			return nil, err
		}
		password = string(randPasswordBytes)
	} else {
		if svc.Config.MinPasswordEntropy > 0 {
			entropy := passwordvalidator.GetEntropy(password)
			if entropy < float64(svc.Config.MinPasswordEntropy) {
				return nil, fmt.Errorf("password entropy is too low (%f), required is %d", entropy, svc.Config.MinPasswordEntropy)
			}
		}
	}

	// we only store the hashed password but return the initial plain text password in the HTTP response
	hashedPassword := security.HashPassword(password)
	user.Password = hashedPassword

	// Create user and the user's accounts
	// We use double-entry bookkeeping so we use 4 accounts: incoming, current, outgoing and fees
	// Wrapping this in a transaction in case something fails
	err = svc.DB.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(user).Exec(ctx); err != nil {
			return err
		}
		accountTypes := []string{
			common.AccountTypeIncoming,
			common.AccountTypeCurrent,
			common.AccountTypeOutgoing,
			common.AccountTypeFees,
		}
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

func (svc *LndhubService) UpdateUser(ctx context.Context, userId int64, login *string, password *string, deactivated *bool) (user *models.User, err error) {
	user, err = svc.FindUser(ctx, userId)
	if err != nil {
		return nil, err
	}
	if login != nil {
		user.Login = *login
	}
	if password != nil {
		if svc.Config.MinPasswordEntropy > 0 {
			entropy := passwordvalidator.GetEntropy(*password)
			if entropy < float64(svc.Config.MinPasswordEntropy) {
				return nil, fmt.Errorf("password entropy is too low (%f), required is %d", entropy, svc.Config.MinPasswordEntropy)
			}
		}
		hashedPassword := security.HashPassword(*password)
		user.Password = hashedPassword
	}
	if deactivated != nil {
		user.Deactivated = *deactivated
	}
	_, err = svc.DB.NewUpdate().Model(user).WherePK().Exec(ctx)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (svc *LndhubService) FindUser(ctx context.Context, userId int64) (*models.User, error) {
	var user models.User

	err := svc.DB.NewSelect().Model(&user).Where("id = ?", userId).Limit(1).Scan(ctx)
	if err != nil {
		return &user, err
	}
	return &user, nil
}

func (svc *LndhubService) FindUserByLogin(ctx context.Context, login string) (*models.User, error) {
	var user models.User

	err := svc.DB.NewSelect().Model(&user).Where("login = ?", login).Limit(1).Scan(ctx)
	if err != nil {
		return &user, err
	}
	return &user, nil
}

func (svc *LndhubService) CheckOutgoingPaymentAllowed(ctx context.Context, lnpayReq *lnd.LNPayReq, userId int64) (result *responses.ErrorResponse, err error) {
	if svc.Config.MaxSendAmount > 0 {
		if lnpayReq.PayReq.NumSatoshis > svc.Config.MaxSendAmount {
			svc.Logger.Errorf("Max send amount exceeded for user_id %v (amount:%v)", userId, lnpayReq.PayReq.NumSatoshis)
			return &responses.SendExceededError, nil
		}
	}

	currentBalance, err := svc.CurrentUserBalance(ctx, userId)
	if err != nil {
		svc.Logger.Errorj(
			log.JSON{
				"message":        "error checking balance",
				"error":          err,
				"lndhub_user_id": userId,
			},
		)
		return nil, err
	}

	minimumBalance := lnpayReq.PayReq.NumSatoshis
	if svc.Config.FeeReserve {
		minimumBalance += svc.CalcFeeLimit(lnpayReq.PayReq.Destination, lnpayReq.PayReq.NumSatoshis)
	}
	if currentBalance < minimumBalance {
		return &responses.NotEnoughBalanceError, nil
	}

	return svc.CheckVolumeAllowed(ctx, userId, common.InvoiceTypeOutgoing)
}

func (svc *LndhubService) CheckIncomingPaymentAllowed(ctx context.Context, amount, userId int64) (result *responses.ErrorResponse, err error) {
	if svc.Config.MaxReceiveAmount > 0 {
		if amount > svc.Config.MaxReceiveAmount {
			svc.Logger.Errorf("Max receive amount exceeded for user_id %d", userId)
			return &responses.ReceiveExceededError, nil 
		}
	}

	if svc.Config.MaxAccountBalance > 0 {
		currentBalance, err := svc.CurrentUserBalance(ctx, userId)
		if err != nil {
			svc.Logger.Errorj(
				log.JSON{
					"message":        "error fetching balance",
					"lndhub_user_id": userId,
					"error":          err,
				},
			)
			return nil, err
		}
		if currentBalance+amount > svc.Config.MaxAccountBalance {
			svc.Logger.Errorf("Max account balance exceeded for user_id %d", userId)
			return &responses.BalanceExceededError, nil
		}
	}

	return svc.CheckVolumeAllowed(ctx, userId, common.InvoiceTypeIncoming)
}

func (svc *LndhubService) CheckVolumeAllowed(ctx context.Context, userId int64, invoiceType string) (result *responses.ErrorResponse, err error) {
	var maxVolume int64
	if invoiceType == common.InvoiceTypeIncoming {
			maxVolume = svc.Config.MaxReceiveVolume
	} else {
			maxVolume = svc.Config.MaxSendVolume
	}
	if maxVolume > 0 {
		volume, err := svc.GetVolumeOverPeriod(ctx, userId, invoiceType, time.Duration(svc.Config.MaxVolumePeriod*int64(time.Second)))
		if err != nil {
			svc.Logger.Errorj(
				log.JSON{
					"message":        "error fetching volume",
					"error":          err,
					"lndhub_user_id": userId,
				},
			)
			return nil, err
		}
		if volume > maxVolume {
			svc.Logger.Errorf("Transaction volume exceeded for user_id %d", userId)
			sentry.CaptureMessage(fmt.Sprintf("transaction volume exceeded for user %d", userId))
			return &responses.TooMuchVolumeError, nil
		}
	}
	return nil, nil
}


func (svc *LndhubService) CalcFeeLimit(destination string, amount int64) int64 {
	if svc.LndClient.IsIdentityPubkey(destination) {
		return 0
	}
	limit := int64(10)
	if amount > 1000 {
		limit = int64(math.Ceil(float64(amount)*float64(0.01)) + 1)
	}
	if limit > svc.Config.MaxFeeAmount {
		limit = svc.Config.MaxFeeAmount
	}
	return limit
}

func (svc *LndhubService) CurrentUserBalance(ctx context.Context, userId int64) (int64, error) {
	var balance int64

	account, err := svc.AccountFor(ctx, common.AccountTypeCurrent, userId)
	if err != nil {
		return balance, err
	}
	err = svc.DB.NewSelect().Table("account_ledgers").ColumnExpr("sum(account_ledgers.amount) as balance").Where("account_ledgers.account_id = ?", account.ID).Scan(ctx, &balance)
	return balance, err
}

func (svc *LndhubService) AccountFor(ctx context.Context, accountType string, userId int64) (models.Account, error) {
	account := models.Account{}
	err := svc.DB.NewSelect().Model(&account).Where("user_id = ? AND type= ?", userId, accountType).Limit(1).Scan(ctx)
	return account, err
}

func (svc *LndhubService) TransactionEntriesFor(ctx context.Context, userId int64) ([]models.TransactionEntry, error) {
	transactionEntries := []models.TransactionEntry{}
	err := svc.DB.NewSelect().Model(&transactionEntries).Where("user_id = ?", userId).Scan(ctx)
	return transactionEntries, err
}

func (svc *LndhubService) InvoicesFor(ctx context.Context, userId int64, invoiceType string) ([]models.Invoice, error) {
	var invoices []models.Invoice

	query := svc.DB.NewSelect().Model(&invoices).Where("user_id = ?", userId)
	if invoiceType != "" {
		query.Where("type = ? AND state <> ?", invoiceType, common.InvoiceStateInitialized)
	}
	query.OrderExpr("id DESC").Limit(100)
	err := query.Scan(ctx)
	if err != nil {
		return nil, err
	}
	return invoices, nil
}

func (svc *LndhubService) GetVolumeOverPeriod(ctx context.Context, userId int64, invoiceType string, period time.Duration) (result int64, err error) {

	err = svc.DB.NewSelect().Table("invoices").
		ColumnExpr("sum(invoices.amount) as result").
		Where("invoices.user_id = ?", userId).
		Where("invoices.type = ?", invoiceType).
		Where("invoices.settled_at >= ?", time.Now().Add(-1*period)).
		Scan(ctx, &result)
	if err != nil {
		return 0, err
	}
	fmt.Println(result, "volume ")
	return result, nil
}
