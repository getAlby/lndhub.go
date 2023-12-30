package controllers

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"

	"github.com/getAlby/lndhub.go/common"
	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/getAlby/lndhub.go/lnd"
	"github.com/getsentry/sentry-go"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
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

	if controller.svc.LndClient.IsIdentityPubkey(reqBody.Destination) && reqBody.CustomRecords[strconv.Itoa(service.TLV_WALLET_ID)] == "" {
		return c.JSON(http.StatusBadRequest, &responses.ErrorResponse{
			Error:          true,
			Code:           8,
			Message:        fmt.Sprintf("Internal keysend payments require the custom record %d to be present.", service.TLV_WALLET_ID),
			HttpStatusCode: 400,
		})
	}

	resp, err := controller.svc.CheckOutgoingPaymentAllowed(c, lnPayReq, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, responses.GeneralServerError)
	}
	if resp != nil {
		c.Logger().Errorf("Error: %v user_id:%v amount:%v", resp.Message, userID, lnPayReq.PayReq.NumSatoshis)
		return c.JSON(resp.HttpStatusCode, resp)
	}
	invoice, errResp := controller.svc.AddOutgoingInvoice(c.Request().Context(), userID, "", lnPayReq)
	if errResp != nil {
		return c.JSON(errResp.HttpStatusCode, errResp)
	}
	if _, err := hex.DecodeString(invoice.DestinationPubkeyHex); err != nil || len(invoice.DestinationPubkeyHex) != common.DestinationPubkeyHexSize {
		c.Logger().Errorf("Invalid destination pubkey hex user_id:%v pubkey:%v", userID, len(invoice.DestinationPubkeyHex))
		return c.JSON(http.StatusBadRequest, responses.InvalidDestinationError)
	}
	invoice.DestinationCustomRecords = map[uint64][]byte{}
	for key, value := range reqBody.CustomRecords {
		intKey, err := strconv.Atoi(key)
		if err != nil {
			c.Logger().Errorj(
				log.JSON{
					"message":        "invalid custom records",
					"error":          err,
					"lndhub_user_id": userID,
				},
			)
			return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
		}
		invoice.DestinationCustomRecords[uint64(intKey)] = []byte(value)
	}
	sendPaymentResponse, err := controller.svc.PayInvoice(c.Request().Context(), invoice)
	if err != nil {
		c.Logger().Errorj(
			log.JSON{
				"message": 	"payment failed",
				"lndhub_user_id": userID,
				"error": err,
			},
		)
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
