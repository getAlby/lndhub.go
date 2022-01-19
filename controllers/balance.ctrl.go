package controllers

import (
	"context"
	"net/http"

	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/lib"
	"github.com/labstack/echo/v4"
)

// BalanceController : BalanceController struct
type BalanceController struct{}

// Balance : Balance Controller
func (BalanceController) Balance(c echo.Context) error {
	ctx := c.(*lib.LndhubContext)
	c.Logger().Warn(ctx.User.ID)

	db := ctx.DB

	// load user's current account
	account := models.Account{}
	if err := db.NewSelect().Model(&account).Where("user_id = ? AND type= ?", ctx.User.ID, "current").Limit(1).Scan(context.TODO()); err != nil {
		// TODO: proper error
		return err
	}
	// calculate the account balance
	var balance int64
	if err := db.NewSelect().Table("account_ledgers").ColumnExpr("sum(account_ledgers.amount) as balance").Where("account_ledgers.account_id = ?", account.ID).Scan(context.TODO(), &balance); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, echo.Map{
		"BTC": echo.Map{
			"AvailableBalance": balance,
		},
	})
}
