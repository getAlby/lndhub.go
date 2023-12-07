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

type PaymentTestSuite struct {
	TestSuite
	mlnd                     *MockLND
	externalLND              *MockLND
	service                  *service.LndhubService
	aliceLogin               ExpectedCreateUserResponseBody
	aliceToken               string
	bobLogin                 ExpectedCreateUserResponseBody
	bobToken                 string
	invoiceUpdateSubCancelFn context.CancelFunc
}

func (suite *PaymentTestSuite) SetupSuite() {
	mlnd := newDefaultMockLND()
	suite.mlnd = mlnd
	externalLND, err := NewMockLND("1234567890abcdefabcd", 0, make(chan (*lnrpc.Invoice)))
	if err != nil {
		log.Fatalf("Error initializing test service: %v", err)
	}
	suite.externalLND = externalLND
	svc, err := LndHubTestServiceInit(mlnd)
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
	suite.echo.GET("/checkpayment/:payment_hash", controllers.NewCheckPaymentController(suite.service).CheckPayment)
}

func (suite *PaymentTestSuite) TearDownSuite() {
	suite.invoiceUpdateSubCancelFn()
}

func (suite *PaymentTestSuite) TearDownTest() {
	clearTable(suite.service, "transaction_entries")
	clearTable(suite.service, "invoices")
}

func (suite *PaymentTestSuite) TestPaymentFeeReserve() {
	//set config to check for fee reserve
	suite.service.Config.FeeReserve = true
	aliceFundingSats := 1000
	//fund alice account
	invoiceResponse := suite.createAddInvoiceReq(aliceFundingSats, "integration test internal payment alice", suite.aliceToken)
	err := suite.mlnd.mockPaidInvoice(invoiceResponse, 0, false, nil)
	assert.NoError(suite.T(), err)

	//wait a bit for the payment to be processed
	time.Sleep(10 * time.Millisecond)

	//try to make external payment
	//which should fail
	//create external invoice
	externalSatRequested := 1000
	externalInvoice := lnrpc.Invoice{
		Memo:  "integration tests: external pay from user",
		Value: int64(externalSatRequested),
	}
	invoice, err := suite.externalLND.AddInvoice(context.Background(), &externalInvoice)
	assert.NoError(suite.T(), err)
	//pay external invoice
	rec := httptest.NewRecorder()
	var buf bytes.Buffer
	assert.NoError(suite.T(), json.NewEncoder(&buf).Encode(&ExpectedPayInvoiceRequestBody{
		Invoice: invoice.PaymentRequest,
	}))
	req := httptest.NewRequest(http.MethodPost, "/payinvoice", &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.aliceToken))
	suite.echo.ServeHTTP(rec, req)
	//should fail because fee reserve is active
	assert.Equal(suite.T(), http.StatusBadRequest, rec.Code)
	//try to make internal payment, which should work
	bobSatRequested := 1000
	//create invoice for bob
	bobInvoice := suite.createAddInvoiceReq(bobSatRequested, "integration test internal payment bob", suite.bobToken)
	//pay bob from alice, this should work because it's internal
	payResponse := suite.createPayInvoiceReq(&ExpectedPayInvoiceRequestBody{
		Invoice: bobInvoice.PayReq,
	}, suite.aliceToken)
	assert.NotEmpty(suite.T(), payResponse.PaymentPreimage)
	//reset fee reserve so it's not used in other tests
	suite.service.Config.FeeReserve = false
}
func (suite *PaymentTestSuite) TestIncomingExceededChecks() {
	//this will cause the payment to fail as the account was already funded
	//with 1000 sats
	aliceFundingSats := 1000
	//fund alice account
	invoiceResponse := suite.createAddInvoiceReq(aliceFundingSats, "integration test internal payment alice", suite.aliceToken)
	err := suite.mlnd.mockPaidInvoice(invoiceResponse, 0, false, nil)
	assert.NoError(suite.T(), err)

	//wait a bit for the payment to be processed
	time.Sleep(10 * time.Millisecond)
	var buf bytes.Buffer
	suite.service.Config.MaxReceiveAmount = 21
	rec := httptest.NewRecorder()
	assert.NoError(suite.T(), json.NewEncoder(&buf).Encode(&ExpectedAddInvoiceRequestBody{
		Amount: aliceFundingSats,
		Memo:   "memo",
	}))
	req := httptest.NewRequest(http.MethodPost, "/addinvoice", &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.aliceToken))
	suite.echo.ServeHTTP(rec, req)
	//should fail because max receive amount check
	assert.Equal(suite.T(), http.StatusBadRequest, rec.Code)
	resp := &responses.ErrorResponse{}
	err = json.NewDecoder(rec.Body).Decode(resp)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), responses.ReceiveExceededError.Message, resp.Message)

	// remove volume and receive config and check if it works
	suite.service.Config.MaxReceiveAmount = 0
	invoiceResponse = suite.createAddInvoiceReq(aliceFundingSats, "integration test internal payment alice", suite.aliceToken)
	err = suite.mlnd.mockPaidInvoice(invoiceResponse, 0, false, nil)
	assert.NoError(suite.T(), err)

	// add max account
	suite.service.Config.MaxAccountBalance = 500
	assert.NoError(suite.T(), json.NewEncoder(&buf).Encode(&ExpectedAddInvoiceRequestBody{
		Amount: aliceFundingSats,
		Memo:   "memo",
	}))
	req = httptest.NewRequest(http.MethodPost, "/addinvoice", &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.aliceToken))
	suite.echo.ServeHTTP(rec, req)
	//should fail because max balance check
	assert.Equal(suite.T(), http.StatusBadRequest, rec.Code)
	resp = &responses.ErrorResponse{}
	err = json.NewDecoder(rec.Body).Decode(resp)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), responses.BalanceExceededError.Message, resp.Message)

	//change the config back and add sats, it should work now
	suite.service.Config.MaxAccountBalance = 0
	invoiceResponse = suite.createAddInvoiceReq(aliceFundingSats, "integration test internal payment alice", suite.aliceToken)
	err = suite.mlnd.mockPaidInvoice(invoiceResponse, 0, false, nil)
	assert.NoError(suite.T(), err)

	// add max receive volume
	suite.service.Config.MaxReceiveVolume = 1999 // because the volume till here is 1000+500+500
	suite.service.Config.MaxVolumePeriod = 2592000
	assert.NoError(suite.T(), json.NewEncoder(&buf).Encode(&ExpectedAddInvoiceRequestBody{
		Amount: aliceFundingSats,
		Memo:   "memo",
	}))
	req = httptest.NewRequest(http.MethodPost, "/addinvoice", &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.aliceToken))
	suite.echo.ServeHTTP(rec, req)
	//should fail because max volume check
	assert.Equal(suite.T(), http.StatusBadRequest, rec.Code)
	resp = &responses.ErrorResponse{}
	err = json.NewDecoder(rec.Body).Decode(resp)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), responses.TooMuchVolumeError.Message, resp.Message)

	//change the config back, it should work now
	suite.service.Config.MaxReceiveVolume = 0
	suite.service.Config.MaxVolumePeriod = 0
	invoiceResponse = suite.createAddInvoiceReq(aliceFundingSats, "integration test internal payment alice", suite.aliceToken)
	err = suite.mlnd.mockPaidInvoice(invoiceResponse, 0, false, nil)
	assert.NoError(suite.T(), err)
}

func (suite *PaymentTestSuite) TestOutgoingExceededChecks() {
	//this will cause the payment to fail as the account was already funded
	//with 1000 sats
	suite.service.Config.MaxSendAmount = 100
	aliceFundingSats := 1000
	//fund alice account
	invoiceResponse := suite.createAddInvoiceReq(aliceFundingSats, "integration test internal payment alice", suite.aliceToken)
	err := suite.mlnd.mockPaidInvoice(invoiceResponse, 0, false, nil)
	assert.NoError(suite.T(), err)

	//wait a bit for the payment to be processed
	time.Sleep(10 * time.Millisecond)

	//try to make external payment
	//which should fail
	//create external invoice
	externalSatRequested := 400
	externalInvoice := lnrpc.Invoice{
		Memo:  "integration tests: external pay from user",
		Value: int64(externalSatRequested),
	}
	invoice, err := suite.externalLND.AddInvoice(context.Background(), &externalInvoice)
	assert.NoError(suite.T(), err)
	//pay external invoice
	rec := httptest.NewRecorder()
	var buf bytes.Buffer
	assert.NoError(suite.T(), json.NewEncoder(&buf).Encode(&ExpectedPayInvoiceRequestBody{
		Invoice: invoice.PaymentRequest,
	}))
	req := httptest.NewRequest(http.MethodPost, "/payinvoice", &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.aliceToken))
	suite.echo.ServeHTTP(rec, req)

	//should fail because max send check
	assert.Equal(suite.T(), http.StatusBadRequest, rec.Code)
	resp := &responses.ErrorResponse{}
	err = json.NewDecoder(rec.Body).Decode(resp)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), responses.SendExceededError.Message, resp.Message)

	suite.service.Config.MaxSendAmount = 2000
	//should work now
	rec = httptest.NewRecorder()
	invoice, err = suite.externalLND.AddInvoice(context.Background(), &externalInvoice)
	assert.NoError(suite.T(), err)
	assert.NoError(suite.T(), json.NewEncoder(&buf).Encode(&ExpectedPayInvoiceRequestBody{
		Invoice: invoice.PaymentRequest,
	}))
	suite.echo.ServeHTTP(rec, req)
	assert.Equal(suite.T(), http.StatusOK, rec.Code)

	suite.service.Config.MaxSendVolume = 100
	suite.service.Config.MaxVolumePeriod = 2592000
	//volume 
	invoice, err = suite.externalLND.AddInvoice(context.Background(), &externalInvoice)
	assert.NoError(suite.T(), err)
	//pay external invoice
	rec = httptest.NewRecorder()
	assert.NoError(suite.T(), json.NewEncoder(&buf).Encode(&ExpectedPayInvoiceRequestBody{
		Invoice: invoice.PaymentRequest,
	}))
	req = httptest.NewRequest(http.MethodPost, "/payinvoice", &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.aliceToken))
	suite.echo.ServeHTTP(rec, req)

	//should fail because maximum volume check
	assert.Equal(suite.T(), http.StatusBadRequest, rec.Code)
	resp = &responses.ErrorResponse{}
	err = json.NewDecoder(rec.Body).Decode(resp)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), responses.TooMuchVolumeError.Message, resp.Message)

	//change the config back
	suite.service.Config.MaxSendAmount = 0
	suite.service.Config.MaxSendVolume = 0
	suite.service.Config.MaxVolumePeriod = 0
}

func (suite *PaymentTestSuite) TestInternalPayment() {
	aliceFundingSats := 1000
	bobSatRequested := 500
	// currently fee is 0 for internal payments
	fee := 0
	//fund alice account
	invoiceResponse := suite.createAddInvoiceReq(aliceFundingSats, "integration test internal payment alice", suite.aliceToken)
	err := suite.mlnd.mockPaidInvoice(invoiceResponse, 0, false, nil)
	assert.NoError(suite.T(), err)

	//wait a bit for the payment to be processed
	time.Sleep(10 * time.Millisecond)

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

	//make sure that the "tooMuch" invoice isn't there
	req := httptest.NewRequest(http.MethodGet, "/checkpayment/"+tooMuch.RHash, nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.aliceToken))
	rec := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec, req)
	assert.Equal(suite.T(), http.StatusBadRequest, rec.Code)

	transactionEntriesAlice, _ := suite.service.TransactionEntriesFor(context.Background(), aliceId)
	aliceBalance, _ := suite.service.CurrentUserBalance(context.Background(), aliceId)
	assert.Equal(suite.T(), 2, len(transactionEntriesAlice))
	assert.Equal(suite.T(), int64(aliceFundingSats), transactionEntriesAlice[0].Amount)
	assert.Equal(suite.T(), int64(bobSatRequested), transactionEntriesAlice[1].Amount)
	assert.Equal(suite.T(), int64(aliceFundingSats-bobSatRequested-fee), aliceBalance)

	bobBalance, _ := suite.service.CurrentUserBalance(context.Background(), bobId)
	transactionEntriesBob, _ := suite.service.TransactionEntriesFor(context.Background(), bobId)
	assert.Equal(suite.T(), 1, len(transactionEntriesBob))
	assert.Equal(suite.T(), int64(bobSatRequested), transactionEntriesBob[0].Amount)
	assert.Equal(suite.T(), int64(bobSatRequested), bobBalance)

	//generate 0 amount invoice
	zeroAmt := suite.createAddInvoiceReq(0, "integration test internal payment bob 0 amount", suite.bobToken)
	toPayForZeroAmt := 10
	rec = httptest.NewRecorder()
	var buf bytes.Buffer
	assert.NoError(suite.T(), json.NewEncoder(&buf).Encode(&ExpectedPayInvoiceRequestBody{
		Invoice: zeroAmt.PayReq,
		Amount:  toPayForZeroAmt,
	}))
	req = httptest.NewRequest(http.MethodPost, "/payinvoice", &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.aliceToken))
	suite.echo.ServeHTTP(rec, req)

	payInvoiceResponse := &ExpectedPayInvoiceResponseBody{}
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(payInvoiceResponse))
	//assert bob was credited the correct amount
	bobBalance, _ = suite.service.CurrentUserBalance(context.Background(), bobId)
	assert.Equal(suite.T(), int64(bobSatRequested+toPayForZeroAmt), bobBalance)
}

func (suite *PaymentTestSuite) TestInternalPaymentFail() {
	aliceFundingSats := 1000
	bobSatRequested := 500
	// currently fee is 0 for internal payments
	fee := 0
	invoiceResponse := suite.createAddInvoiceReq(aliceFundingSats, "integration test internal payment alice", suite.aliceToken)

	err := suite.mlnd.mockPaidInvoice(invoiceResponse, 0, false, nil)
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
	transactionEntries, err := suite.service.TransactionEntriesFor(context.Background(), userId)
	if err != nil {
		fmt.Printf("Error when getting transaction entries %v\n", err.Error())
	}

	aliceBalance, err := suite.service.CurrentUserBalance(context.Background(), userId)
	if err != nil {
		fmt.Printf("Error when getting balance %v\n", err.Error())
	}

	// check if there are 4 transaction entries, with reversed credit and debit account ids for last 2
	assert.Equal(suite.T(), 4, len(transactionEntries))
	assert.Equal(suite.T(), int64(aliceFundingSats), transactionEntries[0].Amount)
	assert.Equal(suite.T(), int64(bobSatRequested), transactionEntries[1].Amount)
	assert.Equal(suite.T(), transactionEntries[2].CreditAccountID, transactionEntries[3].DebitAccountID)
	assert.Equal(suite.T(), transactionEntries[2].DebitAccountID, transactionEntries[3].CreditAccountID)
	assert.Equal(suite.T(), transactionEntries[2].Amount, int64(bobSatRequested))
	assert.Equal(suite.T(), transactionEntries[3].Amount, int64(bobSatRequested))
	// assert that balance was reduced only once
	assert.Equal(suite.T(), int64(aliceFundingSats)-int64(bobSatRequested+fee), int64(aliceBalance))
}
func (suite *PaymentTestSuite) TestInternalPaymentKeysend() {
	aliceFundingSats := 1000
	bobAmt := 100
	memo := "integration test internal keysend from alice"
	invoiceResponse := suite.createAddInvoiceReq(aliceFundingSats, "integration test internal keysend alice", suite.aliceToken)
	err := suite.mlnd.mockPaidInvoice(invoiceResponse, 0, false, nil)
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
		Destination: suite.service.LndClient.GetMainPubkey(),
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
