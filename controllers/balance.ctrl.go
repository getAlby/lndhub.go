package controllers

import (
	"context"
	"net/http"

	"github.com/getAlby/lndhub.go/lib"
	"github.com/labstack/echo/v4"
	"github.com/lightningnetwork/lnd/lnrpc"
)

// BalanceController : BalanceController struct
type BalanceController struct{}

// Balance : Balance Controller
func (BalanceController) Balance(c echo.Context) error {
	ctx := c.(*lib.LndhubContext)
	c.Logger().Warn(ctx.User.ID)
	lndClient := *ctx.LndClient
	getInfo, err := lndClient.GetInfo(context.TODO(), &lnrpc.GetInfoRequest{})
	if err != nil {
		panic(err)
	}
	c.Logger().Infof("Connected to LND: %s - %s", getInfo.Alias, getInfo.IdentityPubkey)

	return c.JSON(http.StatusOK, echo.Map{
		"BTC": echo.Map{
			"AvailableBalance": 1,
		},
	})
}
