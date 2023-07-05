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

	"github.com/getAlby/lndhub.go/common"
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

type PaymentTestErrorsSuite struct {
	TestSuite
	service                  *service.LndhubService
	mlnd                     *MockLND
	externalLND              *MockLND
	userLogin                ExpectedCreateUserResponseBody
	userToken                string
	invoiceUpdateSubCancelFn context.CancelFunc
}

func (suite *PaymentTestErrorsSuite) SetupSuite() {
	mlnd := newDefaultMockLND()
	externalLND, err := NewMockLND("1234567890abcdefabcd", 0, make(chan (*lnrpc.Invoice)))
	if err != nil {
		log.Fatalf("Error initializing test service: %v", err)
	}
	// inject fake lnd client with failing send payment sync into service
	lndClient, err := NewLNDMockWrapper(mlnd)
	suite.mlnd = mlnd
	suite.externalLND = externalLND
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
	suite.userLogin = users[0]
	suite.userToken = userTokens[0]
	suite.echo.Use(tokens.Middleware([]byte(suite.service.Config.JWTSecret)))
	suite.echo.GET("/balance", controllers.NewBalanceController(suite.service).Balance)
	suite.echo.POST("/addinvoice", controllers.NewAddInvoiceController(suite.service).AddInvoice)
	suite.echo.POST("/payinvoice", controllers.NewPayInvoiceController(suite.service).PayInvoice)
}

func (suite *PaymentTestErrorsSuite) TestExternalFailingInvoice() {
	userFundingSats := 1000
	externalSatRequested := 500
	//fund user account
	invoiceResponse := suite.createAddInvoiceReq(userFundingSats, "integration test external payment user", suite.userToken)
	err := suite.mlnd.mockPaidInvoice(invoiceResponse, 0, false, nil)
	assert.NoError(suite.T(), err)

	//wait a bit for the callback event to hit
	time.Sleep(10 * time.Millisecond)

	//test an expired invoice
	externalInvoice := lnrpc.Invoice{
		Memo:   "integration tests: external pay from alice",
		Value:  int64(externalSatRequested),
		Expiry: 1,
	}
	invoice, err := suite.externalLND.AddInvoice(context.Background(), &externalInvoice)

	assert.NoError(suite.T(), err)

	//wait for the invoice to expire
	time.Sleep(2 * time.Second)

	rec := httptest.NewRecorder()
	var buf bytes.Buffer
	assert.NoError(suite.T(), json.NewEncoder(&buf).Encode(&ExpectedPayInvoiceRequestBody{
		Invoice: invoice.PaymentRequest,
		Amount:  10,
	}))

	req := httptest.NewRequest(http.MethodPost, "/payinvoice", &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken))
	suite.echo.ServeHTTP(rec, req)

	errorResponse := &responses.ErrorResponse{}
	assert.Equal(suite.T(), http.StatusBadRequest, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(errorResponse))
	assert.Equal(suite.T(), "invoice expired", errorResponse.Message)

	//create external invoice
	externalInvoice = lnrpc.Invoice{
		Memo:  "integration tests: external pay from user",
		Value: int64(externalSatRequested),
	}
	invoice, err = suite.externalLND.AddInvoice(context.Background(), &externalInvoice)
	assert.NoError(suite.T(), err)
	//pay external from user, mock will fail immediately
	_ = suite.createPayInvoiceReqError(invoice.PaymentRequest, suite.userToken)

	userId := getUserIdFromToken(suite.userToken)

	invoices, err := suite.service.InvoicesFor(context.Background(), userId, common.InvoiceTypeOutgoing)
	if err != nil {
		fmt.Printf("Error when getting invoices %v\n", err.Error())
	}
	assert.Equal(suite.T(), 1, len(invoices))
	assert.Equal(suite.T(), common.InvoiceStateError, invoices[0].State)
	assert.Equal(suite.T(), SendPaymentMockError, invoices[0].ErrorMessage)

	transactonEntries, err := suite.service.TransactionEntriesFor(context.Background(), userId)
	if err != nil {
		fmt.Printf("Error when getting transaction entries %v\n", err.Error())
	}

	userBalance, err := suite.service.CurrentUserBalance(context.Background(), userId)
	if err != nil {
		fmt.Printf("Error when getting balance %v\n", err.Error())
	}

	// check if there are 5 transaction entries:
	//	- the incoming payment
	//  - the outgoing payment
	//  - the fee reserve + the fee reserve reversal
	//  - the outgoing payment reversal
	//  with reversed credit and debit account ids for payment 2/5 & payment 3/4
	assert.Equal(suite.T(), 5, len(transactonEntries))
	assert.Equal(suite.T(), transactonEntries[1].CreditAccountID, transactonEntries[4].DebitAccountID)
	assert.Equal(suite.T(), transactonEntries[1].DebitAccountID, transactonEntries[4].CreditAccountID)
	assert.Equal(suite.T(), transactonEntries[2].CreditAccountID, transactonEntries[3].DebitAccountID)
	assert.Equal(suite.T(), transactonEntries[2].DebitAccountID, transactonEntries[3].CreditAccountID)
	assert.Equal(suite.T(), transactonEntries[1].Amount, int64(externalSatRequested))
	assert.Equal(suite.T(), transactonEntries[2].Amount, int64(externalSatRequested))
	// assert that balance is the same
	assert.Equal(suite.T(), int64(userFundingSats), userBalance)
}

func (suite *PaymentTestErrorsSuite) TearDownSuite() {
	suite.invoiceUpdateSubCancelFn()
}

func TestPaymentTestErrorsSuite(t *testing.T) {
	suite.Run(t, new(PaymentTestErrorsSuite))
}
