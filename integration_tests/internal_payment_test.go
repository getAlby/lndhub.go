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
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type PaymentTestSuite struct {
	TestSuite
	fundingClient            *lnd.LNDWrapper
	service                  *service.LndhubService
	aliceLogin               controllers.CreateUserResponseBody
	aliceToken               string
	bobLogin                 controllers.CreateUserResponseBody
	bobToken                 string
	invoiceUpdateSubCancelFn context.CancelFunc
}

func (suite *PaymentTestSuite) SetupSuite() {
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
	suite.aliceLogin = users[0]
	suite.aliceToken = userTokens[0]
	suite.bobLogin = users[1]
	suite.bobToken = userTokens[1]
	suite.echo.Use(tokens.Middleware([]byte(suite.service.Config.JWTSecret)))
	suite.echo.GET("/balance", controllers.NewBalanceController(suite.service).Balance)
	suite.echo.POST("/addinvoice", controllers.NewAddInvoiceController(suite.service).AddInvoice)
	suite.echo.POST("/payinvoice", controllers.NewPayInvoiceController(suite.service).PayInvoice)
}

func (suite *PaymentTestSuite) TearDownSuite() {
	suite.invoiceUpdateSubCancelFn()
}

func (suite *PaymentTestSuite) TearDownTest() {
	clearTable(suite.service, "transaction_entries")
	clearTable(suite.service, "invoices")
}

func (suite *PaymentTestSuite) TestInternalPayment() {
	aliceFundingSats := 1000
	bobSatRequested := 500
	//fund alice account
	invoiceResponse := suite.createAddInvoiceReq(aliceFundingSats, "integration test internal payment alice", suite.aliceToken)
	sendPaymentRequest := lnrpc.SendRequest{
		PaymentRequest: invoiceResponse.PayReq,
		FeeLimit:       nil,
	}
	_, err := suite.fundingClient.SendPaymentSync(context.Background(), &sendPaymentRequest)
	assert.NoError(suite.T(), err)

	//wait a bit for the callback event to hit
	time.Sleep(100 * time.Millisecond)

	//create invoice for bob
	bobInvoice := suite.createAddInvoiceReq(bobSatRequested, "integration test internal payment bob", suite.bobToken)
	//pay bob from alice
	payResponse := suite.createPayInvoiceReq(bobInvoice.PayReq, suite.aliceToken)
	assert.NotEmpty(suite.T(), payResponse.PaymentPreimage)
	//try to pay Bob more than we currently have
	//create invoice for bob
	tooMuch := suite.createAddInvoiceReq(10000, "integration test internal payment bob", suite.bobToken)
	//pay bob from alice
	errorResp := suite.createPayInvoiceReqError(tooMuch.PayReq, suite.aliceToken)
	assert.Equal(suite.T(), responses.NotEnoughBalanceError.Code, errorResp.Code)
}

func (suite *PaymentTestSuite) TestInternalPaymentFail() {
	aliceFundingSats := 1000
	bobSatRequested := 500
	//fund alice account
	invoiceResponse := suite.createAddInvoiceReq(aliceFundingSats, "integration test internal payment alice", suite.aliceToken)
	sendPaymentRequest := lnrpc.SendRequest{
		PaymentRequest: invoiceResponse.PayReq,
		FeeLimit:       nil,
	}
	_, err := suite.fundingClient.SendPaymentSync(context.Background(), &sendPaymentRequest)
	assert.NoError(suite.T(), err)

	//wait a bit for the callback event to hit
	time.Sleep(100 * time.Millisecond)

	//create invoice for bob
	bobInvoice := suite.createAddInvoiceReq(bobSatRequested, "integration test internal payment bob", suite.bobToken)
	//pay bob from alice
	payResponse := suite.createPayInvoiceReq(bobInvoice.PayReq, suite.aliceToken)
	assert.NotEmpty(suite.T(), payResponse.PaymentPreimage)
	//try to pay same invoice again for make it fail
	_ = suite.createPayInvoiceReqError(bobInvoice.PayReq, suite.aliceToken)

	userId := getUserIdFromToken(suite.aliceToken)
	invoices, err := suite.service.InvoicesFor(context.Background(), userId, common.InvoiceTypeOutgoing)
	if err != nil {
		fmt.Printf("Error when getting invoices %v\n", err.Error())
	}

	// check if first one is settled, but second one error (they are ordered desc by id)
	assert.Equal(suite.T(), 2, len(invoices))
	assert.Equal(suite.T(), common.InvoiceStateError, invoices[0].State)
	assert.Equal(suite.T(), common.InvoiceStateSettled, invoices[1].State)
	transactonEntries, err := suite.service.TransactionEntriesFor(context.Background(), userId)
	if err != nil {
		fmt.Printf("Error when getting transaction entries %v\n", err.Error())
	}

	aliceBalance, err := suite.service.CurrentUserBalance(context.Background(), userId)
	if err != nil {
		fmt.Printf("Error when getting balance %v\n", err.Error())
	}

	// check if there are 4 transaction entries, with reversed credit and debit account ids for last 2
	assert.Equal(suite.T(), 4, len(transactonEntries))
	assert.Equal(suite.T(), transactonEntries[2].CreditAccountID, transactonEntries[3].DebitAccountID)
	assert.Equal(suite.T(), transactonEntries[2].DebitAccountID, transactonEntries[3].CreditAccountID)
	assert.Equal(suite.T(), transactonEntries[2].Amount, int64(bobSatRequested))
	assert.Equal(suite.T(), transactonEntries[3].Amount, int64(bobSatRequested))
	// assert that balance was reduced only once
	assert.Equal(suite.T(), int64(bobSatRequested), int64(aliceBalance))
}

func TestInternalPaymentTestSuite(t *testing.T) {
	suite.Run(t, new(PaymentTestSuite))
}
