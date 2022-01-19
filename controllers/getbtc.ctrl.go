package controllers

import (
	"net/http"

	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
)

// GetBtcController : GetBtcController struct
type GetBtcController struct{}

func NewGetBtcController(svc *service.LndhubService) *GetBtcController {
	return &GetBtcController{}
}

// GetBtc : Get Btc handler
//
// We do NOT support onchain transactions thus we only return an empty array for backwards compatibility
func (GetBtcController) GetBtc(c echo.Context) error {
	addresses := []string{}

	return c.JSON(http.StatusOK, &addresses)
}
