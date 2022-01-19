package controllers

import (
	"net/http"

	"github.com/getAlby/lndhub.go/lib"
	"github.com/labstack/echo/v4"
)

// CheckPaymentController : CheckPaymentController struct
type CheckPaymentController struct{}

func NewCheckPaymentController(svc *lib.LndhubService) *CheckPaymentController {
	return &CheckPaymentController{}
}

// CheckPayment : Check Payment Controller
func (CheckPaymentController) CheckPayment(c echo.Context) error {
	_ = c.Param("payment_hash")

	return c.JSON(http.StatusBadRequest, echo.Map{
		"paid": true,
	})
}
