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
	aliceLogin               ExpectedCreateUserResponseBody
	aliceToken               string
	bobLogin                 ExpectedCreateUserResponseBody
	bobToken                 string
	invoiceUpdateSubCancelFn context.CancelFunc
}

func (suite *PaymentTestSuite) SetupSuite() {
	lndClient, err := lnd.NewLNDclient(lnd.LNDoptions{
		Address:     lnd3RegtestAddress,
		MacaroonHex: lnd3RegtestMacaroonHex,
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
	suite.echo.GET("/gettxs", controllers.NewGetTXSController(suite.service).GetTXS)
	suite.echo.POST("/keysend", controllers.NewKeySendController(suite.service).KeySend)
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
	// currently fee is 0 for internal payments
	fee := 0
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
	payResponse := suite.createPayInvoiceReq(&ExpectedPayInvoiceRequestBody{
		Invoice: bobInvoice.PayReq,
	}, suite.aliceToken)
	assert.NotEmpty(suite.T(), payResponse.PaymentPreimage)

	aliceId := getUserIdFromToken(suite.aliceToken)
	bobId := getUserIdFromToken(suite.bobToken)

	//try to pay Bob more than we currently have
	//create invoice for bob
	tooMuch := suite.createAddInvoiceReq(10000, "integration test internal payment bob", suite.bobToken)
	//pay bob from alice
	errorResp := suite.createPayInvoiceReqError(tooMuch.PayReq, suite.aliceToken)
	assert.Equal(suite.T(), responses.NotEnoughBalanceError.Code, errorResp.Code)

	transactonEntriesAlice, _ := suite.service.TransactionEntriesFor(context.Background(), aliceId)
	aliceBalance, _ := suite.service.CurrentUserBalance(context.Background(), aliceId)
	assert.Equal(suite.T(), 3, len(transactonEntriesAlice))
	assert.Equal(suite.T(), int64(aliceFundingSats), transactonEntriesAlice[0].Amount)
	assert.Equal(suite.T(), int64(bobSatRequested), transactonEntriesAlice[1].Amount)
	assert.Equal(suite.T(), int64(fee), transactonEntriesAlice[2].Amount)
	assert.Equal(suite.T(), transactonEntriesAlice[1].ID, transactonEntriesAlice[2].ParentID)
	assert.Equal(suite.T(), int64(aliceFundingSats-bobSatRequested-fee), aliceBalance)

	bobBalance, _ := suite.service.CurrentUserBalance(context.Background(), bobId)
	transactionEntriesBob, _ := suite.service.TransactionEntriesFor(context.Background(), bobId)
	assert.Equal(suite.T(), 1, len(transactionEntriesBob))
	assert.Equal(suite.T(), int64(bobSatRequested), transactionEntriesBob[0].Amount)
	assert.Equal(suite.T(), int64(bobSatRequested), bobBalance)
}

func (suite *PaymentTestSuite) TestInternalPaymentFail() {
	aliceFundingSats := 1000
	bobSatRequested := 500
	// currently fee is 0 for internal payments
	fee := 0
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
	payResponse := suite.createPayInvoiceReq(&ExpectedPayInvoiceRequestBody{
		Invoice: bobInvoice.PayReq,
	}, suite.aliceToken)
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

	// check if there are 5 transaction entries, with reversed credit and debit account ids for last 2
	assert.Equal(suite.T(), 5, len(transactonEntries))
	assert.Equal(suite.T(), int64(aliceFundingSats), transactonEntries[0].Amount)
	assert.Equal(suite.T(), int64(bobSatRequested), transactonEntries[1].Amount)
	assert.Equal(suite.T(), int64(fee), transactonEntries[2].Amount)
	assert.Equal(suite.T(), transactonEntries[3].CreditAccountID, transactonEntries[4].DebitAccountID)
	assert.Equal(suite.T(), transactonEntries[3].DebitAccountID, transactonEntries[4].CreditAccountID)
	assert.Equal(suite.T(), transactonEntries[3].Amount, int64(bobSatRequested))
	assert.Equal(suite.T(), transactonEntries[4].Amount, int64(bobSatRequested))
	// assert that balance was reduced only once
	assert.Equal(suite.T(), int64(aliceFundingSats)-int64(bobSatRequested+fee), int64(aliceBalance))
}
func (suite *PaymentTestSuite) TestInternalPaymentKeysend() {
	aliceFundingSats := 1000
	bobAmt := 100
	memo := "integration test internal keysend from alice"
	//fund alice account
	invoiceResponse := suite.createAddInvoiceReq(aliceFundingSats, "integration test internal keysend alice", suite.aliceToken)
	sendPaymentRequest := lnrpc.SendRequest{
		PaymentRequest: invoiceResponse.PayReq,
		FeeLimit:       nil,
	}
	_, err := suite.fundingClient.SendPaymentSync(context.Background(), &sendPaymentRequest)
	assert.NoError(suite.T(), err)

	//wait a bit for the callback event to hit
	time.Sleep(100 * time.Millisecond)

	//check bob's balance before payment
	bobId := getUserIdFromToken(suite.bobToken)
	previousBobBalance, _ := suite.service.CurrentUserBalance(context.Background(), bobId)

	//pay bob from alice using a keysend payment
	rec := httptest.NewRecorder()
	var buf bytes.Buffer
	assert.NoError(suite.T(), json.NewEncoder(&buf).Encode(ExpectedKeySendRequestBody{
		Amount:      int64(bobAmt),
		Destination: suite.service.IdentityPubkey,
		Memo:        memo,
		//add memo as WHATSAT_MESSAGE custom record
		CustomRecords: map[string]string{fmt.Sprint(service.TLV_WHATSAT_MESSAGE): memo,
			fmt.Sprint(service.TLV_WALLET_ID): suite.bobLogin.Login},
	}))
	req := httptest.NewRequest(http.MethodPost, "/keysend", &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.aliceToken))
	suite.echo.ServeHTTP(rec, req)
	keySendResponse := &ExpectedKeySendResponseBody{}
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(keySendResponse))

	//check bob's balance after payment
	bobBalance, _ := suite.service.CurrentUserBalance(context.Background(), bobId)
	assert.Equal(suite.T(), int64(bobAmt)+previousBobBalance, bobBalance)
	//check bob's invoices for whatsat message
	invoicesBob, _ := suite.service.InvoicesFor(context.Background(), bobId, common.InvoiceTypeIncoming)
	foundKeySend := false
	for _, invoice := range invoicesBob {
		if invoice.Keysend {
			foundKeySend = true
			assert.Equal(suite.T(), memo, string(invoice.DestinationCustomRecords[service.TLV_WHATSAT_MESSAGE]))
		}
	}
	assert.True(suite.T(), foundKeySend)
}

func TestInternalPaymentTestSuite(t *testing.T) {
	suite.Run(t, new(PaymentTestSuite))
}
