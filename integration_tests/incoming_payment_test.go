package integration_tests

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
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
	fundingClient            *lnd.LNDWrapper
	service                  *service.LndhubService
	userLogin                ExpectedCreateUserResponseBody
	userToken                string
	invoiceUpdateSubCancelFn context.CancelFunc
}

func (suite *IncomingPaymentTestSuite) SetupSuite() {
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
	suite.echo.Use(tokens.Middleware([]byte(suite.service.Config.JWTSecret)))
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
	//try to pay invoice with external node
	// Prepare the LNRPC call
	sendPaymentRequest := lnrpc.SendRequest{
		PaymentRequest: invoiceResponse.PayReq,
		FeeLimit:       nil,
	}
	_, err := suite.fundingClient.SendPaymentSync(context.Background(), &sendPaymentRequest)
	assert.NoError(suite.T(), err)

	//wait a bit for the callback event to hit
	time.Sleep(100 * time.Millisecond)
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
	suite.echo.Use(tokens.Middleware([]byte(suite.service.Config.JWTSecret)))
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
	//try to pay invoice with external node
	// Prepare the LNRPC call
	sendPaymentRequest := lnrpc.SendRequest{
		PaymentRequest: invoiceResponse.PayReq,
		Amt:            int64(sendSatAmt),
		FeeLimit:       nil,
	}
	_, err := suite.fundingClient.SendPaymentSync(context.Background(), &sendPaymentRequest)
	assert.NoError(suite.T(), err)

	//wait a bit for the callback event to hit
	time.Sleep(100 * time.Millisecond)
	//check balance again
	req = httptest.NewRequest(http.MethodGet, "/balance", &buf)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken))
	suite.echo.ServeHTTP(rec, req)
	balance = &ExpectedBalanceResponse{}
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(&balance))
	//assert the payment value was added to the user's account
	assert.Equal(suite.T(), initialBalance+int64(sendSatAmt), balance.BTC.AvailableBalance)
}
func (suite *IncomingPaymentTestSuite) TestIncomingPaymentKeysend() {
	var buf bytes.Buffer
	req := httptest.NewRequest(http.MethodGet, "/balance", &buf)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken))
	rec := httptest.NewRecorder()
	suite.echo.Use(tokens.Middleware([]byte(suite.service.Config.JWTSecret)))
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
	destBytes, err := hex.DecodeString(suite.service.IdentityPubkey)
	assert.NoError(suite.T(), err)
	sendPaymentRequest := lnrpc.SendRequest{
		Dest:         destBytes,
		Amt:          int64(keysendSatAmt),
		PaymentHash:  pHash.Sum(nil),
		DestFeatures: []lnrpc.FeatureBit{lnrpc.FeatureBit_TLV_ONION_REQ},
		DestCustomRecords: map[uint64][]byte{
			service.TLV_WALLET_ID:         []byte(suite.userLogin.Login),
			service.KEYSEND_CUSTOM_RECORD: preImage,
		},
		FeeLimit: nil,
	}
	_, err = suite.fundingClient.SendPaymentSync(context.Background(), &sendPaymentRequest)
	assert.NoError(suite.T(), err)

	//wait a bit for the callback event to hit
	time.Sleep(100 * time.Millisecond)
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

func randBytesFromStr(length int, from string) ([]byte, error) {
	b := make([]byte, length)
	fromLenBigInt := big.NewInt(int64(len(from)))
	for i := range b {
		r, err := rand.Int(rand.Reader, fromLenBigInt)
		if err != nil {
			return nil, err
		}
		b[i] = from[r.Int64()]
	}
	return b, nil
}

func TestIncomingPaymentTestSuite(t *testing.T) {
	suite.Run(t, new(IncomingPaymentTestSuite))
}
