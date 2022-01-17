package controllers

import (
	"net/http"

	"github.com/bumi/lndhub.go/lib"
	"github.com/labstack/echo/v4"
)

// BalanceController : BalanceController struct
type BalanceController struct{}

// Balance : Balance Controller
func (BalanceController) Balance(c echo.Context) error {
	ctx := c.(*lib.LndhubContext)
	c.Logger().Warn(ctx.User.ID)
	return c.JSON(http.StatusOK, echo.Map{
		"BTC": echo.Map{
			"AvailableBalance": 1,
		},
	})
}
