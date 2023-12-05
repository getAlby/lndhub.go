package integration_tests

import (
	"bytes"
	"context"
	"crypto/sha256"
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
	"github.com/labstack/gommon/random"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type IncomingPaymentTestSuite struct {
	TestSuite
	service                  *service.LndhubService
	mockLND                  *MockLND
	userLogin                ExpectedCreateUserResponseBody
	userToken                string
	invoiceUpdateSubCancelFn context.CancelFunc
}

func (suite *IncomingPaymentTestSuite) SetupSuite() {
	mockLND := newDefaultMockLND()
	suite.mockLND = mockLND
	svc, err := LndHubTestServiceInit(mockLND)
	if err != nil {
		log.Fatalf("Error initializing test service: %v", err)
	}
	users, tokens, err := createUsers(svc, 1)
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
	assert.Equal(suite.T(), 1, len(tokens))
	suite.userLogin = users[0]
	suite.userToken = tokens[0]
}

func (suite *IncomingPaymentTestSuite) TearDownSuite() {
	suite.invoiceUpdateSubCancelFn()
}

func (suite *IncomingPaymentTestSuite) TestIncomingPayment() {
	var buf bytes.Buffer
	req := httptest.NewRequest(http.MethodGet, "/balance", &buf)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken))
	rec := httptest.NewRecorder()
	suite.echo.Use(tokens.Middleware([]byte(suite.service.Config.JWTSecret), &lnd.Limits{}))
	suite.echo.GET("/balance", controllers.NewBalanceController(suite.service).Balance)
	suite.echo.POST("/addinvoice", controllers.NewAddInvoiceController(suite.service).AddInvoice)
	suite.echo.ServeHTTP(rec, req)
	balance := &ExpectedBalanceResponse{}
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(&balance))
	//assert the user has no balance to start with
	assert.Equal(suite.T(), int64(0), balance.BTC.AvailableBalance)
	fundingSatAmt := 10
	invoiceResponse := suite.createAddInvoiceReq(fundingSatAmt, "integration test IncomingPaymentTestSuite", suite.userToken)
	err := suite.mockLND.mockPaidInvoice(invoiceResponse, 0, false, nil)
	assert.NoError(suite.T(), err)
	//wait a bit for the payment to be processed
	time.Sleep(10 * time.Millisecond)
	//check balance again
	req = httptest.NewRequest(http.MethodGet, "/balance", &buf)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken))
	suite.echo.ServeHTTP(rec, req)
	balance = &ExpectedBalanceResponse{}
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(&balance))
	//assert the balance was added to the user's account
	assert.Equal(suite.T(), int64(fundingSatAmt), balance.BTC.AvailableBalance)

}
func (suite *IncomingPaymentTestSuite) TestIncomingPaymentZeroAmt() {
	var buf bytes.Buffer
	req := httptest.NewRequest(http.MethodGet, "/balance", &buf)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken))
	rec := httptest.NewRecorder()
	suite.echo.Use(tokens.Middleware([]byte(suite.service.Config.JWTSecret), &lnd.Limits{}))
	suite.echo.GET("/balance", controllers.NewBalanceController(suite.service).Balance)
	suite.echo.POST("/addinvoice", controllers.NewAddInvoiceController(suite.service).AddInvoice)
	suite.echo.ServeHTTP(rec, req)
	//lookup balance before
	balance := &ExpectedBalanceResponse{}
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(&balance))
	initialBalance := balance.BTC.AvailableBalance
	fundingSatAmt := 0
	sendSatAmt := 10
	invoiceResponse := suite.createAddInvoiceReq(fundingSatAmt, "integration test IncomingPaymentTestSuite", suite.userToken)
	err := suite.mockLND.mockPaidInvoice(invoiceResponse, int64(sendSatAmt), false, nil)
	assert.NoError(suite.T(), err)
	//wait a bit for the callback event to hit
	time.Sleep(10 * time.Millisecond)

	//check balance again
	req = httptest.NewRequest(http.MethodGet, "/balance", &buf)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken))
	suite.echo.ServeHTTP(rec, req)
	balance = &ExpectedBalanceResponse{}
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(&balance))
	//assert the payment value was added to the user's account
	assert.Equal(suite.T(), initialBalance+int64(sendSatAmt), balance.BTC.AvailableBalance)

	//check transaction list
	req = httptest.NewRequest(http.MethodGet, "/getuserinvoices", &buf)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken))
	suite.echo.ServeHTTP(rec, req)
	invoices := make([]ExpectedIncomingInvoice, 1)
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(&invoices))
	//assert the payment value was added to the user's account
	assert.Equal(suite.T(), int64(sendSatAmt), invoices[0].Amount)
}
func (suite *IncomingPaymentTestSuite) TestIncomingPaymentKeysend() {
	var buf bytes.Buffer
	req := httptest.NewRequest(http.MethodGet, "/balance", &buf)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken))
	rec := httptest.NewRecorder()
	suite.echo.Use(tokens.Middleware([]byte(suite.service.Config.JWTSecret), &lnd.Limits{}))
	suite.echo.GET("/balance", controllers.NewBalanceController(suite.service).Balance)
	suite.echo.GET("/getuserinvoices", controllers.NewGetTXSController(suite.service).GetUserInvoices)
	suite.echo.ServeHTTP(rec, req)
	//lookup balance before
	balance := &ExpectedBalanceResponse{}
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(&balance))
	initialBalance := balance.BTC.AvailableBalance
	keysendSatAmt := 10

	//make keysend payment
	pHash := sha256.New()
	preImage, err := randBytesFromStr(32, random.Hex)
	assert.NoError(suite.T(), err)
	pHash.Write(preImage)
	err = suite.mockLND.mockPaidInvoice(nil, int64(keysendSatAmt), true, &lnrpc.InvoiceHTLC{
		CustomRecords: map[uint64][]byte{
			service.TLV_WALLET_ID:         []byte(suite.userLogin.Login),
			service.KEYSEND_CUSTOM_RECORD: preImage,
		},
	})
	assert.NoError(suite.T(), err)

	//wait a bit for the callback event to hit
	time.Sleep(10 * time.Millisecond)
	//check balance again
	req = httptest.NewRequest(http.MethodGet, "/balance", &buf)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken))
	suite.echo.ServeHTTP(rec, req)
	balance = &ExpectedBalanceResponse{}
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(&balance))
	//assert the payment value was added to the user's account
	assert.Equal(suite.T(), initialBalance+int64(keysendSatAmt), balance.BTC.AvailableBalance)

	//Look up payment to check the custom records
	// check invoices again
	req = httptest.NewRequest(http.MethodGet, "/getuserinvoices", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken))
	rec = httptest.NewRecorder()
	suite.echo.ServeHTTP(rec, req)
	// controller := controllers.NewGetTXSController(suite.Service)
	incomingPayments := &[]ExpectedIncomingInvoice{}
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(incomingPayments))
	//find the keysend payment, there should be only 1
	var keySendPayment ExpectedIncomingInvoice
	for _, payment := range *incomingPayments {
		if payment.Keysend {
			keySendPayment = payment
			break
		}
	}
	assert.True(suite.T(), keySendPayment.Keysend)
	login := keySendPayment.CustomRecords[service.TLV_WALLET_ID]
	assert.Equal(suite.T(), suite.userLogin.Login, string(login))
}

func TestIncomingPaymentTestSuite(t *testing.T) {
	suite.Run(t, new(IncomingPaymentTestSuite))
}
