package controllers

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

// CheckPaymentController : CheckPaymentController struct
type CheckPaymentController struct{}

// CheckPayment : Check Payment Controller
func (CheckPaymentController) CheckPayment(c echo.Context) error {
	_ = c.Param("payment_hash")

	return c.JSON(http.StatusBadRequest, echo.Map{
		"paid": true,
	})
}
