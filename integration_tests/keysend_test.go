package integration_tests

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/getAlby/lndhub.go/controllers"
	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/getAlby/lndhub.go/lib/tokens"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/lightningnetwork/lnd/lnrpc"
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
	mlnd, err := NewMockLND("1234567890abcdef", fee, make(chan (*lnrpc.Invoice)))
	if err != nil {
		log.Fatalf("Error initializing test service: %v", err)
	}
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
	suite.echo.Use(tokens.Middleware([]byte(suite.service.Config.JWTSecret)))
	suite.echo.GET("/balance", controllers.NewBalanceController(suite.service).Balance)
	suite.echo.POST("/addinvoice", controllers.NewAddInvoiceController(suite.service).AddInvoice)
	suite.echo.POST("/payinvoice", controllers.NewPayInvoiceController(suite.service).PayInvoice)
	suite.echo.POST("/keysend", controllers.NewKeySendController(suite.service).KeySend)
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

	suite.createKeySendReq(int64(externalSatRequested), "key send test", "03abcdef123456789a", suite.aliceToken)

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

	suite.createKeySendReqError(int64(externalSatRequested), "key send test", "12345", suite.aliceToken)
}

func TestKeySendTestSuite(t *testing.T) {
	suite.Run(t, new(KeySendTestSuite))
}
