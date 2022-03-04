package controllers

import (
	"fmt"
	"net/http"

	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/getsentry/sentry-go"
	"github.com/labstack/echo/v4"
)

// PayInvoiceController : Pay invoice controller struct
type PayInvoiceController struct {
	svc *service.LndhubService
}

func NewPayInvoiceController(svc *service.LndhubService) *PayInvoiceController {
	return &PayInvoiceController{svc: svc}
}

type PayInvoiceRequestBody struct {
	Invoice string      `json:"invoice" validate:"required"`
	Amount  interface{} `json:"amount" validate:"omitempty"`
}
type PayInvoiceResponseBody struct {
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

// PayInvoice : Pay invoice Controller
func (controller *PayInvoiceController) PayInvoice(c echo.Context) error {
	userID := c.Get("UserID").(int64)
	reqBody := PayInvoiceRequestBody{}
	if err := c.Bind(&reqBody); err != nil {
		c.Logger().Errorf("Failed to load payinvoice request body: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	if err := c.Validate(&reqBody); err != nil {
		c.Logger().Errorf("Invalid payinvoice request body: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	paymentRequest := reqBody.Invoice
	decodedPaymentRequest, err := controller.svc.DecodePaymentRequest(c.Request().Context(), paymentRequest)
	if err != nil {
		c.Logger().Errorf("Invalid payment request: %v", err)
		sentry.CaptureException(err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
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

	invoice, err := controller.svc.AddOutgoingInvoice(c.Request().Context(), userID, paymentRequest, decodedPaymentRequest, false)
	if err != nil {
		return err
	}

	currentBalance, err := controller.svc.CurrentUserBalance(c.Request().Context(), userID)
	if err != nil {
		return err
	}

	if currentBalance < invoice.Amount {
		c.Logger().Errorf("User does not have enough balance invoice_id=%v user_id=%v balance=%v amount=%v", invoice.ID, userID, currentBalance, invoice.Amount)

		return c.JSON(http.StatusBadRequest, responses.NotEnoughBalanceError)
	}

	sendPaymentResponse, err := controller.svc.PayInvoice(c.Request().Context(), invoice)
	if err != nil {
		c.Logger().Errorf("Payment failed: %v", err)
		sentry.CaptureException(err)
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":   true,
			"code":    10,
			"message": fmt.Sprintf("Payment failed. Does the receiver have enough inbound capacity? (%v)", err),
		})
	}
	responseBody := &PayInvoiceResponseBody{}
	responseBody.RHash = &lib.JavaScriptBuffer{Data: sendPaymentResponse.PaymentHash}
	responseBody.PaymentRequest = paymentRequest
	responseBody.PayReq = paymentRequest
	responseBody.Amount = invoice.Amount
	responseBody.Description = invoice.Memo
	responseBody.DescriptionHashStr = invoice.DescriptionHash
	responseBody.PaymentError = sendPaymentResponse.PaymentError
	responseBody.PaymentPreimage = &lib.JavaScriptBuffer{Data: sendPaymentResponse.PaymentPreimage}
	responseBody.PaymentRoute = sendPaymentResponse.PaymentRoute

	return c.JSON(http.StatusOK, responseBody)
}
