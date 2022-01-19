package controllers

import (
	"context"
	"net/http"

	"github.com/getAlby/lndhub.go/lib"
	"github.com/labstack/echo/v4"
)

// BalanceController : BalanceController struct
type BalanceController struct{}

// Balance : Balance Controller
func (BalanceController) Balance(c echo.Context) error {
	ctx := c.(*lib.LndhubService)
	c.Logger().Warn(ctx.User.ID)

	db := ctx.DB

	balance, err := ctx.User.CurrentBalance(context.TODO(), db)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, echo.Map{
		"BTC": echo.Map{
			"AvailableBalance": balance,
		},
	})
}
