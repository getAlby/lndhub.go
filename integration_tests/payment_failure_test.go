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
	suite.service.Config.ServiceFee = 1
	userFundingSats := 1000
	externalSatRequested := 500
	expectedServiceFee := 1
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

	// verify transaction entries data
	feeAccount, _ := suite.service.AccountFor(context.Background(), common.AccountTypeFees, userId)
	incomingAccount, _ := suite.service.AccountFor(context.Background(), common.AccountTypeIncoming, userId)
	outgoingAccount, _ := suite.service.AccountFor(context.Background(), common.AccountTypeOutgoing, userId)
	currentAccount, _ := suite.service.AccountFor(context.Background(), common.AccountTypeCurrent, userId)

	outgoingInvoices, err := invoicesFor(suite.service, userId, common.InvoiceTypeOutgoing)
	if err != nil {
		fmt.Printf("Error when getting invoices %v\n", err.Error())
	}
	incomingInvoices, err := invoicesFor(suite.service, userId, common.InvoiceTypeIncoming)
	if err != nil {
		fmt.Printf("Error when getting invoices %v\n", err.Error())
	}
	assert.Equal(suite.T(), 1, len(outgoingInvoices))
	assert.Equal(suite.T(), common.InvoiceStateError, outgoingInvoices[0].State)
	assert.Equal(suite.T(), SendPaymentMockError, outgoingInvoices[0].ErrorMessage)

	transactionEntries, err := suite.service.TransactionEntriesFor(context.Background(), userId)
	if err != nil {
		fmt.Printf("Error when getting transaction entries %v\n", err.Error())
	}

	userBalance, err := suite.service.CurrentUserBalance(context.Background(), userId)
	if err != nil {
		fmt.Printf("Error when getting balance %v\n", err.Error())
	}

	// check if there are 7 transaction entries:
	//	- [0] incoming
	//  - [1] outgoing
	//  - [2] fee_reserve
	//  - [3] service_fee
	//  - [4] fee_reserve_reversal
	//  - [5] service_fee_reversal
	//  - [6] outgoing_reversal
	//

	fmt.Printf("%v", transactionEntries[0])
	fmt.Println("")
	fmt.Printf("%v", transactionEntries[1])
	fmt.Println("")
	fmt.Printf("%v", transactionEntries[2])
	fmt.Println("")
	fmt.Printf("%v", transactionEntries[3])
	fmt.Println("")
	fmt.Printf("%v", transactionEntries[4])
	fmt.Println("")
	fmt.Printf("%v", transactionEntries[5])
	fmt.Println("")
	fmt.Printf("%v", transactionEntries[6])
	fmt.Println("")

	assert.Equal(suite.T(), 7, len(transactionEntries))

	// the incoming funding
	assert.Equal(suite.T(), int64(userFundingSats), transactionEntries[0].Amount)
	assert.Equal(suite.T(), currentAccount.ID, transactionEntries[0].CreditAccountID)
	assert.Equal(suite.T(), incomingAccount.ID, transactionEntries[0].DebitAccountID)
	assert.Equal(suite.T(), int64(0), transactionEntries[0].ParentID)
	assert.Equal(suite.T(), incomingInvoices[0].ID, transactionEntries[0].InvoiceID)

	// the outgoing payment
	assert.Equal(suite.T(), int64(externalSatRequested), transactionEntries[1].Amount)
	assert.Equal(suite.T(), outgoingAccount.ID, transactionEntries[1].CreditAccountID)
	assert.Equal(suite.T(), currentAccount.ID, transactionEntries[1].DebitAccountID)
	assert.Equal(suite.T(), int64(0), transactionEntries[1].ParentID)
	assert.Equal(suite.T(), outgoingInvoices[0].ID, transactionEntries[1].InvoiceID)

	// fee reserve + fee reserve reversal
	assert.Equal(suite.T(), transactionEntries[4].Amount, transactionEntries[2].Amount) // the amount of the fee_reserve and the fee_reserve_reversal must be equal
	assert.Equal(suite.T(), feeAccount.ID, transactionEntries[2].CreditAccountID)
	assert.Equal(suite.T(), currentAccount.ID, transactionEntries[2].DebitAccountID)
	assert.Equal(suite.T(), currentAccount.ID, transactionEntries[4].CreditAccountID)
	assert.Equal(suite.T(), feeAccount.ID, transactionEntries[4].DebitAccountID)
	assert.Equal(suite.T(), outgoingInvoices[0].ID, transactionEntries[2].InvoiceID)
	assert.Equal(suite.T(), outgoingInvoices[0].ID, transactionEntries[4].InvoiceID)

	// service fee
	assert.Equal(suite.T(), int64(expectedServiceFee), transactionEntries[3].Amount)
	assert.Equal(suite.T(), feeAccount.ID, transactionEntries[3].CreditAccountID)
	assert.Equal(suite.T(), currentAccount.ID, transactionEntries[3].DebitAccountID)
	assert.Equal(suite.T(), outgoingInvoices[0].ID, transactionEntries[3].InvoiceID)
	// service fee reversal
	assert.Equal(suite.T(), int64(expectedServiceFee), transactionEntries[5].Amount)
	assert.Equal(suite.T(), currentAccount.ID, transactionEntries[5].CreditAccountID)
	assert.Equal(suite.T(), feeAccount.ID, transactionEntries[5].DebitAccountID)
	assert.Equal(suite.T(), outgoingInvoices[0].ID, transactionEntries[5].InvoiceID)

	// the outgoing payment reversal
	assert.Equal(suite.T(), int64(externalSatRequested), transactionEntries[6].Amount)
	assert.Equal(suite.T(), currentAccount.ID, transactionEntries[6].CreditAccountID)
	assert.Equal(suite.T(), outgoingAccount.ID, transactionEntries[6].DebitAccountID)
	assert.Equal(suite.T(), int64(0), transactionEntries[1].ParentID)
	assert.Equal(suite.T(), outgoingInvoices[0].ID, transactionEntries[6].InvoiceID)

	// outgoing debit account must be the outgoing reversal credit account
	assert.Equal(suite.T(), transactionEntries[1].CreditAccountID, transactionEntries[6].DebitAccountID)
	assert.Equal(suite.T(), transactionEntries[1].DebitAccountID, transactionEntries[6].CreditAccountID)
	// outgoing amounts and reversal amounts
	assert.Equal(suite.T(), transactionEntries[1].Amount, int64(externalSatRequested))
	assert.Equal(suite.T(), transactionEntries[6].Amount, int64(externalSatRequested))
	// assert that balance is the same
	assert.Equal(suite.T(), int64(userFundingSats), userBalance)

	suite.service.Config.ServiceFee = 0 // reset service fee - we don't expect this everywhere
}

func (suite *PaymentTestErrorsSuite) TearDownSuite() {
	suite.invoiceUpdateSubCancelFn()
}

func TestPaymentTestErrorsSuite(t *testing.T) {
	suite.Run(t, new(PaymentTestErrorsSuite))
}
