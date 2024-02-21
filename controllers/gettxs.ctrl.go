package controllers

import (
	"net/http"

	"github.com/getAlby/lndhub.go/common"
	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

// GetTXSController : GetTXSController struct
type GetTXSController struct {
	svc *service.LndhubService
}

func NewGetTXSController(svc *service.LndhubService) *GetTXSController {
	return &GetTXSController{svc: svc}
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

func (controller *GetTXSController) GetTXS(c echo.Context) error {
	userId := c.Get("UserID").(int64)

	invoices, err := controller.svc.InvoicesFor(c.Request().Context(), userId, common.InvoiceTypeOutgoing)
	if err != nil {
		c.Logger().Errorj(
			log.JSON{
				"message":        "failed to get transactions",
				"error":          err,
				"lndhub_user_id": userId,
			},
		)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	response := make([]OutgoingInvoice, len(invoices))
	for i, invoice := range invoices {
		//only return settled invoices
		if invoice.State != common.InvoiceStateSettled {
			continue
		}
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

func (controller *GetTXSController) GetUserInvoices(c echo.Context) error {
	userId := c.Get("UserID").(int64)

	invoices, err := controller.svc.InvoicesFor(c.Request().Context(), userId, common.InvoiceTypeIncoming)
	if err != nil {
		c.Logger().Errorj(
			log.JSON{
				"message":        "failed to get invoices",
				"error":          err,
				"lndhub_user_id": userId,
			},
		)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
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
