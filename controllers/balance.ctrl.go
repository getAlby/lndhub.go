package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// BalanceController : BalanceController struct
type BalanceController struct{}

// Balance : Balance Controller
func (BalanceController) Balance(c echo.Context) error {
	return c.JSON(http.StatusOK, echo.Map{
		"BTC": echo.Map{
			"AvailableBalance": 1,
		},
	})
}
