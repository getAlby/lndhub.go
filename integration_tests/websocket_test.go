package integration_tests

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/getAlby/lndhub.go/controllers"
	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/getAlby/lndhub.go/lib/tokens"
	"github.com/getAlby/lndhub.go/lnd"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type KeepAlive struct {
	Type string
}

type WebSocketTestSuite struct {
	TestSuite
	fundingClient            *lnd.LNDWrapper
	service                  *service.LndhubService
	userLogin                ExpectedCreateUserResponseBody
	userToken                string
	userToken2               string
	invoiceUpdateSubCancelFn context.CancelFunc
	websocketServer          *httptest.Server
	wsUrl                    string
	wsUrl2                   string
}
type WsHandler struct {
	handler echo.HandlerFunc
}

func (h *WsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	e := echo.New()
	c := e.NewContext(r, w)

	err := h.handler(c)
	if err != nil {
		_, _ = w.Write([]byte(err.Error()))
	}
}

func (suite *WebSocketTestSuite) SetupSuite() {
	lndClient, err := lnd.NewLNDclient(lnd.LNDoptions{
		Address:     lnd2RegtestAddress,
		MacaroonHex: lnd2RegtestMacaroonHex,
	})
	if err != nil {
		log.Fatalf("Error setting up funding client: %v", err)
	}
	suite.fundingClient = lndClient

	svc, err := LndHubTestServiceInit(nil)
	if err != nil {
		log.Fatalf("Error initializing test service: %v", err)
	}
	users, userTokens, err := createUsers(svc, 2)
	if err != nil {
		log.Fatalf("Error creating test users: %v", err)
	}
	// Subscribe to LND invoice updates in the background
	// store cancel func to be called in tear down suite
	ctx, cancel := context.WithCancel(context.Background())
	suite.invoiceUpdateSubCancelFn = cancel
	go svc.InvoiceUpdateSubscription(ctx)
	suite.service = svc
	e := echo.New()

	e.HTTPErrorHandler = responses.HTTPErrorHandler
	e.Validator = &lib.CustomValidator{Validator: validator.New()}
	suite.echo = e
	assert.Equal(suite.T(), 2, len(users))
	assert.Equal(suite.T(), 2, len(userTokens))
	suite.userLogin = users[0]
	suite.userToken = userTokens[0]
	suite.userToken2 = userTokens[1]
	suite.echo.Use(tokens.Middleware([]byte(suite.service.Config.JWTSecret)))
	suite.echo.POST("/addinvoice", controllers.NewAddInvoiceController(suite.service).AddInvoice)

	//websocket server
	h := WsHandler{handler: controllers.NewInvoiceStreamController(suite.service).StreamInvoices}
	server := httptest.NewServer(http.HandlerFunc(h.ServeHTTP))
	suite.websocketServer = server
	suite.wsUrl = "ws" + strings.TrimPrefix(suite.websocketServer.URL, "http") + fmt.Sprintf("?token=%s", suite.userToken)
	suite.wsUrl2 = "ws" + strings.TrimPrefix(suite.websocketServer.URL, "http") + fmt.Sprintf("?token=%s", suite.userToken2)
}

func (suite *WebSocketTestSuite) TestWebSocket() {
	//start listening to websocket
	ws, _, err := websocket.DefaultDialer.Dial(suite.wsUrl, nil)
	assert.NoError(suite.T(), err)
	_, msg, err := ws.ReadMessage()
	assert.NoError(suite.T(), err)
	keepAlive := KeepAlive{}
	err = json.Unmarshal([]byte(msg), &keepAlive)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "keepalive", keepAlive.Type)

	// create incoming invoice and fund account
	invoice := suite.createAddInvoiceReq(1000, "integration test websocket 1", suite.userToken)
	sendPaymentRequest := lnrpc.SendRequest{
		PaymentRequest: invoice.PayReq,
		FeeLimit:       nil,
	}
	_, err = suite.fundingClient.SendPaymentSync(context.Background(), &sendPaymentRequest)
	assert.NoError(suite.T(), err)

	_, msg, err = ws.ReadMessage()
	assert.NoError(suite.T(), err)
	event := ExpectedInvoiceEventWrapper{}
	err = json.Unmarshal([]byte(msg), &event)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), event.Type, "invoice")
	assert.Equal(suite.T(), int64(1000), event.Invoice.Amount)
	assert.Equal(suite.T(), "integration test websocket 1", event.Invoice.Description)
}

func (suite *WebSocketTestSuite) TestWebSocketDoubeSubscription() {
	//create 1st subscription
	ws1, _, err := websocket.DefaultDialer.Dial(suite.wsUrl, nil)
	assert.NoError(suite.T(), err)
	//read keepalive msg
	_, _, err = ws1.ReadMessage()
	//create 2nd subscription, create invoice, pay invoice, assert that invoice is received twice
	//start listening to websocket
	ws2, _, err := websocket.DefaultDialer.Dial(suite.wsUrl, nil)
	assert.NoError(suite.T(), err)
	//read keepalive msg
	_, _, err = ws2.ReadMessage()
	assert.NoError(suite.T(), err)
	invoice := suite.createAddInvoiceReq(1000, "integration test websocket 2", suite.userToken)
	sendPaymentRequest := lnrpc.SendRequest{
		PaymentRequest: invoice.PayReq,
		FeeLimit:       nil,
	}
	_, err = suite.fundingClient.SendPaymentSync(context.Background(), &sendPaymentRequest)
	assert.NoError(suite.T(), err)
	_, msg1, err := ws1.ReadMessage()
	assert.NoError(suite.T(), err)
	_, msg2, err := ws2.ReadMessage()
	assert.NoError(suite.T(), err)

	event1 := ExpectedInvoiceEventWrapper{}
	err = json.Unmarshal([]byte(msg1), &event1)
	assert.NoError(suite.T(), err)
	event2 := ExpectedInvoiceEventWrapper{}
	err = json.Unmarshal([]byte(msg2), &event2)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "integration test websocket 2", event1.Invoice.Description)
	assert.Equal(suite.T(), "integration test websocket 2", event2.Invoice.Description)
	//close 1 subscription, assert that the existing sub still receives their invoices
	ws1.Close()
	invoice = suite.createAddInvoiceReq(1000, "integration test websocket 3", suite.userToken)
	sendPaymentRequest = lnrpc.SendRequest{
		PaymentRequest: invoice.PayReq,
		FeeLimit:       nil,
	}
	_, err = suite.fundingClient.SendPaymentSync(context.Background(), &sendPaymentRequest)
	assert.NoError(suite.T(), err)
	_, msg3, err := ws2.ReadMessage()
	assert.NoError(suite.T(), err)
	event3 := ExpectedInvoiceEventWrapper{}
	err = json.Unmarshal([]byte(msg3), &event3)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "integration test websocket 3", event3.Invoice.Description)

}
func (suite *WebSocketTestSuite) TestWebSocketDoubleUser() {

	//create subs for 2 different users, assert that they each get their own invoice updates
	user1Ws, _, err := websocket.DefaultDialer.Dial(suite.wsUrl, nil)
	assert.NoError(suite.T(), err)
	//read keepalive msg
	_, _, err = user1Ws.ReadMessage()
	assert.NoError(suite.T(), err)
	//create subs for 2 different users, assert that they each get their own invoice updates
	user2Ws, _, err := websocket.DefaultDialer.Dial(suite.wsUrl2, nil)
	assert.NoError(suite.T(), err)
	//read keepalive msg
	_, _, err = user2Ws.ReadMessage()
	assert.NoError(suite.T(), err)
	// add invoice for user 1
	user1Invoice := suite.createAddInvoiceReq(1000, "integration test websocket user 1", suite.userToken)
	sendPaymentRequestUser1 := lnrpc.SendRequest{
		PaymentRequest: user1Invoice.PayReq,
		FeeLimit:       nil,
	}
	// add invoice for user 2
	user2Invoice := suite.createAddInvoiceReq(1000, "integration test websocket user 2", suite.userToken2)
	sendPaymentRequestUser2 := lnrpc.SendRequest{
		PaymentRequest: user2Invoice.PayReq,
		FeeLimit:       nil,
	}
	//pay invoices
	_, err = suite.fundingClient.SendPaymentSync(context.Background(), &sendPaymentRequestUser1)
	assert.NoError(suite.T(), err)
	_, err = suite.fundingClient.SendPaymentSync(context.Background(), &sendPaymentRequestUser2)
	assert.NoError(suite.T(), err)
	//read user 1 received msg
	_, user1Msg, err := user1Ws.ReadMessage()
	assert.NoError(suite.T(), err)
	//assert it's their's
	eventUser1 := ExpectedInvoiceEventWrapper{}
	err = json.Unmarshal([]byte(user1Msg), &eventUser1)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "integration test websocket user 1", eventUser1.Invoice.Description)
	//read user 2 received msg
	_, user2Msg, err := user2Ws.ReadMessage()
	assert.NoError(suite.T(), err)
	//assert it's their's
	eventUser2 := ExpectedInvoiceEventWrapper{}
	err = json.Unmarshal([]byte(user2Msg), &eventUser2)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "integration test websocket user 2", eventUser2.Invoice.Description)

}
func (suite *WebSocketTestSuite) TestWebSocketMissingInvoice() {
	// create incoming invoice and fund account
	invoice1 := suite.createAddInvoiceReq(1000, "integration test websocket missing invoices", suite.userToken)
	sendPaymentRequest := lnrpc.SendRequest{
		PaymentRequest: invoice1.PayReq,
		FeeLimit:       nil,
	}
	_, err := suite.fundingClient.SendPaymentSync(context.Background(), &sendPaymentRequest)
	assert.NoError(suite.T(), err)

	// create 2nd invoice and pay it as well
	invoice2 := suite.createAddInvoiceReq(1000, "integration test websocket missing invoices 2nd", suite.userToken)
	sendPaymentRequest = lnrpc.SendRequest{
		PaymentRequest: invoice2.PayReq,
		FeeLimit:       nil,
	}
	_, err = suite.fundingClient.SendPaymentSync(context.Background(), &sendPaymentRequest)
	assert.NoError(suite.T(), err)

	//start listening to websocket after 2nd invoice has been paid
	//we should get an event for the 2nd invoice if we specify the hash as the query parameter
	ws, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("%s&since_payment_hash=%s", suite.wsUrl, invoice1.RHash), nil)
	assert.NoError(suite.T(), err)
	_, msg, err := ws.ReadMessage()
	assert.NoError(suite.T(), err)
	keepAlive := KeepAlive{}
	err = json.Unmarshal([]byte(msg), &keepAlive)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "keepalive", keepAlive.Type)

	_, msg, err = ws.ReadMessage()
	assert.NoError(suite.T(), err)
	event := ExpectedInvoiceEventWrapper{}
	err = json.Unmarshal([]byte(msg), &event)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), event.Type, "invoice")
	assert.Equal(suite.T(), int64(1000), event.Invoice.Amount)
	assert.Equal(suite.T(), "integration test websocket missing invoices 2nd", event.Invoice.Description)
}

func (suite *WebSocketTestSuite) TearDownSuite() {
	suite.invoiceUpdateSubCancelFn()
	suite.websocketServer.Close()
	clearTable(suite.service, "invoices")
}

func TestWebSocketSuite(t *testing.T) {
	suite.Run(t, new(WebSocketTestSuite))
}
