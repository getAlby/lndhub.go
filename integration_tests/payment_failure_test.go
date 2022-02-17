package integration_tests

import (
	"context"
	"fmt"
	"log"
	"testing"

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

type PaymentTestErrorsSuite struct {
	TestSuite
	fundingClient *lnd.LNDWrapper
	service       *service.LndhubService
	aliceLogin    controllers.CreateUserResponseBody
	aliceToken    string
	bobLogin      controllers.CreateUserResponseBody
	bobToken      string
	bobId         string
}

func (suite *PaymentTestErrorsSuite) SetupSuite() {
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
	users, userTokens, err := createUsers(svc, 2)
	if err != nil {
		log.Fatalf("Error creating test users: %v", err)
	}
	// Subscribe to LND invoice updates in the background
	go svc.InvoiceUpdateSubscription(context.Background())
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

func (suite *PaymentTestErrorsSuite) TestExternalFailingInvoice() {
	//create external zero amount invoice that will fail
	externalInvoice := lnrpc.Invoice{
		Memo:  "integration tests: external failing pay",
		Value: 0,
	}
	invoice, err := suite.fundingClient.AddInvoice(context.Background(), &externalInvoice)
	assert.NoError(suite.T(), err)
	_ = suite.createPayInvoiceReqError(invoice.PaymentRequest, suite.bobToken)

	userId := getUserIdFromToken(suite.bobToken)

	invoices, err := suite.service.InvoicesFor(context.Background(), userId, common.InvoiceTypeOutgoing)
	if err != nil {
		fmt.Printf("Error when getting invoices %v\n", err.Error())
	}
	assert.Equal(suite.T(), 1, len(invoices))
	assert.Equal(suite.T(), common.InvoiceStateError, invoices[0].State)

	transactonEntries, err := suite.service.TransactionEntriesFor(context.Background(), userId)
	if err != nil {
		fmt.Printf("Error when getting transaction entries %v\n", err.Error())
	}

	// check if there are 2 transaction entries, with reversed credit and debit account ids
	assert.Equal(suite.T(), 2, len(transactonEntries))
	assert.Equal(suite.T(), transactonEntries[0].CreditAccountID, transactonEntries[1].DebitAccountID)
	assert.Equal(suite.T(), transactonEntries[0].DebitAccountID, transactonEntries[1].CreditAccountID)
}

func (suite *PaymentTestErrorsSuite) TearDownSuite() {

}

func TestPaymentTestErrorsSuite(t *testing.T) {
	suite.Run(t, new(PaymentTestErrorsSuite))
}
