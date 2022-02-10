package integration_tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getAlby/lndhub.go/controllers"
	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/getAlby/lndhub.go/lib/tokens"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type GetTxTestSuite struct {
	TestSuite
	Service   *service.LndhubService
	userLogin controllers.CreateUserResponseBody
	userToken string
}

type GetOutgoingInvoiceResponseTest struct {
	RHash           interface{} `json:"r_hash"`
	PaymentHash     interface{} `json:"payment_hash"`
	PaymentPreimage string      `json:"payment_preimage"`
	Value           int64       `json:"value"`
	Fee             int64       `json:"fee"`
	Timestamp       int64       `json:"timestamp"`
	Memo            string      `json:"memo"`
}

type GetIncomingInvoiceResponseTest struct {
	RHash          interface{} `json:"r_hash"`
	PaymentHash    interface{} `json:"payment_hash"`
	PaymentRequest string      `json:"payment_request"`
	Description    string      `json:"description"`
	PayReq         string      `json:"pay_req"`
	Timestamp      int64       `json:"timestamp"`
	Type           string      `json:"type"`
	ExpireTime     int64       `json:"expire_time"`
	Amount         int64       `json:"amt"`
	IsPaid         bool        `json:"ispaid"`
}

func (suite *GetTxTestSuite) SetupSuite() {
	fmt.Println("SETUP")
	svc, err := LndHubTestServiceInit()
	if err != nil {
		log.Fatalf("Error initializing test service: %v", err)
	}
	users, userTokens, err := createUsers(svc, 1)
	if err != nil {
		log.Fatalf("Error creating test users %v", err)
	}
	suite.Service = svc
	e := echo.New()

	e.HTTPErrorHandler = responses.HTTPErrorHandler
	e.Validator = &lib.CustomValidator{Validator: validator.New()}
	suite.echo = e
	suite.echo.Use(tokens.Middleware([]byte(suite.Service.Config.JWTSecret)))
	suite.echo.GET("/gettxs", controllers.NewGetTXSController(suite.Service).GetTXS)
	suite.echo.GET("/getuserinvoices", controllers.NewGetTXSController(svc).GetUserInvoices)
	suite.echo.POST("/addinvoice", controllers.NewAddInvoiceController(suite.Service).AddInvoice)
	suite.echo.POST("/payinvoice", controllers.NewPayInvoiceController(suite.Service).PayInvoice)

	assert.Equal(suite.T(), 1, len(users))
	suite.userLogin = users[0]
	suite.userToken = userTokens[0]
}

func (suite *GetTxTestSuite) TearDownSuite() {

}

func (suite *GetTxTestSuite) TestGetEmptyTXs() {
	// check that invoices are empty
	req := httptest.NewRequest(http.MethodGet, "/gettxs", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken))
	rec := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec, req)
	responseBody := &[]GetOutgoingInvoiceResponseTest{}
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(&responseBody))
	assert.Empty(suite.T(), responseBody)
	// create incoming invoice
	invoice := suite.createAddInvoiceReq(1000, "integration test internal payment alice", suite.userToken)
	// pay invoice
	rec = httptest.NewRecorder()
	var buf bytes.Buffer
	assert.NoError(suite.T(), json.NewEncoder(&buf).Encode(&controllers.PayInvoiceRequestBody{
		Invoice: invoice.PayReq,
	}))
	req = httptest.NewRequest(http.MethodPost, "/payinvoice", &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken))
	suite.echo.ServeHTTP(rec, req)
	// check invoices again
	req = httptest.NewRequest(http.MethodGet, "/gettxs", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken))
	rec = httptest.NewRecorder()
	suite.echo.ServeHTTP(rec, req)
	responseBody = &[]GetOutgoingInvoiceResponseTest{}
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(&responseBody))
	assert.Equal(suite.T(), 1, len(*responseBody))
}

func (suite *GetTxTestSuite) TestGetIncomingTXs() {
	// check that invoices are empty
	req := httptest.NewRequest(http.MethodGet, "/getuserinvoices", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken))
	rec := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec, req)
	responseBody := &[]GetIncomingInvoiceResponseTest{}
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(&responseBody))
	assert.Empty(suite.T(), responseBody)
	// create incoming invoice
	suite.createAddInvoiceReq(1000, "integration test internal payment alice", suite.userToken)
	// check invoices again
	req = httptest.NewRequest(http.MethodGet, "/getuserinvoices", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken))
	rec = httptest.NewRecorder()
	suite.echo.ServeHTTP(rec, req)
	// controller := controllers.NewGetTXSController(suite.Service)
	responseBody = &[]GetIncomingInvoiceResponseTest{}
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(&responseBody))
	assert.Equal(suite.T(), 1, len(*responseBody))
}

func TestGetTXsTestSuite(t *testing.T) {
	suite.Run(t, new(GetTxTestSuite))
}
