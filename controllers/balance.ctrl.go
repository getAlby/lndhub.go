package controllers

import (
	"net/http"

	"github.com/getAlby/lndhub.go/lib/service"
	plugin "github.com/getAlby/lndhub.go/plugins"
	"github.com/labstack/echo/v4"
)

// BalanceController : BalanceController struct
type BalanceController struct {
	svc *service.LndhubService
}

func NewBalanceController(svc *service.LndhubService) *BalanceController {
	return &BalanceController{svc: svc}
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
	v, err := plugin.CreatePlugin("plugins/middleware_example.go", "plugin.ProcessBalanceResponse")
	if err != nil {
		return err
	}
	fu := v.Interface().(func(in int64, svc *service.LndhubService) (int64, error))
	balance, err = fu(balance, controller.svc)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, &BalanceResponse{
		BTC: struct{ AvailableBalance int64 }{
			AvailableBalance: balance,
		},
	})
}
