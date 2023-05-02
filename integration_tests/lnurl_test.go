package integration_tests

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/getAlby/lndhub.go/controllers"
	v2controllers "github.com/getAlby/lndhub.go/controllers_v2"
	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	tokensmw "github.com/getAlby/lndhub.go/lib/tokens"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type LnurlTestSuite struct {
	TestSuite
	service                  *service.LndhubService
	mlnd                     *MockLND
	houseToken               string
	userLogin                []ExpectedCreateUserResponseBody
	userToken                []string
	invoiceUpdateSubCancelFn context.CancelFunc
}

func (suite *LnurlTestSuite) SetupSuite() {
	mockLND := newDefaultMockLND()
	svc, err := LndHubTestServiceInit(mockLND)
	if err != nil {
		log.Fatalf("Error initializing test service: %v", err)
	}
	suite.service = svc
	suite.mlnd = mockLND
	users, tokens, err := createUsers(svc, 7)
	if err != nil {
		log.Fatalf("Error creating test users: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	suite.houseToken, _, err = svc.GenerateToken(ctx, suite.service.Config.HouseUser, mockHousePassword, "")
	if err != nil {
		log.Fatalf("Error generating house token: %v", err)
	}
	suite.invoiceUpdateSubCancelFn = cancel
	go svc.InvoiceUpdateSubscription(ctx)
	e := echo.New()

	e.HTTPErrorHandler = responses.HTTPErrorHandler
	e.Validator = &lib.CustomValidator{Validator: validator.New()}
	suite.echo = e
	assert.Equal(suite.T(), 7, len(users))
	suite.userLogin = users
	suite.userToken = tokens
	suite.echo.POST("/addinvoice", controllers.NewAddInvoiceController(suite.service).AddInvoice, tokensmw.Middleware([]byte(suite.service.Config.JWTSecret)))
	suite.echo.GET("/v2/lnurlp/:user", v2controllers.NewLnurlController(suite.service).Lnurlp)
	suite.echo.GET("/v2/invoice", v2controllers.NewInvoiceController(suite.service).Lud6Invoice)
	suite.echo.GET("/v2/balance", v2controllers.NewBalanceController(suite.service).Balance, tokensmw.Middleware([]byte(suite.service.Config.JWTSecret)))
	suite.echo.POST("/v2/payinvoice", v2controllers.NewPayInvoiceController(suite.service).PayInvoice, tokensmw.Middleware([]byte(suite.service.Config.JWTSecret)))
	suite.echo.GET("/v2/invoices/incoming", v2controllers.NewInvoiceController(suite.service).GetIncomingInvoices, tokensmw.Middleware([]byte(suite.service.Config.JWTSecret)))
	suite.echo.GET("/v2/invoices/:payment_hash", v2controllers.NewInvoiceController(suite.service).GetInvoice, tokensmw.Middleware([]byte(suite.service.Config.JWTSecret)))
}

func (suite *LnurlTestSuite) TearDownTest() {
	clearTable(suite.service, "invoices")
	clearTable(suite.service, "transaction_entries")
	fmt.Println("Tear down test success")
}
func (suite *LnurlTestSuite) TearDownSuite() {
	suite.invoiceUpdateSubCancelFn()
	clearTable(suite.service, "users")
}

func (suite *LnurlTestSuite) TestLud6InvoiceWithMetadata() {
	invoiceResponse := &v2controllers.Lud6InvoiceResponseBody{}
	rec := httptest.NewRecorder()
	const fakeAcc = "bafkreibaejvf3wyblh3s4yhbrwtxto7wpcac7zkkx36cswjzjez2cbmzvu"
	memo := "Invoicememo😀"
	//failed user
	req := httptest.NewRequest(http.MethodGet, "/v2/invoice?user="+fakeAcc, nil)
	suite.echo.ServeHTTP(rec, req)
	assert.Equal(suite.T(), http.StatusBadRequest, rec.Code)
	//single user + memo
	rec2 := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v2/invoice?user="+suite.userLogin[1].Login+"&memo="+memo, nil)
	suite.echo.ServeHTTP(rec2, req)
	assert.Equal(suite.T(), http.StatusOK, rec2.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec2.Body).Decode(invoiceResponse))
	decodedPayreq, err := suite.mlnd.DecodeBolt11(context.Background(), invoiceResponse.Payreq)
	assert.NoError(suite.T(), err)
	assert.EqualValues(suite.T(), 0, decodedPayreq.NumMsat)
	assert.EqualValues(suite.T(), memo, decodedPayreq.Description)
	//2 users + amount
	const sliceAcc1 = 0.45
	const sliceAcc2 = 0.53
	var amountmsats = 12000
	metadata := &v2controllers.PaymentMetadata{
		Source:  "",
		Authors: map[string]float64{suite.userLogin[1].Login: sliceAcc1, suite.userLogin[2].Login: sliceAcc2},
	}
	req = httptest.NewRequest(http.MethodGet, "/v2/invoice?"+metadata.URL()+"&amount="+strconv.Itoa(amountmsats), nil)
	rec3 := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec3, req)
	assert.Equal(suite.T(), http.StatusOK, rec3.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec3.Body).Decode(invoiceResponse))
	decodedPayreq, err = suite.mlnd.DecodeBolt11(context.Background(), invoiceResponse.Payreq)
	assert.NoError(suite.T(), err)
	assert.EqualValues(suite.T(), amountmsats, decodedPayreq.NumMsat)
	//3 users err
	metadata.Authors[suite.userLogin[3].Login] = 0.1
	req = httptest.NewRequest(http.MethodGet, "/v2/invoice?"+metadata.URL(), nil)
	rec4 := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec4, req)
	assert.Equal(suite.T(), http.StatusBadRequest, rec4.Code)

	//3 users + source
	metadata.Authors[suite.userLogin[3].Login] = 0.02
	metadata.Source = fakeAcc
	req = httptest.NewRequest(http.MethodGet, "/v2/invoice?"+metadata.URL(), nil)
	rec5 := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec5, req)
	assert.Equal(suite.T(), http.StatusOK, rec5.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec5.Body).Decode(invoiceResponse))
	assert.Equal(suite.T(), 0, len(invoiceResponse.Routes))
	decodedPayreq, err = suite.mlnd.DecodeBolt11(context.Background(), invoiceResponse.Payreq)
	assert.NoError(suite.T(), err)
	assert.EqualValues(suite.T(), "87e04946ff05ec28174aeb6f0e19f8295304a70f45f19bb3d915dc9aa6976587", decodedPayreq.DescriptionHash)

	//Repeated users Add up
	const remainder = 0.03
	amountmsats = 1000000
	metadata = &v2controllers.PaymentMetadata{
		Source: "",
		Authors: map[string]float64{suite.userLogin[3].Login: remainder,
			suite.userLogin[4].Login: sliceAcc1 - remainder,
			suite.userLogin[5].Login: remainder,
			suite.userLogin[6].Login: sliceAcc2 - remainder},
	}
	url := strings.Replace(metadata.URL(), suite.userLogin[3].Login, suite.userLogin[4].Login, -1)
	url = strings.Replace(url, suite.userLogin[6].Login, suite.userLogin[5].Login, -1)
	req = httptest.NewRequest(http.MethodGet, "/v2/invoice?"+url, nil)
	rec6 := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec6, req)
	assert.Equal(suite.T(), http.StatusOK, rec6.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec6.Body).Decode(invoiceResponse))
	assert.Equal(suite.T(), 0, len(invoiceResponse.Routes))
	decodedPayreq, err = suite.mlnd.DecodeBolt11(context.Background(), invoiceResponse.Payreq)
	assert.NoError(suite.T(), err)
	assert.EqualValues(suite.T(), "814261a99011a3367569953de2a1673fdb1a77b20649afb52136d49eb450304f", decodedPayreq.DescriptionHash)

	//Lead user in secondary
	rec7 := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v2/invoice?user="+suite.service.Config.HouseUser+"&memo="+memo, nil)
	suite.echo.ServeHTTP(rec7, req)
	assert.Equal(suite.T(), http.StatusBadRequest, rec7.Code)

}

func (suite *LnurlTestSuite) TestInternalSplitPayment() {
	payreqResponse := &v2controllers.Lud6InvoiceResponseBody{}
	invoiceResponse := &v2controllers.Invoice{}
	invoicesResponse := &v2controllers.GetInvoicesResponseBody{}
	payResponse := &v2controllers.PayInvoiceResponseBody{}
	rec := httptest.NewRecorder()
	const sliceAcc1 = 0.38
	const sliceAcc2 = 0.58
	const amountmsats = 150000

	metadata := &v2controllers.PaymentMetadata{
		Source:  "",
		Authors: map[string]float64{suite.userLogin[1].Login: sliceAcc1, suite.userLogin[2].Login: sliceAcc2},
	}
	req := httptest.NewRequest(http.MethodGet, "/v2/invoice?"+metadata.URL()+"&amount="+strconv.Itoa(amountmsats), nil)
	suite.echo.ServeHTTP(rec, req)
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(payreqResponse))
	decodedPayreq, err := suite.mlnd.DecodeBolt11(context.Background(), payreqResponse.Payreq)
	assert.NoError(suite.T(), err)

	// Check Lead invoice invoice exists and is not paid
	req = httptest.NewRequest(http.MethodGet, "/v2/invoices/"+decodedPayreq.PaymentHash, nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.houseToken))
	rec2 := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec2, req)
	assert.Equal(suite.T(), http.StatusOK, rec2.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec2.Body).Decode(invoiceResponse))
	assert.EqualValues(suite.T(), amountmsats/1000, invoiceResponse.Amount)
	assert.False(suite.T(), invoiceResponse.IsPaid)

	// Check split invoice exists and is not paid
	req = httptest.NewRequest(http.MethodGet, "/v2/invoices/incoming", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken[1]))
	rec3 := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec3, req)
	assert.Equal(suite.T(), http.StatusOK, rec3.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec3.Body).Decode(invoicesResponse))
	assert.False(suite.T(), invoicesResponse.Invoices[0].IsPaid)
	assert.EqualValues(suite.T(), int(sliceAcc1*amountmsats/1000), invoicesResponse.Invoices[0].Amount)

	// Check the other split invoice exists and is not paid
	req = httptest.NewRequest(http.MethodGet, "/v2/invoices/incoming", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken[2]))
	rec4 := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec4, req)
	assert.Equal(suite.T(), http.StatusOK, rec4.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec4.Body).Decode(invoicesResponse))
	assert.False(suite.T(), invoicesResponse.Invoices[0].IsPaid)
	assert.EqualValues(suite.T(), int(sliceAcc2*amountmsats/1000), invoicesResponse.Invoices[0].Amount)

	//fund payer's account
	fundInvoiceResponse := suite.createAddInvoiceReq(amountmsats*1.1, "funding account", suite.userToken[3])
	assert.NoError(suite.T(), suite.mlnd.mockPaidInvoice(fundInvoiceResponse, 0, false, nil))
	time.Sleep(100 * time.Millisecond) // give time to update the balances
	// Pay the invoice
	rec5 := httptest.NewRecorder()
	var buf bytes.Buffer
	assert.NoError(suite.T(), json.NewEncoder(&buf).Encode(&ExpectedPayInvoiceRequestBody{
		Invoice: payreqResponse.Payreq,
	}))
	req = httptest.NewRequest(http.MethodPost, "/v2/payinvoice", &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken[3]))
	suite.echo.ServeHTTP(rec5, req)
	assert.Equal(suite.T(), http.StatusOK, rec5.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec5.Body).Decode(payResponse))
	assert.NotEmpty(suite.T(), payResponse.PaymentPreimage)
	assert.EqualValues(suite.T(), amountmsats/1000, payResponse.Amount)

	// Check Lead invoice invoice exists and is paid
	req = httptest.NewRequest(http.MethodGet, "/v2/invoices/incoming", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.houseToken))
	rec6 := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec6, req)
	assert.Equal(suite.T(), http.StatusOK, rec6.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec6.Body).Decode(invoicesResponse))
	assert.EqualValues(suite.T(), amountmsats/1000, invoicesResponse.Invoices[0].Amount)
	assert.True(suite.T(), invoicesResponse.Invoices[0].IsPaid)
	balance := suite.getBalance(suite.houseToken)
	assert.EqualValues(suite.T(), (1-sliceAcc1-sliceAcc2)*amountmsats/1000, balance)

	// Check split invoice exists and is paid
	req = httptest.NewRequest(http.MethodGet, "/v2/invoices/incoming", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken[1]))
	rec7 := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec7, req)
	assert.Equal(suite.T(), http.StatusOK, rec7.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec7.Body).Decode(invoicesResponse))
	assert.True(suite.T(), invoicesResponse.Invoices[0].IsPaid)
	assert.EqualValues(suite.T(), int(sliceAcc1*amountmsats/1000), invoicesResponse.Invoices[0].Amount)
	balance = suite.getBalance(suite.userToken[1])
	assert.EqualValues(suite.T(), sliceAcc1*amountmsats/1000, balance)

	// Check the other split invoice exists and is paid
	req = httptest.NewRequest(http.MethodGet, "/v2/invoices/incoming", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken[2]))
	rec8 := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec8, req)
	assert.Equal(suite.T(), http.StatusOK, rec8.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec8.Body).Decode(invoicesResponse))
	assert.True(suite.T(), invoicesResponse.Invoices[0].IsPaid)
	assert.EqualValues(suite.T(), int(sliceAcc2*amountmsats/1000), invoicesResponse.Invoices[0].Amount)
	balance = suite.getBalance(suite.userToken[2])
	assert.EqualValues(suite.T(), sliceAcc2*amountmsats/1000, balance)
}

func (suite *LnurlTestSuite) TestExternalSplitPayment() {
	payreqResponse := &v2controllers.Lud6InvoiceResponseBody{}
	invoicesResponse := &v2controllers.GetInvoicesResponseBody{}
	rec := httptest.NewRecorder()
	const sliceAcc1 = 0.45
	const sliceAcc2 = 0.53
	const amountmsats = 200000
	const invoiceMemo = "TestExternalSplitPayment"
	metadata := &v2controllers.PaymentMetadata{
		Source:  "",
		Authors: map[string]float64{suite.userLogin[4].Login: sliceAcc1, suite.userLogin[5].Login: sliceAcc2},
	}
	req := httptest.NewRequest(http.MethodGet, "/v2/invoice?"+metadata.URL()+"&amount="+strconv.Itoa(amountmsats)+"&memo="+invoiceMemo, nil)
	suite.echo.ServeHTTP(rec, req)
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(payreqResponse))
	decodedPayreq, err := suite.mlnd.DecodeBolt11(context.Background(), payreqResponse.Payreq)
	assert.NoError(suite.T(), err)
	//Externally pay
	assert.NoError(suite.T(), suite.mlnd.mockPaidInvoice(&ExpectedAddInvoiceResponseBody{
		RHash:          decodedPayreq.PaymentHash,
		PayReq:         payreqResponse.Payreq,
		PaymentRequest: payreqResponse.Payreq,
	}, 0, false, nil))
	time.Sleep(100 * time.Millisecond) // give time to update the balances

	// Check Lead invoice invoice exists and is paid
	req = httptest.NewRequest(http.MethodGet, "/v2/invoices/incoming", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.houseToken))
	rec6 := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec6, req)
	assert.Equal(suite.T(), http.StatusOK, rec6.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec6.Body).Decode(invoicesResponse))
	assert.Len(suite.T(), invoicesResponse.Invoices, 1)
	assert.EqualValues(suite.T(), amountmsats/1000, invoicesResponse.Invoices[0].Amount)

	assert.True(suite.T(), invoicesResponse.Invoices[0].IsPaid)
	balance := suite.getBalance(suite.houseToken)
	assert.EqualValues(suite.T(), (1-sliceAcc1-sliceAcc2)*amountmsats/1000, balance)

	// Check split invoice exists and is paid
	req = httptest.NewRequest(http.MethodGet, "/v2/invoices/incoming", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken[4]))
	rec7 := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec7, req)
	assert.Equal(suite.T(), http.StatusOK, rec7.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec7.Body).Decode(invoicesResponse))
	assert.Len(suite.T(), invoicesResponse.Invoices, 1)
	assert.EqualValues(suite.T(), int(sliceAcc1*amountmsats/1000), invoicesResponse.Invoices[0].Amount)
	balance = suite.getBalance(suite.userToken[4])
	assert.EqualValues(suite.T(), sliceAcc1*amountmsats/1000, balance)

	// Check the other split invoice exists and is paid
	req = httptest.NewRequest(http.MethodGet, "/v2/invoices/incoming", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken[5]))
	rec8 := httptest.NewRecorder()

	suite.echo.ServeHTTP(rec8, req)
	assert.Equal(suite.T(), http.StatusOK, rec8.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec8.Body).Decode(invoicesResponse))
	assert.Len(suite.T(), invoicesResponse.Invoices, 1)
	assert.EqualValues(suite.T(), int(sliceAcc2*amountmsats/1000), invoicesResponse.Invoices[0].Amount)
	balance = suite.getBalance(suite.userToken[5])
	assert.EqualValues(suite.T(), sliceAcc2*amountmsats/1000, balance)
}
func (suite *LnurlTestSuite) TestGetLnurlInvoiceZeroAmt() {
	// call the lnurl endpoint
	req := httptest.NewRequest(http.MethodGet, "/v2/lnurlp/"+suite.userLogin[1].Login, nil)
	rec := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec, req)
	lnurlResponse := &ExpectedLnurlpResponseBody{}
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(lnurlResponse))
	assert.EqualValues(suite.T(), lnurlResponse.Tag, v2controllers.LNURLP_TAG)
	assert.EqualValues(suite.T(), lnurlResponse.CommentAllowed, v2controllers.LNURLP_COMMENT_SIZE)
	assert.EqualValues(suite.T(), lnurlResponse.MinSendable, 1000)
	assert.EqualValues(suite.T(), lnurlResponse.MaxSendable, suite.service.Config.MaxReceiveAmount*1000)
	urlStart := strings.Index(lnurlResponse.Callback, "/v2/invoice")
	assert.Greater(suite.T(), urlStart, 0)

	// call callback
	req = httptest.NewRequest(http.MethodGet, lnurlResponse.Callback[urlStart:], nil)
	rec2 := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec2, req)
	invoiceResponse := &v2controllers.Lud6InvoiceResponseBody{}
	assert.Equal(suite.T(), http.StatusOK, rec2.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec2.Body).Decode(invoiceResponse))
	assert.Equal(suite.T(), 0, len(invoiceResponse.Routes))
	decodedPayreq, err := suite.mlnd.DecodeBolt11(context.Background(), invoiceResponse.Payreq)
	assert.NoError(suite.T(), err)
	assert.EqualValues(suite.T(), 0, decodedPayreq.NumMsat)
	assert.NoError(suite.T(), err)
	descriptionHash := sha256.Sum256([]byte(lnurlResponse.Metadata))
	expectedPH := hex.EncodeToString(descriptionHash[:])
	assert.EqualValues(suite.T(), expectedPH, decodedPayreq.DescriptionHash)
}

func (suite *LnurlTestSuite) TestGetLnurlInvoiceCustomAmt() {
	// call the lnurl endpoint
	const payreq_type = "payRequest"
	const amt_sats = int64(1245)
	req := httptest.NewRequest(http.MethodGet, "/v2/lnurlp/"+suite.userLogin[1].Login+"?amt="+strconv.FormatInt(amt_sats, 10), nil)
	rec := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec, req)
	lnurlResponse := &ExpectedLnurlpResponseBody{}
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(lnurlResponse))
	assert.Equal(suite.T(), lnurlResponse.Tag, payreq_type)
	assert.EqualValues(suite.T(), lnurlResponse.MinSendable, amt_sats*1000)
	assert.EqualValues(suite.T(), lnurlResponse.MaxSendable, amt_sats*1000)
	urlStart := strings.Index(lnurlResponse.Callback, "/v2/invoice")
	assert.Greater(suite.T(), urlStart, 0)

	// call callback
	const amt_msats = amt_sats * 1000
	req = httptest.NewRequest(http.MethodGet, lnurlResponse.Callback[urlStart:], nil)
	rec2 := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec2, req)
	invoiceResponse := &v2controllers.Lud6InvoiceResponseBody{}
	assert.Equal(suite.T(), http.StatusOK, rec2.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec2.Body).Decode(invoiceResponse))
	assert.Equal(suite.T(), 0, len(invoiceResponse.Routes))
	decodedPayreq, err := suite.mlnd.DecodeBolt11(context.Background(), invoiceResponse.Payreq)
	assert.NoError(suite.T(), err)
	assert.EqualValues(suite.T(), amt_msats, decodedPayreq.NumMsat)
}

func TestLnurlSuite(t *testing.T) {
	suite.Run(t, new(LnurlTestSuite))
}

func (suite *LnurlTestSuite) getBalance(token string) int64 {
	balance := &v2controllers.BalanceResponse{}
	req := httptest.NewRequest(http.MethodGet, "/v2/balance", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	rec := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec, req)
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(balance))
	return balance.Balance
}
