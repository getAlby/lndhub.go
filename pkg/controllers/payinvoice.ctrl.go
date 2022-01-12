package controllers

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

// PayInvoiceController : Pay invoice controller struct
type PayInvoiceController struct{}

// PayInvoice : Pay invoice Controller
func (PayInvoiceController) PayInvoice(c echo.Context) error {
	var body struct {
		ID      uint   `json:"id"`
		Invoice string `json:"invoice" validate:"required"`
		Amount  int    `json:"amount" validate:"gt=0"`
	}

	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": "failed to bind json",
		})
	}

	if err := c.Validate(&body); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": "invalid request",
		})
	}

	return nil
}
