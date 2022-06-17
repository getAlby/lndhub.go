package v2controllers

import (
	"net/http"

	"github.com/getAlby/lndhub.go/common"
	"github.com/getAlby/lndhub.go/lib"
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

type OutgoingInvoice struct {
	RHash           interface{}       `json:"r_hash,omitempty"`
	PaymentHash     interface{}       `json:"payment_hash"`
	PaymentPreimage string            `json:"payment_preimage"`
	Value           int64             `json:"value"`
	Type            string            `json:"type"`
	Fee             int64             `json:"fee"`
	Timestamp       int64             `json:"timestamp"`
	Memo            string            `json:"memo"`
	Keysend         bool              `json:"keysend"`
	CustomRecords   map[uint64][]byte `json:"custom_records"`
}

type IncomingInvoice struct {
	RHash          interface{}       `json:"r_hash,omitempty"`
	PaymentHash    interface{}       `json:"payment_hash"`
	PaymentRequest string            `json:"payment_request"`
	Description    string            `json:"description"`
	PayReq         string            `json:"pay_req"`
	Timestamp      int64             `json:"timestamp"`
	Type           string            `json:"type"`
	ExpireTime     int64             `json:"expire_time"`
	Amount         int64             `json:"amt"`
	IsPaid         bool              `json:"ispaid"`
	Keysend        bool              `json:"keysend"`
	CustomRecords  map[uint64][]byte `json:"custom_records"`
}

// GetTXS godoc
// @Summary      Retrieve outgoing payments
// @Description  Returns a list of outgoing payments for a user
// @Accept       json
// @Produce      json
// @Tags         Account
// @Success      200  {object}  []OutgoingInvoice
// @Failure      400  {object}  responses.ErrorResponse
// @Failure      500  {object}  responses.ErrorResponse
// @Router       /gettxs [get]
// @Security     OAuth2Password
func (controller *InvoiceController) GetOutgoingInvoices(c echo.Context) error {
	userId := c.Get("UserID").(int64)

	invoices, err := controller.svc.InvoicesFor(c.Request().Context(), userId, common.InvoiceTypeOutgoing)
	if err != nil {
		return err
	}

	response := make([]OutgoingInvoice, len(invoices))
	for i, invoice := range invoices {
		rhash, _ := lib.ToJavaScriptBuffer(invoice.RHash)
		response[i] = OutgoingInvoice{
			RHash:           rhash,
			PaymentHash:     rhash,
			PaymentPreimage: invoice.Preimage,
			Value:           invoice.Amount,
			Type:            common.InvoiceTypePaid,
			Fee:             invoice.Fee,
			Timestamp:       invoice.CreatedAt.Unix(),
			Memo:            invoice.Memo,
			Keysend:         invoice.Keysend,
			CustomRecords:   invoice.DestinationCustomRecords,
		}
	}
	return c.JSON(http.StatusOK, &response)
}

// GetUserInvoices godoc
// @Summary      Retrieve incoming invoices
// @Description  Returns a list of incoming invoices for a user
// @Accept       json
// @Produce      json
// @Tags         Account
// @Success      200  {object}  []IncomingInvoice
// @Failure      400  {object}  responses.ErrorResponse
// @Failure      500  {object}  responses.ErrorResponse
// @Router       /getuserinvoices [get]
// @Security     OAuth2Password
func (controller *InvoiceController) GetIncomingInvoices(c echo.Context) error {
	userId := c.Get("UserID").(int64)

	invoices, err := controller.svc.InvoicesFor(c.Request().Context(), userId, common.InvoiceTypeIncoming)
	if err != nil {
		return err
	}

	response := make([]IncomingInvoice, len(invoices))
	for i, invoice := range invoices {
		rhash, _ := lib.ToJavaScriptBuffer(invoice.RHash)
		response[i] = IncomingInvoice{
			RHash:          rhash,
			PaymentHash:    invoice.RHash,
			PaymentRequest: invoice.PaymentRequest,
			Description:    invoice.Memo,
			PayReq:         invoice.PaymentRequest,
			Timestamp:      invoice.CreatedAt.Unix(),
			Type:           common.InvoiceTypeUser,
			ExpireTime:     3600 * 24,
			Amount:         invoice.Amount,
			IsPaid:         invoice.State == common.InvoiceStateSettled,
			Keysend:        invoice.Keysend,
			CustomRecords:  invoice.DestinationCustomRecords,
		}
	}
	return c.JSON(http.StatusOK, &response)
}

type AddInvoiceRequestBody struct {
	Amount          interface{} `json:"amt"` // amount in Satoshi
	Memo            string      `json:"memo"`
	DescriptionHash string      `json:"description_hash" validate:"omitempty,hexadecimal,len=64"`
}

type AddInvoiceResponseBody struct {
	RHash          string `json:"r_hash"`
	PaymentRequest string `json:"payment_request"`
	PayReq         string `json:"pay_req"`
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
// @Router       /addinvoice [post]
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

	amount, err := controller.svc.ParseInt(body.Amount)
	if err != nil || amount < 0 {
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}
	c.Logger().Infof("Adding invoice: user_id:%v memo:%s value:%v description_hash:%s", userID, body.Memo, amount, body.DescriptionHash)

	invoice, err := controller.svc.AddIncomingInvoice(c.Request().Context(), userID, amount, body.Memo, body.DescriptionHash)
	if err != nil {
		c.Logger().Errorf("Error creating invoice: user_id:%v error: %v", userID, err)
		sentry.CaptureException(err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}
	responseBody := AddInvoiceResponseBody{}
	responseBody.RHash = invoice.RHash
	responseBody.PaymentRequest = invoice.PaymentRequest
	responseBody.PayReq = invoice.PaymentRequest

	return c.JSON(http.StatusOK, &responseBody)
}
func (controller *InvoiceController) GetInvoice(c echo.Context) error {
	return nil
}
