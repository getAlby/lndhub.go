package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/getAlby/lndhub.go/common"
	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
)

// GetTXSController : GetTXSController struct
type InvoiceStreamController struct {
	svc *service.LndhubService
}

func NewInvoiceStreamController(svc *service.LndhubService) *InvoiceStreamController {
	return &InvoiceStreamController{svc: svc}
}

// Stream invoices streams incoming payments to the client
func (controller *InvoiceStreamController) StreamInvoices(c echo.Context) error {
	userId := c.Get("UserID").(int64)
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	c.Response().WriteHeader(http.StatusOK)
	enc := json.NewEncoder(c.Response())
	invoiceChan := make(chan models.Invoice)
	controller.svc.InvoiceSubscribers[userId] = invoiceChan
	for invoice := range invoiceChan {
		if err := enc.Encode(IncomingInvoice{
			PaymentHash:    invoice.RHash,
			PaymentRequest: invoice.PaymentRequest,
			Description:    invoice.Memo,
			PayReq:         invoice.PaymentRequest,
			Timestamp:      invoice.CreatedAt.Unix(),
			Type:           common.InvoiceTypeUser,
			ExpireTime:     3600 * 24,
			Amount:         invoice.Amount,
			IsPaid:         invoice.State == common.InvoiceStateSettled,
		}); err != nil {
			return err
		}
		c.Response().Flush()
	}
	return nil
}
