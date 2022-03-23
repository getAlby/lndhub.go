package controllers

import (
	"encoding/json"
	"net/http"
	"time"

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

type InvoiceEvent struct {
	Invoice *IncomingInvoice `json:"invoice,omitempty"`
	Type    string
}

// Stream invoices streams incoming payments to the client
func (controller *InvoiceStreamController) StreamInvoices(c echo.Context) error {
	userId := c.Get("UserID").(int64)
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	c.Response().WriteHeader(http.StatusOK)
	enc := json.NewEncoder(c.Response())
	invoiceChan := make(chan models.Invoice)
	controller.svc.InvoiceSubscribers[userId] = invoiceChan
	ctx := c.Request().Context()
	ticker := time.NewTicker(30 * time.Second)
	for {
		select {
		case <-ticker.C:
			if err := enc.Encode(
				InvoiceEvent{
					Type: "keepalive",
				}); err != nil {
				return err
			}
		case <-ctx.Done():
			return nil
		case invoice := <-invoiceChan:
			if err := enc.Encode(
				InvoiceEvent{
					Type: "invoice",
					Invoice: &IncomingInvoice{
						PaymentHash:    invoice.RHash,
						PaymentRequest: invoice.PaymentRequest,
						Description:    invoice.Memo,
						PayReq:         invoice.PaymentRequest,
						Timestamp:      invoice.CreatedAt.Unix(),
						Type:           common.InvoiceTypeUser,
						Amount:         invoice.Amount,
						IsPaid:         invoice.State == common.InvoiceStateSettled,
					}}); err != nil {
				return err
			}
		}
		c.Response().Flush()
	}
}
