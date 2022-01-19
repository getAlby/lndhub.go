package controllers

import (
	"net/http"

	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
)

// BlankController : Controller for endpoints that we do not support and simply return
//            a blank response for compatibility

// GetBtcController : GetBtcController struct
type BlankController struct{}

func NewBlankController(svc *service.LndhubService) *BlankController {
	return &BlankController{}
}

// We do NOT support onchain transactions thus we only return an empty array for backwards compatibility
func (controller *BlankController) GetBtc(c echo.Context) error {
	addresses := []string{}

	return c.JSON(http.StatusOK, &addresses)
}

func (controller *BlankController) GetPending(c echo.Context) error {
	addresses := []string{}

	return c.JSON(http.StatusOK, &addresses)
}
