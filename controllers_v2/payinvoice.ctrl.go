package v2controllers

import (
	"net/http"
	"strings"
	"time"

	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/getAlby/lndhub.go/lnd"
	"github.com/getsentry/sentry-go"
	sentryecho "github.com/getsentry/sentry-go/echo"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

// PayInvoiceController : Pay invoice controller struct
type PayInvoiceController struct {
	svc *service.LndhubService
}

func NewPayInvoiceController(svc *service.LndhubService) *PayInvoiceController {
	return &PayInvoiceController{svc: svc}
}

type PayInvoiceRequestBody struct {
	Invoice string `json:"invoice" validate:"required"`
	Amount  int64  `json:"amount" validate:"omitempty,gte=0"`
}
type PayInvoiceResponseBody struct {
	PaymentRequest  string `json:"payment_request,omitempty"`
	Amount          int64  `json:"amount,omitempty"`
	Fee             int64  `json:"fee"`
	Description     string `json:"description,omitempty"`
	DescriptionHash string `json:"description_hash,omitempty"`
	Destination     string `json:"destination,omitempty"`
	PaymentPreimage string `json:"payment_preimage,omitempty"`
	PaymentHash     string `json:"payment_hash,omitempty"`
}

// PayInvoice godoc
// @Summary      Pay an invoice
// @Description  Pay a bolt11 invoice
// @Accept       json
// @Produce      json
// @Tags         Payment
// @Param        PayInvoiceRequest  body      PayInvoiceRequestBody  True  "Invoice to pay"
// @Success      200                {object}  PayInvoiceResponseBody
// @Failure      400                {object}  responses.ErrorResponse
// @Failure      500                {object}  responses.ErrorResponse
// @Router       /v2/payments/bolt11 [post]
// @Security     OAuth2Password
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
		if strings.Contains(err.Error(), "invoice not for current active network") {
			c.Logger().Errorf("Incorrect network user_id:%v error: %v", userID, err)
			return c.JSON(http.StatusBadRequest, responses.IncorrectNetworkError)
		}
		c.Logger().Errorf("Invalid payment request user_id:%v error: %v", userID, err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	lnPayReq := &lnd.LNPayReq{
		PayReq:  decodedPaymentRequest,
		Keysend: false,
	}
	if (decodedPaymentRequest.Timestamp + decodedPaymentRequest.Expiry) < time.Now().Unix() {
		c.Logger().Errorf("Payment request expired")
		return c.JSON(http.StatusBadRequest, responses.InvoiceExpiredError)
	}

	if decodedPaymentRequest.NumSatoshis == 0 {
		amt, err := controller.svc.ParseInt(reqBody.Amount)
		if err != nil || amt <= 0 {
			c.Logger().Errorj(
				log.JSON{
					"message":        "invalid amount",
					"error":          err,
					"lndhub_user_id": userID,
				},
			)
			return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
		}
		lnPayReq.PayReq.NumSatoshis = amt
	}
	resp, err := controller.svc.CheckOutgoingPaymentAllowed(c, lnPayReq, userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, responses.GeneralServerError)
	}
	if resp != nil {
		c.Logger().Errorf("Error: %v user_id:%v amount:%v", resp.Message, userID, lnPayReq.PayReq.NumSatoshis)
		return c.JSON(resp.HttpStatusCode, resp)
	}
	invoice, errResp := controller.svc.AddOutgoingInvoice(c.Request().Context(), userID, paymentRequest, lnPayReq)
	if errResp != nil {
		return c.JSON(errResp.HttpStatusCode, errResp)
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
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error":   true,
			"code":    10,
			"message": err.Error(),
		})
	}
	responseBody := &PayInvoiceResponseBody{
		PaymentRequest:  paymentRequest,
		Amount:          invoice.Amount,
		Fee:             invoice.Fee,
		Description:     invoice.Memo,
		DescriptionHash: invoice.DescriptionHash,
		Destination:     invoice.DestinationPubkeyHex,
		PaymentPreimage: sendPaymentResponse.PaymentPreimageStr,
		PaymentHash:     sendPaymentResponse.PaymentHashStr,
	}

	return c.JSON(http.StatusOK, responseBody)
}
