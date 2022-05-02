package integration_tests

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getAlby/lndhub.go/common"
	"github.com/getAlby/lndhub.go/controllers"
	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/getAlby/lndhub.go/lib/tokens"
	"github.com/getAlby/lndhub.go/lnd"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type WebHookTestSuite struct {
	TestSuite
	fundingClient            *lnd.LNDWrapper
	service                  *service.LndhubService
	userLogin                ExpectedCreateUserResponseBody
	userToken                string
	webHookServer            *httptest.Server
	invoiceChan              chan (models.Invoice)
	invoiceUpdateSubCancelFn context.CancelFunc
}

func (suite *WebHookTestSuite) SetupSuite() {
	lndClient, err := lnd.NewLNDclient(lnd.LNDoptions{
		Address:     lnd2RegtestAddress,
		MacaroonHex: lnd2RegtestMacaroonHex,
	})
	if err != nil {
		log.Fatalf("Error setting up funding client: %v", err)
	}
	suite.fundingClient = lndClient

	suite.invoiceChan = make(chan models.Invoice)
	webhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		invoice := models.Invoice{}
		err = json.NewDecoder(r.Body).Decode(&invoice)
		if err != nil {
			suite.echo.Logger.Error(err)
			close(suite.invoiceChan)
			return
		}
		suite.invoiceChan <- invoice
	}))
	suite.webHookServer = webhookServer
	svc, err := LndHubTestServiceInit(nil)
	if err != nil {
		log.Fatalf("Error initializing test service: %v", err)
	}
	svc.Config.WebhookUrl = suite.webHookServer.URL

	users, userTokens, err := createUsers(svc, 1)
	if err != nil {
		log.Fatalf("Error creating test users: %v", err)
	}
	// Subscribe to LND invoice updates in the background
	// store cancel func to be called in tear down suite
	ctx, cancel := context.WithCancel(context.Background())
	suite.invoiceUpdateSubCancelFn = cancel
	go svc.InvoiceUpdateSubscription(ctx)

	go svc.StartWebhookSubscribtion(ctx, svc.Config.WebhookUrl)

	suite.service = svc
	e := echo.New()

	e.HTTPErrorHandler = responses.HTTPErrorHandler
	e.Validator = &lib.CustomValidator{Validator: validator.New()}
	suite.echo = e
	suite.userLogin = users[0]
	suite.userToken = userTokens[0]
	suite.echo.Use(tokens.Middleware([]byte(suite.service.Config.JWTSecret)))
	suite.echo.POST("/addinvoice", controllers.NewAddInvoiceController(suite.service).AddInvoice)
}
func (suite *WebHookTestSuite) TestWebHook() {
	// create incoming invoice and fund account
	invoice := suite.createAddInvoiceReq(1000, "integration test webhook", suite.userToken)
	sendPaymentRequest := lnrpc.SendRequest{
		PaymentRequest: invoice.PayReq,
		FeeLimit:       nil,
	}
	_, err := suite.fundingClient.SendPaymentSync(context.Background(), &sendPaymentRequest)
	assert.NoError(suite.T(), err)
	invoiceFromWebhook := <-suite.invoiceChan
	assert.Equal(suite.T(), "integration test webhook", invoiceFromWebhook.Memo)
	assert.Equal(suite.T(), common.InvoiceTypeIncoming, invoiceFromWebhook.Type)
}
func (suite *WebHookTestSuite) TearDownSuite() {
	suite.invoiceUpdateSubCancelFn()
	suite.webHookServer.Close()
	clearTable(suite.service, "invoices")
}

func TestWebHookSuite(t *testing.T) {
	suite.Run(t, new(WebHookTestSuite))
}
