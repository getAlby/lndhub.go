package controllers

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

// PayInvoiceController : Pay invoice controller struct
type PayInvoiceController struct{}

// PayInvoice : Pay invoice Controller
func (PayInvoiceController) PayInvoice(c echo.Context) error {
	var reqBody struct {
		Invoice string `json:"invoice" validate:"required"`
		Amount  int    `json:"amount" validate:"gt=0"`
	}

	if err := c.Bind(&reqBody); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": "failed to bind json",
		})
	}

	if err := c.Validate(&reqBody); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": "invalid request",
		})
	}

	return nil
}
