package controllers

import (
	"net/http"

	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
)

// BalanceController : BalanceController struct
type BalanceController struct {
	svc    *service.LndhubService
	plugin func(int64, *service.LndhubService) (int64, error)
}

func NewBalanceController(svc *service.LndhubService, plug func(int64, *service.LndhubService) (int64, error)) *BalanceController {
	return &BalanceController{svc: svc, plugin: plug}
}

type BalanceResponse struct {
	BTC struct {
		AvailableBalance int64
	}
}

// Balance : Balance Controller
func (controller *BalanceController) Balance(c echo.Context) error {
	userId := c.Get("UserID").(int64)
	balance, err := controller.svc.CurrentUserBalance(c.Request().Context(), userId)
	if err != nil {
		return err
	}

	balance, err = controller.plugin(balance, controller.svc)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, &BalanceResponse{
		BTC: struct{ AvailableBalance int64 }{
			AvailableBalance: balance,
		},
	})
}
