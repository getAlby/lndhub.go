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
	userLogin ExpectedCreateUserResponseBody
}

func (suite *LnurlTestSuite) SetupSuite() {
	mockLND := newDefaultMockLND()
	svc, err := LndHubTestServiceInit(mockLND)
	if err != nil {
		log.Fatalf("Error initializing test service: %v", err)
	}
	suite.service = svc
	suite.mlnd = mockLND
	users, _, err := createUsers(svc, 1)
	if err != nil {
		log.Fatalf("Error creating test users: %v", err)
	}

	e := echo.New()

	e.HTTPErrorHandler = responses.HTTPErrorHandler
	e.Validator = &lib.CustomValidator{Validator: validator.New()}
	suite.echo = e
	assert.Equal(suite.T(), 1, len(users))
	suite.userLogin = users[0]
	suite.echo.GET("/v2/lnurlp/:user", v2controllers.NewLnurlController(suite.service).Lnurlp)
	suite.echo.GET("/v2/invoice/:user", v2controllers.NewInvoiceController(suite.service).Invoice)
}

func (suite *LnurlTestSuite) TearDownSuite() {
	err := clearTable(suite.service, "users")
	if err != nil {
		fmt.Printf("Tear down suite error %v\n", err.Error())
		return
	}
	fmt.Println("Tear down suite success")
}

func (suite *LnurlTestSuite) TestGetLnurlInvoiceZeroAmt() {

	// call the lnurl endpoint
	const payreq_type = "payRequest"
	req := httptest.NewRequest(http.MethodGet, "/v2/lnurlp/"+suite.userLogin.Nickname, nil)
	rec := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec, req)
	lnurlResponse := &ExpectedLnurlpResponseBody{}
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(lnurlResponse))
	assert.Equal(suite.T(), lnurlResponse.Tag, payreq_type)
	assert.EqualValues(suite.T(), lnurlResponse.MinSendable, 1)
	assert.EqualValues(suite.T(), lnurlResponse.MaxSendable, suite.service.Config.MaxReceiveAmount)
	urlStart := strings.Index(lnurlResponse.Callback, "/v2/invoice")
	assert.Greater(suite.T(), urlStart, 0)

	// call callback
	const amt_msats = 1546000
	req = httptest.NewRequest(http.MethodGet, lnurlResponse.Callback[urlStart:]+"?amount="+strconv.FormatInt(int64(amt_msats), 10), nil)
	rec2 := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec2, req)
	invoiceResponse := &InvoiceResponseBody{}
	assert.Equal(suite.T(), http.StatusOK, rec2.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec2.Body).Decode(invoiceResponse))
	assert.Equal(suite.T(), 0, len(invoiceResponse.Routes))
	decodedPayreq, err := suite.mlnd.DecodeBolt11(context.Background(), invoiceResponse.Payreq)
	assert.NoError(suite.T(), err)
	assert.EqualValues(suite.T(), amt_msats, decodedPayreq.NumMsat)
	assert.NoError(suite.T(), err)
	descriptionHash := sha256.Sum256([]byte(lnurlResponse.Metadata))
	expectedPH := hex.EncodeToString(descriptionHash[:])
	assert.EqualValues(suite.T(), expectedPH, decodedPayreq.DescriptionHash)
}

func (suite *LnurlTestSuite) TestGetLnurlInvoiceCustomAmt() {

	// call the lnurl endpoint
	const payreq_type = "payRequest"
	const amt_sats = int64(1245)
	req := httptest.NewRequest(http.MethodGet, "/v2/lnurlp/"+suite.userLogin.Nickname+"?amt="+strconv.FormatInt(amt_sats, 10), nil)
	rec := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec, req)
	lnurlResponse := &ExpectedLnurlpResponseBody{}
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(lnurlResponse))
	assert.Equal(suite.T(), lnurlResponse.Tag, payreq_type)
	assert.EqualValues(suite.T(), lnurlResponse.MinSendable, amt_sats)
	assert.EqualValues(suite.T(), lnurlResponse.MaxSendable, amt_sats)
	urlStart := strings.Index(lnurlResponse.Callback, "/v2/invoice")
	assert.Greater(suite.T(), urlStart, 0)

	// call callback
	const amt_msats = 1546000
	req = httptest.NewRequest(http.MethodGet, lnurlResponse.Callback[urlStart:]+"?amount="+strconv.FormatInt(int64(amt_msats), 10), nil)
	rec2 := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec2, req)
	invoiceResponse := &InvoiceResponseBody{}
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
