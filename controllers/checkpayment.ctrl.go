package controllers

import (
	"net/http"

	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
)

// CheckPaymentController : CheckPaymentController struct
type CheckPaymentController struct {
	svc *service.LndhubService
}

type CheckPaymentResponseBody struct {
	IsPaid bool `json:"paid"`
}

func NewCheckPaymentController(svc *service.LndhubService) *CheckPaymentController {
	return &CheckPaymentController{svc: svc}
}

// CheckPayment : Check Payment Controller
func (controller *CheckPaymentController) CheckPayment(c echo.Context) error {
	userId := c.Get("UserID").(int64)
	rHash := c.Param("payment_hash")

	invoice, err := controller.svc.FindInvoiceByPaymentHash(c.Request().Context(), userId, rHash)

	// Probably we did not find the invoice
	if err != nil {
		c.Logger().Errorf("Invalid checkpayment request payment_hash=%s", rHash)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	responseBody := &CheckPaymentResponseBody{}
	responseBody.IsPaid = !invoice.SettledAt.IsZero()
	return c.JSON(http.StatusOK, &responseBody)
}
