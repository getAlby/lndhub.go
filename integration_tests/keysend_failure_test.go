package integration_tests

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/getAlby/lndhub.go/common"
	"github.com/getAlby/lndhub.go/controllers"
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

type KeySendFailureTestSuite struct {
	TestSuite
	service                  *service.LndhubService
	mlnd                     *MockLND
	aliceLogin               ExpectedCreateUserResponseBody
	aliceToken               string
	invoiceUpdateSubCancelFn context.CancelFunc
	serviceClient            *LNDMockWrapperAsync
}

func (suite *KeySendFailureTestSuite) TearDownTest() {
	clearTable(suite.service, "transaction_entries")
	clearTable(suite.service, "invoices")
}

func (suite *KeySendFailureTestSuite) TearDownSuite() {
	suite.invoiceUpdateSubCancelFn()
}

func (suite *KeySendFailureTestSuite) SetupSuite() {
	fee := int64(1)
	mlnd := newDefaultMockLND()
	mlnd.fee = fee
	suite.mlnd = mlnd
	// inject fake lnd client with failing send payment sync into service
	lndClient, err := NewLNDMockWrapperAsync(mlnd)
	suite.serviceClient = lndClient
	if err != nil {
		log.Fatalf("Error setting up test client: %v", err)
	}
	svc, err := LndHubTestServiceInit(lndClient)
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
	suite.echo.POST("/addinvoice", controllers.NewAddInvoiceController(suite.service).AddInvoice)
	suite.echo.POST("/keysend", controllers.NewKeySendController(suite.service).KeySend)
}


func (suite *KeySendFailureTestSuite) TestKeysendPayment() {
	aliceFundingSats := 1000
	externalSatRequested := 500
	//fund alice account
	invoiceResponse := suite.createAddInvoiceReq(aliceFundingSats, "integration test external payment alice", suite.aliceToken)
	err := suite.mlnd.mockPaidInvoice(invoiceResponse, 0, false, nil)
	assert.NoError(suite.T(), err)

	//wait a bit for the callback event to hit
	time.Sleep(10 * time.Millisecond)

	go suite.createKeySendReqError(int64(externalSatRequested), "key send test", "123456789012345678901234567890123456789012345678901234567890abcdef", suite.aliceToken)

	suite.serviceClient.FailPayment(SendPaymentMockError)
	time.Sleep(2 * time.Second)

	// check that balance was reverted
	userId := getUserIdFromToken(suite.aliceToken)
	aliceBalance, err := suite.service.CurrentUserBalance(context.Background(), userId)
	if err != nil {
		fmt.Printf("Error when getting balance %v\n", err.Error())
	}
	assert.Equal(suite.T(), int64(aliceFundingSats), aliceBalance)

	invoices, err := suite.service.InvoicesFor(context.Background(), userId, common.InvoiceTypeOutgoing)
	if err != nil {
		fmt.Printf("Error when getting invoices %v\n", err.Error())
	}
	assert.Equal(suite.T(), 1, len(invoices))
	assert.Equal(suite.T(), common.InvoiceStateError, invoices[0].State)
	assert.Equal(suite.T(), SendPaymentMockError, invoices[0].ErrorMessage)
	assert.NotEmpty(suite.T(), invoices[0].RHash)
	assert.NotEmpty(suite.T(), invoices[0].Preimage)
}

func TestKeySendFailureTestSuite(t *testing.T) {
	suite.Run(t, new(KeySendFailureTestSuite))
}
