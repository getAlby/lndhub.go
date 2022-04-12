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
	invoiceUpdateSubCancelFn context.CancelFunc
	websocketServer          *httptest.Server
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
	users, userTokens, err := createUsers(svc, 1)
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
	assert.Equal(suite.T(), 1, len(users))
	assert.Equal(suite.T(), 1, len(userTokens))
	suite.userLogin = users[0]
	suite.userToken = userTokens[0]
	suite.echo.Use(tokens.Middleware([]byte(suite.service.Config.JWTSecret)))
	suite.echo.POST("/addinvoice", controllers.NewAddInvoiceController(suite.service).AddInvoice)

	//websocket server
	h := WsHandler{handler: controllers.NewInvoiceStreamController(suite.service).StreamInvoices}
	server := httptest.NewServer(http.HandlerFunc(h.ServeHTTP))
	suite.websocketServer = server
}

func (suite *WebSocketTestSuite) TestWebSocket() {

	//start listening to websocket
	wsURL := "ws" + strings.TrimPrefix(suite.websocketServer.URL, "http") + fmt.Sprintf("?token=%s", suite.userToken)
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(suite.T(), err, err)
	//assert that there is a subscription
	_, msg, err := ws.ReadMessage()
	assert.NoError(suite.T(), err, err)
	keepAlive := KeepAlive{}
	err = json.Unmarshal([]byte(msg), &keepAlive)
	assert.NoError(suite.T(), err, err)
	assert.Equal(suite.T(), "keepalive", keepAlive.Type)
	//assert that there are no more subscriptions

	//create subscription, create invoice, pay invoice, assert that invoice is received
	//create 2nd subscription, create invoice, pay invoice, assert that invoice is received twice
	//assert that there are 2 subscriptions
	//close 1 subscription, assert that there is a single subscription left, assert that the existing sub still receives their invoices
	//create subs for 2 different users, assert that they each get their own invoice updates
}

func (suite *WebSocketTestSuite) TearDownSuite() {
	suite.invoiceUpdateSubCancelFn()
	suite.websocketServer.Close()
}

func TestWebSocketSuite(t *testing.T) {
	suite.Run(t, new(WebSocketTestSuite))
}
