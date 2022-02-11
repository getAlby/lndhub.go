package integration_tests

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/getAlby/lndhub.go/controllers"
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

type GetTxTestSuite struct {
	TestSuite
	Service                  *service.LndhubService
	fundingClient            *lnd.LNDWrapper
	userLogin                controllers.CreateUserResponseBody
	userToken                string
	invoiceUpdateSubCancelFn context.CancelFunc
}

type GetOutgoingInvoiceResponseTest struct {
	RHash           interface{} `json:"r_hash"`
	PaymentHash     interface{} `json:"payment_hash"`
	PaymentPreimage string      `json:"payment_preimage"`
	Value           int64       `json:"value"`
	Fee             int64       `json:"fee"`
	Timestamp       int64       `json:"timestamp"`
	Memo            string      `json:"memo"`
}

type GetIncomingInvoiceResponseTest struct {
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

func (suite *GetTxTestSuite) SetupSuite() {
	lndClient, err := lnd.NewLNDclient(lnd.LNDoptions{
		Address:     lnd2RegtestAddress,
		MacaroonHex: lnd2RegtestMacaroonHex,
	})
	if err != nil {
		log.Fatalf("Error setting up funding client: %v", err)
	}
	suite.fundingClient = lndClient

	svc, err := LndHubTestServiceInit()
	if err != nil {
		log.Fatalf("Error initializing test service: %v", err)
	}
	users, userTokens, err := createUsers(svc, 1)
	if err != nil {
		log.Fatalf("Error creating test users %v", err)
	}
	// Subscribe to LND invoice updates in the background
	// store cancel func to be called in tear down suite
	ctx, cancel := context.WithCancel(context.Background())
	suite.invoiceUpdateSubCancelFn = cancel
	go svc.InvoiceUpdateSubscription(ctx)
	suite.Service = svc
	e := echo.New()

	e.HTTPErrorHandler = responses.HTTPErrorHandler
	e.Validator = &lib.CustomValidator{Validator: validator.New()}
	suite.echo = e
	suite.echo.Use(tokens.Middleware([]byte(suite.Service.Config.JWTSecret)))
	suite.echo.GET("/gettxs", controllers.NewGetTXSController(suite.Service).GetTXS)
	suite.echo.GET("/getuserinvoices", controllers.NewGetTXSController(svc).GetUserInvoices)
	suite.echo.POST("/addinvoice", controllers.NewAddInvoiceController(suite.Service).AddInvoice)
	suite.echo.POST("/payinvoice", controllers.NewPayInvoiceController(suite.Service).PayInvoice)

	assert.Equal(suite.T(), 1, len(users))
	suite.userLogin = users[0]
	suite.userToken = userTokens[0]
}

func (suite *GetTxTestSuite) TearDownSuite() {
	suite.invoiceUpdateSubCancelFn()
}

func (suite *GetTxTestSuite) TearDownTest() {}

func (suite *GetTxTestSuite) TestGetOutgoingInvoices() {
	// check that invoices are empty
	req := httptest.NewRequest(http.MethodGet, "/gettxs", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken))
	rec := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec, req)
	responseBody := &[]GetOutgoingInvoiceResponseTest{}
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(&responseBody))
	assert.Empty(suite.T(), responseBody)
	// create incoming invoice and fund account
	invoice := suite.createAddInvoiceReq(1000, "integration test internal payment alice", suite.userToken)
	sendPaymentRequest := lnrpc.SendRequest{
		PaymentRequest: invoice.PayReq,
		FeeLimit:       nil,
	}
	_, err := suite.fundingClient.SendPaymentSync(context.Background(), &sendPaymentRequest)
	assert.NoError(suite.T(), err)

	//wait a bit for the callback event to hit
	time.Sleep(100 * time.Millisecond)
	// create invoice
	invoice = suite.createAddInvoiceReq(500, "integration test internal payment alice", suite.userToken)
	// pay invoice, this will create outgoing invoice and settle it
	suite.createPayInvoiceReq(invoice.PayReq, suite.userToken)
	// check invoices again
	req = httptest.NewRequest(http.MethodGet, "/gettxs", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken))
	rec = httptest.NewRecorder()
	suite.echo.ServeHTTP(rec, req)
	responseBody = &[]GetOutgoingInvoiceResponseTest{}
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(&responseBody))
	assert.Equal(suite.T(), 1, len(*responseBody))
}

func (suite *GetTxTestSuite) TestGetIncomingInvoices() {
	// check that invoices are empty
	req := httptest.NewRequest(http.MethodGet, "/getuserinvoices", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken))
	rec := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec, req)
	responseBody := &[]GetIncomingInvoiceResponseTest{}
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(&responseBody))
	assert.Empty(suite.T(), responseBody)
	// create incoming invoice
	suite.createAddInvoiceReq(1000, "integration test internal payment", suite.userToken)
	// check invoices again
	req = httptest.NewRequest(http.MethodGet, "/getuserinvoices", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken))
	rec = httptest.NewRecorder()
	suite.echo.ServeHTTP(rec, req)
	// controller := controllers.NewGetTXSController(suite.Service)
	responseBody = &[]GetIncomingInvoiceResponseTest{}
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(&responseBody))
	assert.Equal(suite.T(), 1, len(*responseBody))
}

func TestGetTXsTestSuite(t *testing.T) {
	suite.Run(t, new(GetTxTestSuite))
}
