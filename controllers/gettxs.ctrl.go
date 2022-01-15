package controllers

import (
	"net/http"

	"github.com/bumi/lndhub.go/database/models"
	"github.com/labstack/echo/v4"
)

// GetTXSController : GetTXSController struct
type GetTXSController struct{}

// GetTXS : Get TXS Controller
func (GetTXSController) GetTXS(c echo.Context) error {
	return c.JSON(http.StatusOK, &models.Invoice{})
}
