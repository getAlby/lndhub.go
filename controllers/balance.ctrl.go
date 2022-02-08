package controllers

import (
	"net/http"

	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
)

// BalanceController : BalanceController struct
type BalanceController struct {
	svc *service.LndhubService
}

func NewBalanceController(svc *service.LndhubService) *BalanceController {
	return &BalanceController{svc: svc}
}

// Balance : Balance Controller
func (controller *BalanceController) Balance(c echo.Context) error {
	userId := c.Get("UserID").(int64)
	balance, err := controller.svc.CurrentUserBalance(c.Request().Context(), userId)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, echo.Map{
		"BTC": echo.Map{
			"AvailableBalance": balance,
		},
	})
}
