package controllers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/getAlby/lndhub.go/lnd"
	"github.com/getsentry/sentry-go"
	sentryecho "github.com/getsentry/sentry-go/echo"
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
	PaymentError       string                `json:"payment_error"`
	PaymentPreimage    *lib.JavaScriptBuffer `json:"payment_preimage,omitempty"`
	PaymentRoute       *service.Route        `json:"payment_route,omitempty"`
}

func (controller *PayInvoiceController) PayInvoice(c echo.Context) error {
	userID := c.Get("UserID").(int64)
	reqBody := PayInvoiceRequestBody{}
	if err := c.Bind(&reqBody); err != nil {
		c.Logger().Errorf("Failed to load payinvoice request body: user_id:%v error: %v", userID, err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	if err := c.Validate(&reqBody); err != nil {
		c.Logger().Errorf("Invalid payinvoice request body user_id:%v error: %v", userID, err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	paymentRequest := reqBody.Invoice
	paymentRequest = strings.ToLower(paymentRequest)
	decodedPaymentRequest, err := controller.svc.DecodePaymentRequest(c.Request().Context(), paymentRequest)
	if err != nil {
		c.Logger().Errorf("Invalid payment request user_id:%v error: %v", userID, err)
		sentry.CaptureException(err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	lnPayReq := &lnd.LNPayReq{
		PayReq:  decodedPaymentRequest,
		Keysend: false,
	}
	if decodedPaymentRequest.NumSatoshis == 0 {
		amt, err := controller.svc.ParseInt(reqBody.Amount)
		if err != nil || amt <= 0 {
			return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
		}
		lnPayReq.PayReq.NumSatoshis = amt
	}

	if controller.svc.Config.MaxSendAmount > 0 {
		if lnPayReq.PayReq.NumSatoshis > controller.svc.Config.MaxSendAmount {
			c.Logger().Errorf("Max send amount exceeded for user_id:%v (amount:%v)", userID, lnPayReq.PayReq.NumSatoshis)
			return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
		}
	}

	invoice, err := controller.svc.AddOutgoingInvoice(c.Request().Context(), userID, paymentRequest, lnPayReq)
	if err != nil {
		return err
	}

	currentBalance, err := controller.svc.CurrentUserBalance(c.Request().Context(), userID)
	if err != nil {
		controller.svc.DB.NewDelete().Where("id = ?", invoice.ID).Exec(c.Request().Context())
		return err
	}

	minimumBalance := invoice.Amount
	if controller.svc.Config.FeeReserve {
		minimumBalance += invoice.CalcFeeLimit(controller.svc.IdentityPubkey)
	}
	if currentBalance < minimumBalance {
		c.Logger().Errorf("User does not have enough balance invoice_id:%v user_id:%v balance:%v amount:%v", invoice.ID, userID, currentBalance, invoice.Amount)
		controller.svc.DB.NewDelete().Model(&invoice).Where("id = ?", invoice.ID).Exec(c.Request().Context())
		return c.JSON(http.StatusBadRequest, responses.NotEnoughBalanceError)
	}

	sendPaymentResponse, err := controller.svc.PayInvoice(c.Request().Context(), invoice)
	if err != nil {
		c.Logger().Errorf("Payment failed invoice_id:%v user_id:%v error: %v", invoice.ID, userID, err)
		if hub := sentryecho.GetHubFromContext(c); hub != nil {
			hub.WithScope(func(scope *sentry.Scope) {
				scope.SetExtra("invoice_id", invoice.ID)
				scope.SetExtra("destination_pubkey_hex", invoice.DestinationPubkeyHex)
				scope.SetExtra("payment_request", invoice.PaymentRequest)
				hub.CaptureException(err)
			})
		}
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
