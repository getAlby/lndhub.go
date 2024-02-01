package v2controllers

import (
	"net/http"
	"strconv"

	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
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
// @Router       /v2/balance/:asset_id [get]
// @Security     OAuth2Password
func (controller *BalanceController) Balance(c echo.Context) error {
	userId := c.Get("UserID").(int64)
	assetParam := c.Param("asset_id")
	assetId, err := strconv.ParseInt(assetParam, 10, 64)
	// default to bitcoin if error parsing the param
	if  err != nil {
		assetId = 1
	}
	balance, err := controller.svc.CurrentUserBalance(c.Request().Context(), assetId, userId)
	if err != nil {
		c.Logger().Errorj(
			log.JSON{
				"message":        "failed to retrieve user balance",
				"lndhub_user_id": userId,
				"error":          err,
			},
		)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}
	return c.JSON(http.StatusOK, &BalanceResponse{
		Balance:  balance,
		Currency: "BTC",
		Unit:     "sat",
	})
}
