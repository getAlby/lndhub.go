package controllers

import (
	"net/http"
	"time"

	"github.com/getAlby/lndhub.go/common"
	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/getAlby/lndhub.go/lib/tokens"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

// GetTXSController : GetTXSController struct
type InvoiceStreamController struct {
	svc *service.LndhubService
}

type InvoiceEventWrapper struct {
	Type    string           `json:"type"`
	Invoice *IncomingInvoice `json:"invoice,omitempty"`
}

func NewInvoiceStreamController(svc *service.LndhubService) *InvoiceStreamController {
	return &InvoiceStreamController{svc: svc}
}

// Stream invoices streams incoming payments to the client
func (controller *InvoiceStreamController) StreamInvoices(c echo.Context) error {
	userId, err := tokens.ParseToken(controller.svc.Config.JWTSecret, (c.QueryParam("token")))
	if err != nil {
		return err
	}
	invoiceChan := make(chan models.Invoice)
	ticker := time.NewTicker(30 * time.Second)
	ws, done, err := createWebsocketUpgrader(c)
	defer ws.Close()
	if err != nil {
		return err
	}
	//start subscription
	subId := controller.svc.InvoicePubSub.Subscribe(userId, invoiceChan)

	//start with keepalive message
	err = ws.WriteJSON(&InvoiceEventWrapper{Type: "keepalive"})
	if err != nil {
		controller.svc.Logger.Error(err)
		controller.svc.InvoicePubSub.Unsubscribe(subId, userId)
		return err
	}
SocketLoop:
	for {
		select {
		case <-done:
			break SocketLoop
		case <-ticker.C:
			err := ws.WriteJSON(&InvoiceEventWrapper{Type: "keepalive"})
			if err != nil {
				controller.svc.Logger.Error(err)
				break SocketLoop
			}
		case invoice := <-invoiceChan:
			err := ws.WriteJSON(
				&InvoiceEventWrapper{
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
					}})
			if err != nil {
				controller.svc.Logger.Error(err)
				break SocketLoop
			}
		}
	}
	controller.svc.InvoicePubSub.Unsubscribe(subId, userId)
	return nil
}

//open the websocket and start listening for close messages in a goroutine
func createWebsocketUpgrader(c echo.Context) (conn *websocket.Conn, done chan struct{}, err error) {
	upgrader := websocket.Upgrader{}
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return nil, nil, err
	}

	//start listening for close messages
	done = make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, _, err := ws.ReadMessage()
			if err != nil {
				return
			}
		}
	}()
	return ws, done, nil
}
