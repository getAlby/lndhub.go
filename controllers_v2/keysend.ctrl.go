package v2controllers

import (
	"net/http"
	"strconv"

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

type MultiKeySendRequestBody struct {
	Keysends []KeySendRequestBody `json:"keysends"`
}
type MultiKeySendResponseBody struct {
	Keysends []KeySendResult `json:"keysends"`
}

type KeySendResult struct {
	Keysend KeySendResponseBody     `json:",omitempty"`
	Error   responses.ErrorResponse `json:",omitempty"`
}

type KeySendResponseBody struct {
	Amount          int64  `json:"amount"`
	Fee             int64  `json:"fee"`
	Description     string `json:"description,omitempty"`
	DescriptionHash string `json:"description_hash,omitempty"`
	Destination     string `json:"destination,omitempty"`
	PaymentPreimage string `json:"payment_preimage,omitempty"`
	PaymentHash     string `json:"payment_hash,omitempty"`
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
	_, err := controller.SingleKeySend(c, &reqBody, userID)
	return err
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
	// TODO
	// - V create request and response structs
	// - V extract shared code
	// - V call shared code in loop
	// - V fill and return response
	// - test
	// - integration tests
	// - PR
	// - deploy
	// - Update API
	// - mail sam
	userID := c.Get("UserID").(int64)
	reqBody := MultiKeySendRequestBody{}
	if err := c.Bind(&reqBody); err != nil {
		c.Logger().Errorf("Failed to load keysend request body: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}
	result := &MultiKeySendResponseBody{
		Keysends: []KeySendResult{},
	}
	for _, keysend := range reqBody.Keysends {
		keysend := keysend
		res, err := controller.SingleKeySend(c, &keysend, userID)
		if err != nil {
			controller.svc.Logger.Errorf("Error making keysend split payment %v %s", keysend, err.Error())
			result.Keysends = append(result.Keysends, KeySendResult{
				Keysend: KeySendResponseBody{
					Destination: keysend.Destination,
				},
				Error: responses.ErrorResponse{Error: true, Code: 500, Message: err.Error()},
			})
		}
		result.Keysends = append(result.Keysends, KeySendResult{
			Keysend: *res,
			Error: responses.ErrorResponse{
				Error: false,
				Code:  200,
			},
		})
	}
	return c.JSON(http.StatusOK, result)
}

func (controller *KeySendController) SingleKeySend(c echo.Context, reqBody *KeySendRequestBody, userID int64) (result *KeySendResponseBody, err error) {

	if err := c.Validate(&reqBody); err != nil {
		c.Logger().Errorf("Invalid keysend request body: %v", err)
		return nil, c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
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
		return nil, err
	}

	currentBalance, err := controller.svc.CurrentUserBalance(c.Request().Context(), userID)
	if err != nil {
		return nil, err
	}

	minimumBalance := invoice.Amount
	if controller.svc.Config.FeeReserve {
		minimumBalance += invoice.CalcFeeLimit(controller.svc.IdentityPubkey)
	}
	if currentBalance < minimumBalance {
		c.Logger().Errorf("User does not have enough balance invoice_id:%v user_id:%v balance:%v amount:%v", invoice.ID, userID, currentBalance, invoice.Amount)
		return nil, c.JSON(http.StatusBadRequest, responses.NotEnoughBalanceError)
	}

	invoice.DestinationCustomRecords = map[uint64][]byte{}
	for key, value := range reqBody.CustomRecords {
		intKey, err := strconv.Atoi(key)
		if err != nil {
			return nil, c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
		}
		invoice.DestinationCustomRecords[uint64(intKey)] = []byte(value)
	}
	sendPaymentResponse, err := controller.svc.PayInvoice(c.Request().Context(), invoice)
	if err != nil {
		c.Logger().Errorf("Payment failed: user_id:%v error: %v", userID, err)
		sentry.CaptureException(err)
		return nil, c.JSON(http.StatusBadRequest, echo.Map{
			"error":   true,
			"code":    10,
			"message": err.Error(),
		})
	}

	responseBody := &KeySendResponseBody{
		Amount:          sendPaymentResponse.PaymentRoute.TotalAmt,
		Fee:             sendPaymentResponse.PaymentRoute.TotalFees,
		Description:     reqBody.Memo,
		Destination:     reqBody.Destination,
		PaymentPreimage: sendPaymentResponse.PaymentPreimageStr,
		PaymentHash:     sendPaymentResponse.PaymentHashStr,
	}

	return responseBody, c.JSON(http.StatusOK, responseBody)
}
