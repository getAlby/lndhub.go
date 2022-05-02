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

// CheckPayment godoc
// @Summary      Check if an invoice is paid
// @Description  Checks if an invoice is paid, can be incoming our outgoing
// @Accept       json
// @Produce      json
// @Tags         Invoice
// @Param        payment_hash  path      string  true  "Payment hash"
// @Success      200           {object}  CheckPaymentResponseBody
// @Failure      400           {object}  responses.ErrorResponse
// @Failure      500           {object}  responses.ErrorResponse
// @Router       /checkpayment/{payment_hash} [get]
// @Security     OAuth2Password
func (controller *CheckPaymentController) CheckPayment(c echo.Context) error {
	userID := c.Get("UserID").(int64)
	rHash := c.Param("payment_hash")

	invoice, err := controller.svc.FindInvoiceByPaymentHash(c.Request().Context(), userID, rHash)

	// Probably we did not find the invoice
	if err != nil {
		c.Logger().Errorf("Invalid checkpayment request user_id:%v payment_hash:%s", userID, rHash)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	responseBody := &CheckPaymentResponseBody{}
	responseBody.IsPaid = !invoice.SettledAt.IsZero()
	return c.JSON(http.StatusOK, &responseBody)
}
