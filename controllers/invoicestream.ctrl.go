package controllers

import (
	"net/http"
	"strconv"
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

// StreamInvoices godoc
// @Summary      Websocket for incoming payments
// @Description  Websocket: won't work with Swagger web UI. Returns a stream of settled incoming payments.
// @Description  A keep-alive message is sent on startup and every 30s.
// @Accept       json
// @Produce      json
// @Tags         Invoice
// @Param        token               query     string  true   "Auth token, retrieved from /auth endpoint"
// @Param        since_payment_hash  query     string  false  "Payment hash of earliest invoice. If specified, missing updates starting from this payment will be sent."
// @Success      200                 {object}  []InvoiceEventWrapper
// @Failure      400                 {object}  responses.ErrorResponse
// @Failure      500                 {object}  responses.ErrorResponse
// @Router       /invoices/stream [get]
// @Security     OAuth2Password
func (controller *InvoiceStreamController) StreamInvoices(c echo.Context) error {
	userId, err := tokens.ParseToken(controller.svc.Config.JWTSecret, (c.QueryParam("token")), false)
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
	subId, err := controller.svc.InvoicePubSub.Subscribe(strconv.FormatInt(userId, 10), invoiceChan)
	if err != nil {
		controller.svc.Logger.Error(err)
		return err
	}
	//start with keepalive message
	err = ws.WriteJSON(&InvoiceEventWrapper{Type: "keepalive"})
	if err != nil {
		controller.svc.Logger.Error(err)
		controller.svc.InvoicePubSub.Unsubscribe(subId, strconv.FormatInt(userId, 10))
		return err
	}
	fromPaymentHash := c.QueryParam("since_payment_hash")
	if fromPaymentHash != "" {
		err = controller.writeMissingInvoices(c, userId, ws, fromPaymentHash)
		if err != nil {
			controller.svc.Logger.Error(err)
			controller.svc.InvoicePubSub.Unsubscribe(subId, strconv.FormatInt(userId, 10))
			return err
		}
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
	controller.svc.InvoicePubSub.Unsubscribe(subId, strconv.FormatInt(userId, 10))
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

func (controller *InvoiceStreamController) writeMissingInvoices(c echo.Context, userId int64, ws *websocket.Conn, hash string) error {
	invoices, err := controller.svc.InvoicesFor(c.Request().Context(), userId, common.InvoiceTypeIncoming)
	if err != nil {
		return err
	}
	for _, inv := range invoices {
		//invoices are order from newest to oldest (with a maximum of 100 invoices being returned)
		//so if we get a match on the hash, we have processed all missing invoices for this client
		if inv.RHash == hash {
			break
		}
		if inv.State == common.InvoiceStateSettled {
			err := ws.WriteJSON(
				&InvoiceEventWrapper{
					Type: "invoice",
					Invoice: &IncomingInvoice{
						PaymentHash:    inv.RHash,
						PaymentRequest: inv.PaymentRequest,
						Description:    inv.Memo,
						PayReq:         inv.PaymentRequest,
						Timestamp:      inv.CreatedAt.Unix(),
						Type:           common.InvoiceTypeUser,
						Amount:         inv.Amount,
						IsPaid:         inv.State == common.InvoiceStateSettled,
					}})
			if err != nil {
				return err
			}
		}
	}
	return nil
}
