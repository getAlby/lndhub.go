package controllers

import (
	"net/http"

	"github.com/getAlby/lndhub.go/common"
	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
)

// GetTXSController : GetTXSController struct
type GetTXSController struct {
	svc *service.LndhubService
}

func NewGetTXSController(svc *service.LndhubService) *GetTXSController {
	return &GetTXSController{svc: svc}
}

type OutgoingInvoice struct {
	RHash           interface{} `json:"r_hash,omitempty"`
	PaymentHash     interface{} `json:"payment_hash"`
	PaymentPreimage string      `json:"payment_preimage"`
	Value           int64       `json:"value"`
	Type            string      `json:"type"`
	Fee             int64       `json:"fee"`
	Timestamp       int64       `json:"timestamp"`
	Memo            string      `json:"memo"`
}

type IncomingInvoice struct {
	RHash          interface{} `json:"r_hash,omitempty"`
	PaymentHash    interface{} `json:"payment_hash"`
	PaymentRequest string      `json:"payment_request"`
	Description    string      `json:"description"`
	PayReq         string      `json:"pay_req"`
	Timestamp      int64       `json:"timestamp"`
	Type           string      `json:"type"`
	ExpireTime     int64       `json:"expire_time"`
	Amount         int64       `json:"amt"`
	IsPaid         bool        `json:"ispaid"`
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
func (controller *GetTXSController) GetTXS(c echo.Context) error {
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
func (controller *GetTXSController) GetUserInvoices(c echo.Context) error {
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
		}
	}
	return c.JSON(http.StatusOK, &response)
}
