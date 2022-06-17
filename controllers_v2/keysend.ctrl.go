package v2controllers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/getAlby/lndhub.go/lnd"
	"github.com/getsentry/sentry-go"
	"github.com/labstack/echo/v4"
	"github.com/lightningnetwork/lnd/lnrpc"
)

// KeySendController : Key send controller struct
type KeySendController struct {
	svc *service.LndhubService
}

func NewKeySendController(svc *service.LndhubService) *KeySendController {
	return &KeySendController{svc: svc}
}

type KeySendRequestBody struct {
	Amount        int64             `json:"amount" validate:"required,gt=0"`
	Destination   string            `json:"destination" validate:"required"`
	Memo          string            `json:"memo" validate:"omitempty"`
	CustomRecords map[string]string `json:"customRecords" validate:"omitempty"`
}

type KeySendResponseBody struct {
	RHash              *lib.JavaScriptBuffer `json:"payment_hash,omitempty"`
	Amount             int64                 `json:"num_satoshis,omitempty"`
	Description        string                `json:"description,omitempty"`
	Destination        string                `json:"destination,omitempty"`
	DescriptionHashStr string                `json:"description_hash,omitempty"`
	PaymentError       string                `json:"payment_error,omitempty"`
	PaymentPreimage    *lib.JavaScriptBuffer `json:"payment_preimage,omitempty"`
	PaymentRoute       *service.Route        `json:"payment_route,omitempty"`
}

//// KeySend godoc
// @Summary      Make a keysend payment
// @Description  Pay a node without an invoice using it's public key
// @Accept       json
// @Produce      json
// @Tags         Payment
// @Param        KeySendRequestBody  body      KeySendRequestBody  True  "Invoice to pay"
// @Success      200                 {object}  KeySendResponseBody
// @Failure      400                 {object}  responses.ErrorResponse
// @Failure      500                 {object}  responses.ErrorResponse
// @Router       /keysend [post]
// @Security     OAuth2Password
func (controller *KeySendController) KeySend(c echo.Context) error {
	userID := c.Get("UserID").(int64)
	reqBody := KeySendRequestBody{}
	if err := c.Bind(&reqBody); err != nil {
		c.Logger().Errorf("Failed to load keysend request body: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	if err := c.Validate(&reqBody); err != nil {
		c.Logger().Errorf("Invalid keysend request body: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	lnPayReq := &lnd.LNPayReq{
		PayReq: &lnrpc.PayReq{
			Destination: reqBody.Destination,
			NumSatoshis: reqBody.Amount,
			Description: reqBody.Memo,
		},
		Keysend: true,
	}

	invoice, err := controller.svc.AddOutgoingInvoice(c.Request().Context(), userID, "", lnPayReq)
	if err != nil {
		return err
	}

	currentBalance, err := controller.svc.CurrentUserBalance(c.Request().Context(), userID)
	if err != nil {
		return err
	}

	minimumBalance := invoice.Amount
	if controller.svc.Config.FeeReserve {
		minimumBalance += invoice.CalcFeeLimit()
	}
	if currentBalance < minimumBalance {
		c.Logger().Errorf("User does not have enough balance invoice_id:%v user_id:%v balance:%v amount:%v", invoice.ID, userID, currentBalance, invoice.Amount)
		return c.JSON(http.StatusBadRequest, responses.NotEnoughBalanceError)
	}

	invoice.DestinationCustomRecords = map[uint64][]byte{}
	for key, value := range reqBody.CustomRecords {
		intKey, err := strconv.Atoi(key)
		if err != nil {
			return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
		}
		invoice.DestinationCustomRecords[uint64(intKey)] = []byte(value)
	}
	sendPaymentResponse, err := controller.svc.PayInvoice(c.Request().Context(), invoice)
	if err != nil {
		c.Logger().Errorf("Payment failed: user_id:%v error: %v", userID, err)
		sentry.CaptureException(err)
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":   true,
			"code":    10,
			"message": fmt.Sprintf("Payment failed. Does the receiver have enough inbound capacity? (%v)", err),
		})
	}

	responseBody := &KeySendResponseBody{}
	responseBody.RHash = &lib.JavaScriptBuffer{Data: sendPaymentResponse.PaymentHash}
	responseBody.Amount = invoice.Amount
	responseBody.Destination = invoice.DestinationPubkeyHex
	responseBody.Description = invoice.Memo
	responseBody.DescriptionHashStr = invoice.DescriptionHash
	responseBody.PaymentError = sendPaymentResponse.PaymentError
	responseBody.PaymentPreimage = &lib.JavaScriptBuffer{Data: sendPaymentResponse.PaymentPreimage}
	responseBody.PaymentRoute = sendPaymentResponse.PaymentRoute

	return c.JSON(http.StatusOK, responseBody)
}
