package integration_tests

import (
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

	v2controllers "github.com/getAlby/lndhub.go/controllers_v2"
	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type LnurlTestSuite struct {
	TestSuite
	service   *service.LndhubService
	mlnd      *MockLND
	userLogin []ExpectedCreateUserResponseBody
}

func (suite *LnurlTestSuite) SetupSuite() {
	mockLND := newDefaultMockLND()
	svc, err := LndHubTestServiceInit(mockLND)
	if err != nil {
		log.Fatalf("Error initializing test service: %v", err)
	}
	suite.service = svc
	suite.mlnd = mockLND
	users, _, err := createUsers(svc, 4)
	if err != nil {
		log.Fatalf("Error creating test users: %v", err)
	}

	e := echo.New()

	e.HTTPErrorHandler = responses.HTTPErrorHandler
	e.Validator = &lib.CustomValidator{Validator: validator.New()}
	suite.echo = e
	assert.Equal(suite.T(), 4, len(users))
	suite.userLogin = users
	suite.echo.GET("/v2/lnurlp/:user", v2controllers.NewLnurlController(suite.service).Lnurlp)
	suite.echo.GET("/v2/invoice", v2controllers.NewInvoiceController(suite.service).Lud6Invoice)
}

func (suite *LnurlTestSuite) TearDownSuite() {
	err := clearTable(suite.service, "users")
	if err != nil {
		fmt.Printf("Tear down suite error %v\n", err.Error())
		return
	}
	fmt.Println("Tear down suite success")
}

// TODO: launch a gorutine that listen for payed invoices and then settle the splits associated with them (call sendInternalPayment)
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
