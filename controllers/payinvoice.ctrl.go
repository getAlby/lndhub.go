package controllers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
)

// PayInvoiceController : Pay invoice controller struct
type PayInvoiceController struct {
	svc *service.LndhubService
}

func NewPayInvoiceController(svc *service.LndhubService) *PayInvoiceController {
	return &PayInvoiceController{svc: svc}
}

// PayInvoice : Pay invoice Controller
func (controller *PayInvoiceController) PayInvoice(c echo.Context) error {
	userID := c.Get("UserID").(int64)
	var reqBody struct {
		Invoice string `json:"invoice" validate:"required"`
		Amount  int    `json:"amount" validate:"omitempty,gte=0"`
	}

	if err := c.Bind(&reqBody); err != nil {
		c.Logger().Errorf("Failed to load payinvoice request body: %v", err)
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":   true,
			"code":    8,
			"message": "Bad arguments",
		})
	}

	if err := c.Validate(&reqBody); err != nil {
		c.Logger().Errorf("Invalid payinvoice request body: %v", err)
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":   true,
			"code":    8,
			"message": "Bad arguments",
		})
	}

	paymentRequest := reqBody.Invoice
	decodedPaymentRequest, err := controller.svc.DecodePaymentRequest(paymentRequest)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":   true,
			"code":    8,
			"message": "Bad arguments",
		})
	}
	c.Logger().Info("%v", decodedPaymentRequest)

	invoice, err := controller.svc.AddOutgoingInvoice(userID, paymentRequest, *decodedPaymentRequest)
	if err != nil {
		c.Logger().Errorf("Error creating invoice: %v", err)
		// TODO: sentry notification
		return c.JSON(http.StatusInternalServerError, nil)
	}

	currentBalance, err := controller.svc.CurrentUserBalance(context.TODO(), userID)
	if err != nil {
		return err
	}

	if currentBalance < invoice.Amount {
		c.Logger().Errorf("User does not have enough balance invoice_id=%v user_id=%v balance=%v amount=%v", invoice.ID, userID, currentBalance, invoice.Amount)

		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":   true,
			"code":    2,
			"message": fmt.Sprintf("not enough balance (%v). Make sure you have at least 1%% reserved for potential fees", currentBalance),
		})
	}

	entry, err := controller.svc.PayInvoice(invoice)
	if err != nil {
		c.Logger().Errorf("Failed: %v", err)
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":   true,
			"code":    10,
			"message": fmt.Sprintf("Payment failed. Does the receiver have enough inbound capacity? (%v)", err),
		})
	}
	return c.JSON(http.StatusOK, &entry)
}
