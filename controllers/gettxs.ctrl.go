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
	RHash           interface{} `json:"r_hash"`
	PaymentHash     interface{} `json:"payment_hash"`
	PaymentPreimage string      `json:"payment_preimage"`
	Value           int64       `json:"value"`
	Type            string      `json:"type"`
	Fee             int64       `json:"fee"`
	Timestamp       int64       `json:"timestamp"`
	Memo            string      `json:"memo"`
}

type IncomingInvoice struct {
	RHash          interface{} `json:"r_hash"`
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

// GetTXS : Get TXS Controller
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
			rhash,
			rhash,
			invoice.Preimage,
			invoice.Amount,
			common.InvoiceTypePaid,
			0, //TODO charge fees
			invoice.CreatedAt.Unix(),
			invoice.Memo,
		}
	}
	return c.JSON(http.StatusOK, &response)
}

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
			rhash,
			invoice.RHash,
			invoice.PaymentRequest,
			invoice.Memo,
			invoice.PaymentRequest,
			invoice.CreatedAt.Unix(),
			common.InvoiceTypeUser,
			3600 * 24,
			invoice.Amount,
			invoice.State == common.InvoiceStateSettled,
		}
	}
	return c.JSON(http.StatusOK, &response)
}
