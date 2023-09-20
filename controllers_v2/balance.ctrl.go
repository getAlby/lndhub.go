package v2controllers

import (
	"net/http"

	"github.com/getAlby/lndhub.go/lib/responses"
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
	Balance  int64  `json:"balance"`
	Currency string `json:"currency"`
	Unit     string `json:"unit"`
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
// @Router       /v2/balance [get]
// @Security     OAuth2Password
func (controller *BalanceController) Balance(c echo.Context) error {
	userId := c.Get("UserID").(int64)
	balance, err := controller.svc.CurrentUserBalance(c.Request().Context(), userId)
	if err != nil {
		c.Logger().Errorf("Error fetching balance for user_id:%v error: %v", userId, err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}
	return c.JSON(http.StatusOK, &BalanceResponse{
		Balance:  balance,
		Currency: "BTC",
		Unit:     "sat",
	})
}
