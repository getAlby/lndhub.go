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

type BalanceResponse struct {
	BTC struct {
		AvailableBalance int64
	}
}

// Balance godoc
// @Summary      Retrieve balance
// @Description  Current user's balance in satoshi
// @Accept       json
// @Produce      json
// @Tags         Account
// @Success      200  {object}  BalanceResponse
// @Failure      400  {object}  responses.ErrorResponse
// @Failure      500  {object}  responses.ErrorResponse
// @Router       /balance [get]
// @Security     OAuth2Password
func (controller *BalanceController) Balance(c echo.Context) error {
	userId := c.Get("UserID").(int64)
	balance, err := controller.svc.CurrentUserBalance(c.Request().Context(), userId)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, &BalanceResponse{
		BTC: struct{ AvailableBalance int64 }{
			AvailableBalance: balance,
		},
	})
}
