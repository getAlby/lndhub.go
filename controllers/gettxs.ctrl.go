package controllers

import (
	"net/http"

	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
)

// GetTXSController : GetTXSController struct
type GetTXSController struct{}

func NewGetTXSController(svc *service.LndhubService) *GetTXSController {
	return &GetTXSController{}
}

// GetTXS : Get TXS Controller
func (GetTXSController) GetTXS(c echo.Context) error {
	transactions := []string{}
	return c.JSON(http.StatusOK, &transactions)
}

func (GetTXSController) GetUserInvoices(c echo.Context) error {
	transactions := []string{}
	return c.JSON(http.StatusOK, &transactions)
}
