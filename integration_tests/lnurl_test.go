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
	users, tokens, err := createUsers(svc, 4)
	if err != nil {
		log.Fatalf("Error creating test users: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	suite.invoiceUpdateSubCancelFn = cancel
	go svc.InvoiceUpdateSubscription(ctx)
	e := echo.New()

	e.HTTPErrorHandler = responses.HTTPErrorHandler
	e.Validator = &lib.CustomValidator{Validator: validator.New()}
	suite.echo = e
	assert.Equal(suite.T(), 4, len(users))
	suite.userLogin = users
	suite.userToken = tokens
	suite.echo.POST("/addinvoice", controllers.NewAddInvoiceController(suite.service).AddInvoice, tokensmw.Middleware([]byte(suite.service.Config.JWTSecret)))
	suite.echo.GET("/v2/lnurlp/:user", v2controllers.NewLnurlController(suite.service).Lnurlp)
	suite.echo.GET("/v2/invoice", v2controllers.NewInvoiceController(suite.service).Lud6Invoice)
	suite.echo.POST("/v2/payinvoice", v2controllers.NewPayInvoiceController(suite.service).PayInvoice, tokensmw.Middleware([]byte(suite.service.Config.JWTSecret)))
	suite.echo.GET("/v2/invoices/:payment_hash", v2controllers.NewInvoiceController(suite.service).GetInvoice, tokensmw.Middleware([]byte(suite.service.Config.JWTSecret)))
}

func (suite *LnurlTestSuite) TearDownSuite() {
	suite.invoiceUpdateSubCancelFn()
	err := clearTable(suite.service, "users")
	if err != nil {
		fmt.Printf("Tear down suite error %v\n", err.Error())
		return
	}
	fmt.Println("Tear down suite success")
}

// TODO: launch a goroutine that listen for paid invoices and then settle the splits associated with them (call sendInternalPayment)
func (suite *LnurlTestSuite) TestLud6InvoiceWithMetadata() {
	invoiceResponse := &Lud6InvoiceResponseBody{}
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
	const amountmsats = 12000
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

	//Lead user in secondary
	rec6 := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v2/invoice?user="+suite.userLogin[0].Login+"&memo="+memo, nil)
	suite.echo.ServeHTTP(rec6, req)
	assert.Equal(suite.T(), http.StatusBadRequest, rec6.Code)
}

func (suite *LnurlTestSuite) TestSettleSplitPayment() {
	payreqResponse := &Lud6InvoiceResponseBody{}
	invoiceResponse := &v2controllers.Invoice{}
	payResponse := &ExpectedPayInvoiceResponseBody{}
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
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken[0]))
	rec2 := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec2, req)
	assert.Equal(suite.T(), http.StatusOK, rec2.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec2.Body).Decode(invoiceResponse))
	assert.EqualValues(suite.T(), amountmsats/1000, invoiceResponse.Amount)
	assert.False(suite.T(), invoiceResponse.IsPaid)

	// Check split invoice exists and is not paid
	req = httptest.NewRequest(http.MethodGet, "/v2/invoices/"+decodedPayreq.PaymentHash, nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken[1]))
	rec3 := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec3, req)
	assert.Equal(suite.T(), http.StatusOK, rec3.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec3.Body).Decode(invoiceResponse))
	assert.False(suite.T(), invoiceResponse.IsPaid)
	assert.EqualValues(suite.T(), int(sliceAcc1*amountmsats/1000), invoiceResponse.Amount)

	// Check the other split invoice exists and is not paid
	req = httptest.NewRequest(http.MethodGet, "/v2/invoices/"+decodedPayreq.PaymentHash, nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken[2]))
	rec4 := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec4, req)
	assert.Equal(suite.T(), http.StatusOK, rec4.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec4.Body).Decode(invoiceResponse))
	assert.False(suite.T(), invoiceResponse.IsPaid)
	assert.EqualValues(suite.T(), int(sliceAcc2*amountmsats/1000), invoiceResponse.Amount)

	//fund payer's account
	fundInvoiceResponse := suite.createAddInvoiceReq(amountmsats*1.1, "funding account", suite.userToken[3])
	assert.NoError(suite.T(), suite.mlnd.mockPaidInvoice(fundInvoiceResponse, 0, false, nil))
	// Pay the invoice
	rec5 := httptest.NewRecorder()
	var buf bytes.Buffer
	assert.NoError(suite.T(), json.NewEncoder(&buf).Encode(&ExpectedPayInvoiceRequestBody{
		Invoice: payreqResponse.Payreq,
	}))
	req = httptest.NewRequest(http.MethodPost, "/v2/payinvoice", &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken[3]))
	time.Sleep(100 * time.Millisecond) // give time to update the balances
	suite.echo.ServeHTTP(rec5, req)
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(payResponse))
	assert.NotEmpty(suite.T(), payResponse.PaymentPreimage)
	assert.EqualValues(suite.T(), amountmsats/1000, payResponse.Amount)

	// Check Lead invoice invoice exists and is paid
	req = httptest.NewRequest(http.MethodGet, "/v2/invoices/"+decodedPayreq.PaymentHash, nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken[0]))
	rec6 := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec6, req)
	assert.Equal(suite.T(), http.StatusOK, rec6.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec6.Body).Decode(invoiceResponse))
	assert.EqualValues(suite.T(), amountmsats/1000, invoiceResponse.Amount)
	assert.True(suite.T(), invoiceResponse.IsPaid)

	// Check split invoice exists and is paid
	req = httptest.NewRequest(http.MethodGet, "/v2/invoices/"+decodedPayreq.PaymentHash, nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken[1]))
	rec7 := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec7, req)
	assert.Equal(suite.T(), http.StatusOK, rec7.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec7.Body).Decode(invoiceResponse))
	assert.True(suite.T(), invoiceResponse.IsPaid)
	assert.EqualValues(suite.T(), int(1*amountmsats/1000), invoiceResponse.Amount)

	// Check the other split invoice exists and is paid
	req = httptest.NewRequest(http.MethodGet, "/v2/invoices/"+decodedPayreq.PaymentHash, nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken[2]))
	rec8 := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec8, req)
	assert.Equal(suite.T(), http.StatusOK, rec8.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec8.Body).Decode(invoiceResponse))
	assert.False(suite.T(), invoiceResponse.IsPaid)
	assert.EqualValues(suite.T(), int(sliceAcc2*amountmsats/1000), invoiceResponse.Amount)
}
func (suite *LnurlTestSuite) TestGetLnurlInvoiceZeroAmt() {
	// call the lnurl endpoint
	req := httptest.NewRequest(http.MethodGet, "/v2/lnurlp/"+suite.userLogin[0].Login, nil)
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
	invoiceResponse := &Lud6InvoiceResponseBody{}
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
	req := httptest.NewRequest(http.MethodGet, "/v2/lnurlp/"+suite.userLogin[0].Login+"?amt="+strconv.FormatInt(amt_sats, 10), nil)
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
	invoiceResponse := &Lud6InvoiceResponseBody{}
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
