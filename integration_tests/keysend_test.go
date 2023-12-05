package integration_tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/getAlby/lndhub.go/controllers"
	v2controllers "github.com/getAlby/lndhub.go/controllers_v2"
	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/getAlby/lndhub.go/lib/tokens"
	"github.com/getAlby/lndhub.go/lnd"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type KeySendTestSuite struct {
	TestSuite
	service                  *service.LndhubService
	mlnd                     *MockLND
	aliceLogin               ExpectedCreateUserResponseBody
	aliceToken               string
	invoiceUpdateSubCancelFn context.CancelFunc
}

func (suite *KeySendTestSuite) SetupSuite() {
	fee := int64(1)
	mlnd := newDefaultMockLND()
	mlnd.fee = fee
	suite.mlnd = mlnd
	svc, err := LndHubTestServiceInit(mlnd)
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
	suite.aliceLogin = users[0]
	suite.aliceToken = userTokens[0]
	suite.echo.Use(tokens.Middleware([]byte(suite.service.Config.JWTSecret), &lnd.Limits{}))
	suite.echo.GET("/balance", controllers.NewBalanceController(suite.service).Balance)
	suite.echo.POST("/addinvoice", controllers.NewAddInvoiceController(suite.service).AddInvoice)
	suite.echo.POST("/payinvoice", controllers.NewPayInvoiceController(suite.service).PayInvoice)
	suite.echo.POST("/keysend", controllers.NewKeySendController(suite.service).KeySend)
	suite.echo.POST("/v2/payments/keysend/multi", v2controllers.NewKeySendController(suite.service).MultiKeySend)
}

func (suite *KeySendTestSuite) TearDownTest() {
	clearTable(suite.service, "transaction_entries")
	clearTable(suite.service, "invoices")
}

func (suite *KeySendTestSuite) TearDownSuite() {
	suite.invoiceUpdateSubCancelFn()
}

func (suite *KeySendTestSuite) TestKeysendPayment() {
	aliceFundingSats := 1000
	externalSatRequested := 500
	//fund alice account
	invoiceResponse := suite.createAddInvoiceReq(aliceFundingSats, "integration test external payment alice", suite.aliceToken)
	err := suite.mlnd.mockPaidInvoice(invoiceResponse, 0, false, nil)
	assert.NoError(suite.T(), err)

	//wait a bit for the callback event to hit
	time.Sleep(10 * time.Millisecond)

	suite.createKeySendReq(int64(externalSatRequested), "key send test", "123456789012345678901234567890123456789012345678901234567890abcdef", suite.aliceToken)
	// check that balance was reduced
	userId := getUserIdFromToken(suite.aliceToken)
	aliceBalance, err := suite.service.CurrentUserBalance(context.Background(), userId)
	if err != nil {
		fmt.Printf("Error when getting balance %v\n", err.Error())
	}
	assert.Equal(suite.T(), int64(aliceFundingSats)-int64(externalSatRequested+int(suite.mlnd.fee)), aliceBalance)
}

func (suite *KeySendTestSuite) TestKeysendPaymentNonExistentDestination() {
	aliceFundingSats := 1000
	externalSatRequested := 500
	//fund alice account
	invoiceResponse := suite.createAddInvoiceReq(aliceFundingSats, "integration test external payment alice", suite.aliceToken)
	err := suite.mlnd.mockPaidInvoice(invoiceResponse, 0, false, nil)
	assert.NoError(suite.T(), err)

	//wait a bit for the callback event to hit
	time.Sleep(100 * time.Millisecond)

	errResponse := suite.createKeySendReqError(int64(externalSatRequested), "key send test", "12345", suite.aliceToken)
	assert.Equal(suite.T(), "invalid destination pubkey", errResponse.Message)
}

func (suite *KeySendTestSuite) TestMultiKeysend() {
	aliceFundingSats := 1000
	//fund alice account
	invoiceResponse := suite.createAddInvoiceReq(aliceFundingSats, "integration test external payment alice", suite.aliceToken)
	err := suite.mlnd.mockPaidInvoice(invoiceResponse, 0, false, nil)
	assert.NoError(suite.T(), err)
	//wait a bit for the callback event to hit
	time.Sleep(100 * time.Millisecond)

	rec := httptest.NewRecorder()
	var buf bytes.Buffer
	assert.NoError(suite.T(), json.NewEncoder(&buf).Encode(v2controllers.MultiKeySendRequestBody{
		Keysends: []v2controllers.KeySendRequestBody{
			{
				Amount:      150,
				Destination: "123456789012345678901234567890123456789012345678901234567890abcdef",
			},
			{
				Amount:      100,
				Destination: "123456789012345678901234567890123456789012345678901234567890abcdef",
			},
			{
				Amount:      50,
				Destination: "123456789012345678901234567890123456789012345678901234567890abcdef",
			},
		},
	}))
	req := httptest.NewRequest(http.MethodPost, "/v2/payments/keysend/multi", &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.aliceToken))
	suite.echo.ServeHTTP(rec, req)

	keySendResponse := &v2controllers.MultiKeySendResponseBody{}
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(keySendResponse))
	//check response
	assert.Equal(suite.T(), len(keySendResponse.Keysends), 3)
	//check that balance was reduced appropriately
	userId := getUserIdFromToken(suite.aliceToken)
	aliceBalance, err := suite.service.CurrentUserBalance(context.Background(), userId)
	if err != nil {
		fmt.Printf("Error when getting balance %v\n", err.Error())
	}
	assert.Equal(suite.T(), int64(aliceFundingSats)-300-3*suite.mlnd.fee, aliceBalance)
}

func TestKeySendTestSuite(t *testing.T) {
	suite.Run(t, new(KeySendTestSuite))
}
