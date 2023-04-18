package v2controllers

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"

	"github.com/getAlby/lndhub.go/common"
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
	Amount                  int64             `json:"amount" validate:"required,gt=0"`
	Destination             string            `json:"destination" validate:"required"`
	Memo                    string            `json:"memo" validate:"omitempty"`
	DeprecatedCustomRecords map[string]string `json:"customRecords" validate:"omitempty"`
	CustomRecords           map[string]string `json:"custom_records" validate:"omitempty"`
}

type MultiKeySendRequestBody struct {
	Keysends []KeySendRequestBody `json:"keysends"`
}
type MultiKeySendResponseBody struct {
	Keysends []KeySendResult `json:"keysends"`
}

type KeySendResult struct {
	Keysend *KeySendResponseBody     `json:"keysend,omitempty"`
	Error   *responses.ErrorResponse `json:"error,omitempty"`
}

type KeySendResponseBody struct {
	Amount          int64             `json:"amount"`
	Fee             int64             `json:"fee"`
	Description     string            `json:"description,omitempty"`
	DescriptionHash string            `json:"description_hash,omitempty"`
	Destination     string            `json:"destination,omitempty"`
	CustomRecords   map[string]string `json:"custom_records" validate:"omitempty"`
	PaymentPreimage string            `json:"payment_preimage,omitempty"`
	PaymentHash     string            `json:"payment_hash,omitempty"`
}

// // KeySend godoc
// @Summary      Make a keysend payment
// @Description  Pay a node without an invoice using it's public key
// @Accept       json
// @Produce      json
// @Tags         Payment
// @Param        KeySendRequestBody  body      KeySendRequestBody  True  "Invoice to pay"
// @Success      200                 {object}  KeySendResponseBody
// @Failure      400                 {object}  responses.ErrorResponse
// @Failure      500                 {object}  responses.ErrorResponse
// @Router       /v2/payments/keysend [post]
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

	result, errResp := controller.SingleKeySend(c, &reqBody, userID)
	if errResp != nil {
		c.Logger().Errorf("Failed to send keysend: %s", errResp.Message)
		return c.JSON(errResp.HttpStatusCode, errResp)
	}
	return c.JSON(http.StatusOK, result)
}

// // MultiKeySend godoc
// @Summary      Make multiple keysend payments
// @Description  Pay multiple nodes without an invoice using their public key
// @Accept       json
// @Produce      json
// @Tags         Payment
// @Param        MultiKeySendRequestBody  body      MultiKeySendRequestBody  True  "Invoice to pay"
// @Success      200                 {object}  MultiKeySendResponseBody
// @Failure      400                 {object}  responses.ErrorResponse
// @Failure      500                 {object}  responses.ErrorResponse
// @Router       /v2/payments/keysend/multi [post]
// @Security     OAuth2Password
func (controller *KeySendController) MultiKeySend(c echo.Context) error {
	userID := c.Get("UserID").(int64)
	reqBody := MultiKeySendRequestBody{}
	if err := c.Bind(&reqBody); err != nil {
		c.Logger().Errorf("Failed to load keysend request body: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}
	if err := c.Validate(&reqBody); err != nil {
		c.Logger().Errorf("Invalid keysend request body: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}
	for _, split := range reqBody.Keysends {
		if err := c.Validate(&split); err != nil {
			c.Logger().Errorf("Invalid keysend request body: %v", err)
			return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
		}
	}
	result := &MultiKeySendResponseBody{
		Keysends: []KeySendResult{},
	}
	singleSuccesfulPayment := false
	for _, keysend := range reqBody.Keysends {
		keysend := keysend
		res, err := controller.SingleKeySend(c, &keysend, userID)
		if err != nil {
			controller.svc.Logger.Errorf("Error making keysend split payment %v %s", keysend, err.Message)
			result.Keysends = append(result.Keysends, KeySendResult{
				Keysend: &KeySendResponseBody{
					Destination:   keysend.Destination,
					CustomRecords: keysend.CustomRecords,
				},
				Error: err,
			})
			continue
		}
		result.Keysends = append(result.Keysends, KeySendResult{
			Keysend: res,
		})
		singleSuccesfulPayment = true
	}
	status := http.StatusOK
	if !singleSuccesfulPayment {
		status = http.StatusInternalServerError
	}
	return c.JSON(status, result)
}

func (controller *KeySendController) SingleKeySend(c echo.Context, reqBody *KeySendRequestBody, userID int64) (result *KeySendResponseBody, resp *responses.ErrorResponse) {
	lnPayReq := &lnd.LNPayReq{
		PayReq: &lnrpc.PayReq{
			Destination: reqBody.Destination,
			NumSatoshis: reqBody.Amount,
			Description: reqBody.Memo,
		},
		Keysend: true,
	}
	//temporary workaround due to an inconsistency in json snake case vs camel case
	//DeprecatedCustomRecords to be removed later
	customRecords := reqBody.DeprecatedCustomRecords
	if reqBody.CustomRecords != nil {
		customRecords = reqBody.CustomRecords
	}
	if reqBody.Destination == controller.svc.IdentityPubkey && customRecords[strconv.Itoa(service.TLV_WALLET_ID)] == "" {
		return nil, &responses.ErrorResponse{
			Error:          true,
			Code:           8,
			Message:        fmt.Sprintf("Internal keysend payments require the custom record %d to be present.", service.TLV_WALLET_ID),
			HttpStatusCode: 400,
		}
	}
	ok, err := controller.svc.BalanceCheck(c.Request().Context(), lnPayReq, userID)
	if err != nil {
		controller.svc.Logger.Error(err)
		return nil, &responses.GeneralServerError
	}
	if !ok {
		c.Logger().Errorf("User does not have enough balance user_id:%v amount:%v", userID, lnPayReq.PayReq.NumSatoshis)
		return nil, &responses.NotEnoughBalanceError
	}
	invoice, err := controller.svc.AddOutgoingInvoice(c.Request().Context(), userID, "", lnPayReq)
	if err != nil {
		controller.svc.Logger.Error(err)
		return nil, &responses.GeneralServerError
	}
	if _, err := hex.DecodeString(invoice.DestinationPubkeyHex); err != nil || len(invoice.DestinationPubkeyHex) != common.DestinationPubkeyHexSize {
		c.Logger().Errorf("Invalid destination pubkey hex user_id:%v pubkey:%v", userID, len(invoice.DestinationPubkeyHex))
		return nil, &responses.InvalidDestinationError
	}
	invoice.DestinationCustomRecords = map[uint64][]byte{}

	for key, value := range customRecords {
		intKey, err := strconv.Atoi(key)
		if err != nil {
			return nil, &responses.BadArgumentsError
		}
		invoice.DestinationCustomRecords[uint64(intKey)] = []byte(value)
	}
	sendPaymentResponse, err := controller.svc.PayInvoice(c.Request().Context(), invoice)
	if err != nil {
		c.Logger().Errorf("Payment failed: user_id:%v error: %v", userID, err)
		sentry.CaptureException(err)
		return nil, &responses.ErrorResponse{
			Error:   true,
			Code:    10,
			Message: err.Error(),
		}
	}

	responseBody := &KeySendResponseBody{
		Amount:          sendPaymentResponse.PaymentRoute.TotalAmt,
		Fee:             sendPaymentResponse.PaymentRoute.TotalFees,
		CustomRecords:   customRecords,
		Description:     reqBody.Memo,
		Destination:     reqBody.Destination,
		PaymentPreimage: sendPaymentResponse.PaymentPreimageStr,
		PaymentHash:     sendPaymentResponse.PaymentHashStr,
	}

	return responseBody, nil
}
