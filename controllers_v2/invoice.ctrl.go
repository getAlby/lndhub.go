package v2controllers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/getAlby/lndhub.go/common"
	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/getsentry/sentry-go"
	"github.com/labstack/echo/v4"
)

// InvoiceController : Add invoice controller struct
type InvoiceController struct {
	svc *service.LndhubService
}

func NewInvoiceController(svc *service.LndhubService) *InvoiceController {
	return &InvoiceController{svc: svc}
}

type Invoice struct {
	PaymentHash     string            `json:"payment_hash"`
	PaymentRequest  string            `json:"payment_request"`
	Description     string            `json:"description"`
	DescriptionHash string            `json:"description_hash,omitempty"`
	PaymentPreimage string            `json:"payment_preimage,omitempty"`
	Destination     string            `json:"destination"`
	Amount          int64             `json:"amount"`
	Fee             int64             `json:"fee"`
	Status          string            `json:"status"`
	Type            string            `json:"type"`
	ErrorMessage    string            `json:"error_message,omitempty"`
	SettledAt       time.Time         `json:"settled_at"`
	ExpiresAt       time.Time         `json:"expires_at"`
	IsPaid          bool              `json:"is_paid"`
	Keysend         bool              `json:"keysend"`
	CustomRecords   map[uint64][]byte `json:"custom_records,omitempty"`
}

// GetOutgoingInvoices godoc
// @Summary      Retrieve outgoing payments
// @Description  Returns a list of outgoing payments for a user
// @Accept       json
// @Produce      json
// @Tags         Invoice
// @Success      200  {object}  []Invoice
// @Failure      400  {object}  responses.ErrorResponse
// @Failure      500  {object}  responses.ErrorResponse
// @Router       /v2/invoices/outgoing [get]
// @Security     OAuth2Password
func (controller *InvoiceController) GetOutgoingInvoices(c echo.Context) error {
	userId := c.Get("UserID").(int64)

	invoices, err := controller.svc.InvoicesFor(c.Request().Context(), userId, common.InvoiceTypeOutgoing)
	if err != nil {
		return err
	}

	response := make([]Invoice, len(invoices))
	for i, invoice := range invoices {
		response[i] = Invoice{
			PaymentHash:     invoice.RHash,
			PaymentRequest:  invoice.PaymentRequest,
			Description:     invoice.Memo,
			DescriptionHash: invoice.DescriptionHash,
			PaymentPreimage: invoice.Preimage,
			Destination:     invoice.DestinationPubkeyHex,
			Amount:          invoice.Amount,
			Fee:             invoice.Fee,
			Status:          invoice.State,
			Type:            common.InvoiceTypePaid,
			ErrorMessage:    invoice.ErrorMessage,
			SettledAt:       invoice.SettledAt.Time,
			ExpiresAt:       invoice.ExpiresAt.Time,
			IsPaid:          invoice.State == common.InvoiceStateSettled,
			Keysend:         invoice.Keysend,
			CustomRecords:   invoice.DestinationCustomRecords,
		}
	}
	return c.JSON(http.StatusOK, &GetInvoicesResponseBody{Invoices: response})
}

// GetIncomingInvoices godoc
// @Summary      Retrieve incoming invoices
// @Description  Returns a list of incoming invoices for a user
// @Accept       json
// @Produce      json
// @Tags         Invoice
// @Success      200  {object}  []Invoice
// @Failure      400  {object}  responses.ErrorResponse
// @Failure      500  {object}  responses.ErrorResponse
// @Router       /v2/invoices/incoming [get]
// @Security     OAuth2Password
func (controller *InvoiceController) GetIncomingInvoices(c echo.Context) error {
	userId := c.Get("UserID").(int64)

	invoices, err := controller.svc.InvoicesFor(c.Request().Context(), userId, common.InvoiceTypeIncoming)
	if err != nil {
		return err
	}

	response := make([]Invoice, len(invoices))
	for i, invoice := range invoices {
		response[i] = Invoice{
			PaymentHash:     invoice.RHash,
			PaymentRequest:  invoice.PaymentRequest,
			Description:     invoice.Memo,
			DescriptionHash: invoice.DescriptionHash,
			Destination:     invoice.DestinationPubkeyHex,
			Amount:          invoice.Amount,
			Fee:             invoice.Fee,
			Status:          invoice.State,
			Type:            common.InvoiceTypeUser,
			ErrorMessage:    invoice.ErrorMessage,
			SettledAt:       invoice.SettledAt.Time,
			ExpiresAt:       invoice.ExpiresAt.Time,
			IsPaid:          invoice.State == common.InvoiceStateSettled,
			Keysend:         invoice.Keysend,
			CustomRecords:   invoice.DestinationCustomRecords,
		}
	}
	return c.JSON(http.StatusOK, &GetInvoicesResponseBody{Invoices: response})
}

type AddInvoiceRequestBody struct {
	Amount          int64  `json:"amount" validate:"gte=0"`
	Description     string `json:"description"`
	DescriptionHash string `json:"description_hash" validate:"omitempty,hexadecimal,len=64"`
}

type AddInvoiceResponseBody struct {
	PaymentHash    string    `json:"payment_hash"`
	PaymentRequest string    `json:"payment_request"`
	ExpiresAt      time.Time `json:"expires_at"`
}

type GetInvoicesResponseBody struct {
	Invoices []Invoice `json:"invoices"`
}

type InvoiceResponseBody struct {
	Payreq string   `json:"pr"`
	Routes []string `json:"routes"`
}

type PaymentMetadata struct {
	DocumentID string             `json:"document_id"`
	Authors    map[string]float64 `json:"authors"`
}

// AddInvoice godoc
// @Summary      Generate a new invoice
// @Description  Returns a new bolt11 invoice
// @Accept       json
// @Produce      json
// @Tags         Invoice
// @Param        invoice  body      AddInvoiceRequestBody  True  "Add Invoice"
// @Success      200      {object}  AddInvoiceResponseBody
// @Failure      400      {object}  responses.ErrorResponse
// @Failure      500      {object}  responses.ErrorResponse
// @Router       /v2/addinvoice [post]
// @Security     OAuth2Password
func (controller *InvoiceController) AddInvoice(c echo.Context) error {
	userID := c.Get("UserID").(int64)
	var body AddInvoiceRequestBody

	if err := c.Bind(&body); err != nil {
		c.Logger().Errorf("Failed to load addinvoice request body: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	if err := c.Validate(&body); err != nil {
		c.Logger().Errorf("Invalid addinvoice request body: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	c.Logger().Infof("Adding invoice: user_id:%v memo:%s value:%v description_hash:%s, documentID: %s, authors: %s",
		userID, body.Description, body.Amount, body.DescriptionHash)

	invoice, err := controller.svc.AddIncomingInvoice(c.Request().Context(), userID, body.Amount, body.Description, body.DescriptionHash)
	if err != nil {
		c.Logger().Errorf("Error creating invoice: user_id:%v error: %v", userID, err)
		sentry.CaptureException(err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}
	responseBody := AddInvoiceResponseBody{
		PaymentHash:    invoice.RHash,
		PaymentRequest: invoice.PaymentRequest,
		ExpiresAt:      invoice.ExpiresAt.Time,
	}

	return c.JSON(http.StatusOK, &responseBody)
}

// Invoice godoc
// @Summary      Generate an invoice without credentials
// @Description  Ask a user to generate an invoice
// @Accept       json
// @Produce      json
// @Param        user_login path string true "User login or nickname"
// @Param        amount path string true "amount in millisatoshis at the invoice"
// @Tags         Invoice
// @Success      200  {object}  InvoiceResponseBody
// @Failure      400  {object}  responses.LnurlErrorResponse
// @Failure      500  {object}  responses.LnurlErrorResponse
// @Router       /v2/invoice/{user} [get]
// @Security     OAuth2Password
func (controller *InvoiceController) Invoice(c echo.Context) error {
	// The user param could be userID (login) or a nickname (lnaddress)
	user, err := controller.svc.FindUserByLoginOrNickname(c.Request().Context(), c.Param("user"))
	if err != nil {
		c.Logger().Errorf("Failed to find user by login or nickname: user %v error %v", c.Param("user"), err)
		return c.JSON(http.StatusBadRequest, responses.LnurlpBadArgumentsError)
	}
	var amt_msat int64 = -1
	if c.QueryParams().Has("amount") {
		amt_msat, err = strconv.ParseInt(c.QueryParam("amount"), 10, 64)
		if err != nil {
			c.Logger().Errorf("Could not convert %v to uint64. %v", c.QueryParam("amount"), err)
			return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
		}
	}

	if controller.svc.Config.MaxReceiveAmount > 0 && amt_msat/1000 > controller.svc.Config.MaxReceiveAmount {
		c.Logger().Errorf("Max receive amount exceeded for user_id:%v (amount:%v)", user.ID, amt_msat/1000)
		return c.JSON(http.StatusBadRequest, responses.LnurlpBadArgumentsError)
	}
	paymentMeta := PaymentMetadata{Authors: map[string]float64{}}
	if c.QueryParams().Has("account_id") {
		for _, slice := range c.QueryParams()["account_id"] {
			authorSlice := strings.Split(slice, ",")
			if len(authorSlice) != 2 {
				c.Logger().Debugf("account_id param must be in the format<acc_id>,<floatauthorship>, got %s", slice)
				return c.JSON(http.StatusBadRequest, responses.LnurlpBadArgumentsError)
			}
			authorship, err := strconv.ParseFloat(authorSlice[1], 64)
			if err != nil {
				c.Logger().Debugf("Could not parse authorship from account_id: %v", c.QueryParams()["account_id"])
				return c.JSON(http.StatusBadRequest, responses.LnurlpBadArgumentsError)
			}
			paymentMeta.Authors[authorSlice[0]] = authorship
		}
	}
	if c.QueryParams().Has("document_id") {
		paymentMeta.DocumentID = c.QueryParam("document_id")
	}
	records := []byte{}
	if paymentMeta.DocumentID != "" {
		records, err = json.Marshal(paymentMeta)
		if err != nil {
			c.Logger().Debugf("Could not parse to json: %v", paymentMeta)
			return c.JSON(http.StatusBadRequest, responses.LnurlpBadArgumentsError)
		}
	}
	var descriptionhash_string string = ""
	var memo string = ""
	if c.QueryParams().Has("memo") {
		memo = c.QueryParam("memo")
	} else {
		descriptionHash := lnurlDescriptionHash(user.Nickname, controller.svc.Config.LnurlDomain)
		descriptionhash_hex := sha256.Sum256([]byte(descriptionHash))
		descriptionhash_string = hex.EncodeToString(descriptionhash_hex[:])
	}

	c.Logger().Infof("Adding invoice: user_id:%v value:%v description_hash:%s memo:%s", user.ID, amt_msat/1000, descriptionhash_string, memo)
	invoice, err := controller.svc.AddIncomingInvoice(c.Request().Context(), user.ID, amt_msat/1000, memo, descriptionhash_string, records...)
	if err != nil {
		c.Logger().Errorf("Error creating invoice: user_id:%v error: %v", user.ID, err)
		sentry.CaptureException(err)
		return c.JSON(http.StatusBadRequest, responses.LnurlpBadArgumentsError)
	}
	responseBody := InvoiceResponseBody{}
	responseBody.Payreq = invoice.PaymentRequest

	return c.JSON(http.StatusOK, &responseBody)

}

// GetInvoice godoc
// @Summary      Get a specific invoice
// @Description  Retrieve information about a specific invoice by payment hash
// @Accept       json
// @Produce      json
// @Tags         Invoice
// @Param        payment_hash  path      string  true  "Payment hash"
// @Success      200  {object}  Invoice
// @Failure      400  {object}  responses.ErrorResponse
// @Failure      500  {object}  responses.ErrorResponse
// @Router       /v2/invoices/{payment_hash} [get]
// @Security     OAuth2Password
func (controller *InvoiceController) GetInvoice(c echo.Context) error {
	userID := c.Get("UserID").(int64)
	rHash := c.Param("payment_hash")
	invoice, err := controller.svc.FindInvoiceByPaymentHash(c.Request().Context(), userID, rHash)
	// Probably we did not find the invoice
	if err != nil {
		c.Logger().Errorf("Invalid checkpayment request user_id:%v payment_hash:%s", userID, rHash)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}
	responseBody := Invoice{
		PaymentHash:     invoice.RHash,
		PaymentRequest:  invoice.PaymentRequest,
		Description:     invoice.Memo,
		DescriptionHash: invoice.DescriptionHash,
		PaymentPreimage: invoice.Preimage,
		Destination:     invoice.DestinationPubkeyHex,
		Amount:          invoice.Amount,
		Fee:             invoice.Fee,
		Status:          invoice.State,
		Type:            invoice.Type,
		ErrorMessage:    invoice.ErrorMessage,
		SettledAt:       invoice.SettledAt.Time,
		ExpiresAt:       invoice.ExpiresAt.Time,
		IsPaid:          invoice.State == common.InvoiceStateSettled,
		Keysend:         invoice.Keysend,
		CustomRecords:   invoice.DestinationCustomRecords,
	}
	return c.JSON(http.StatusOK, &responseBody)
}
