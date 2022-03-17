package controllers

import (
	"net/http"

	"github.com/getAlby/lndhub.go/common"
	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
)

// GetTXSController : GetTXSController struct
type GetTXSController struct {
	svc    *service.LndhubService
	plugin func([]models.Invoice, *service.LndhubService) ([]models.Invoice, error)
}

func NewGetTXSController(svc *service.LndhubService) *GetTXSController {
	result := &GetTXSController{svc: svc}
	//check for plugin
	if plug, ok := svc.MiddlewarePlugins["gettxs"]; ok {
		mwPlugin := plug.Interface().(func(in []models.Invoice, svc *service.LndhubService) ([]models.Invoice, error))
		result.plugin = mwPlugin
	}

	return result
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
			RHash:           rhash,
			PaymentHash:     rhash,
			PaymentPreimage: invoice.Preimage,
			Value:           invoice.Amount,
			Type:            common.InvoiceTypePaid,
			Fee:             0, //TODO charge fees
			Timestamp:       invoice.CreatedAt.Unix(),
			Memo:            invoice.Memo,
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
