package controllers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/getAlby/lndhub.go/lib"
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
		Invoice string      `json:"invoice" validate:"required"`
		Amount  interface{} `json:"amount" validate:"omitempty"`
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
	// TODO: zero amount invoices
	/*
		_, err = controller.svc.ParseInt(reqBody.Amount)
		if err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{
				"error":   true,
				"code":    8,
				"message": "Bad arguments",
			})
		}
	*/

	invoice, err := controller.svc.AddOutgoingInvoice(userID, paymentRequest, *decodedPaymentRequest)
	if err != nil {
		c.Logger().Errorf("Error creating invoice: %v", err)
		// TODO: sentry notification
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error":   true,
			"code":    6,
			"message": "Something went wrong. Please try again later",
		})
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

	sendPaymentResponse, err := controller.svc.PayInvoice(invoice)
	if err != nil {
		c.Logger().Errorf("Payment failed: %v", err)
		// TODO: sentry notification
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":   true,
			"code":    10,
			"message": fmt.Sprintf("Payment failed. Does the receiver have enough inbound capacity? (%v)", err),
		})
	}

	var responseBody struct {
		RHash              *lib.JavaScriptBuffer `json:"payment_hash,omitempty"`
		PaymentRequest     string                `json:"payment_request,omitempty"`
		PayReq             string                `json:"pay_req,omitempty"`
		Amount             int64                 `json:"num_satoshis,omitempty"`
		Description        string                `json:"description,omitempty"`
		DescriptionHashStr string                `json:"description_hash,omitempty"`
		PaymentError       string                `json:"payment_error,omitempty"`
		PaymentPreimage    *lib.JavaScriptBuffer `json:"payment_preimage,omitempty"`
		PaymentRoute       *service.Route        `json:"route,omitempty"`
	}

	responseBody.RHash = &lib.JavaScriptBuffer{Data: sendPaymentResponse.PaymentHash}
	responseBody.PaymentRequest = paymentRequest
	responseBody.PayReq = paymentRequest
	responseBody.Amount = invoice.Amount
	responseBody.Description = invoice.Memo
	responseBody.DescriptionHashStr = invoice.DescriptionHash
	responseBody.PaymentError = sendPaymentResponse.PaymentError
	responseBody.PaymentPreimage = &lib.JavaScriptBuffer{Data: sendPaymentResponse.PaymentPreimage}
	responseBody.PaymentRoute = sendPaymentResponse.PaymentRoute

	return c.JSON(http.StatusOK, &responseBody)
}
