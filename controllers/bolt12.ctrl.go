package controllers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/getsentry/sentry-go"
	"github.com/labstack/echo/v4"
)

// Bolt12Controller : Bolt12Controller struct
type Bolt12Controller struct {
	svc *service.LndhubService
}
type FetchInvoiceRequestBody struct {
	Amount int64  `json:"amt" validate:"required"` // todo: validate properly, amount not strictly needed always amount in Satoshi
	Memo   string `json:"memo"`
	Offer  string `json:"offer" validate:"required"`
}

func NewBolt12Controller(svc *service.LndhubService) *Bolt12Controller {
	return &Bolt12Controller{svc: svc}
}

// Decode : Decode handler
func (controller *Bolt12Controller) Decode(c echo.Context) error {
	offer := c.Param("offer")
	decoded, err := controller.svc.LndClient.DecodeBolt12(context.TODO(), offer)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, decoded)
}

// FetchInvoice: fetches an invoice from a bolt12 offer for a certain amount
func (controller *Bolt12Controller) FetchInvoice(c echo.Context) error {
	var body FetchInvoiceRequestBody

	if err := c.Bind(&body); err != nil {
		c.Logger().Errorf("Failed to load fetchinvoice request body: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	if err := c.Validate(&body); err != nil {
		c.Logger().Errorf("Invalid fetchinvoice request body: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	invoice, err := controller.svc.FetchBolt12Invoice(context.TODO(), body.Offer, body.Memo, body.Amount)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, invoice)
}

// PayOffer: fetches an invoice from a bolt12 offer for a certain amount, and pays it
func (controller *Bolt12Controller) PayBolt12(c echo.Context) error {
	userID := c.Get("UserID").(int64)
	var body FetchInvoiceRequestBody

	if err := c.Bind(&body); err != nil {
		c.Logger().Errorf("Failed to load fetchinvoice request body: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	if err := c.Validate(&body); err != nil {
		c.Logger().Errorf("Invalid fetchinvoice request body: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	bolt12, err := controller.svc.FetchBolt12Invoice(c.Request().Context(), body.Offer, body.Memo, body.Amount)
	if err != nil {
		return err
	}
	decodedPaymentRequest, err := controller.svc.TransformBolt12(bolt12)
	if err != nil {
		c.Logger().Errorf("Invalid payment request: %v", err)
		sentry.CaptureException(err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	invoice, err := controller.svc.AddOutgoingInvoice(c.Request().Context(), userID, bolt12.Encoded, decodedPaymentRequest)
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
	responseBody.PaymentRequest = bolt12.Encoded
	responseBody.PayReq = bolt12.Encoded
	responseBody.Description = bolt12.PayerNote
	responseBody.PaymentError = sendPaymentResponse.PaymentError
	responseBody.PaymentPreimage = &lib.JavaScriptBuffer{Data: sendPaymentResponse.PaymentPreimage}
	responseBody.PaymentRoute = sendPaymentResponse.PaymentRoute

	return c.JSON(http.StatusOK, &responseBody)
}
