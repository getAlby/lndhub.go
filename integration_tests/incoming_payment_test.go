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

const (
	lnd2RegtestAddress     = "rpc.lnd2.regtest.getalby.com:443"
	lnd2RegtestMacaroonHex = "0201036C6E6402F801030A101782922F4358E80655920FC7A7C3E9291201301A160A0761646472657373120472656164120577726974651A130A04696E666F120472656164120577726974651A170A08696E766F69636573120472656164120577726974651A210A086D616361726F6F6E120867656E6572617465120472656164120577726974651A160A076D657373616765120472656164120577726974651A170A086F6666636861696E120472656164120577726974651A160A076F6E636861696E120472656164120577726974651A140A057065657273120472656164120577726974651A180A067369676E6572120867656E657261746512047265616400000620628FFB2938C8540DD3AA5E578D9B43456835FAA176E175FFD4F9FBAE540E3BE9"
)

type IncomingPaymentTestSuite struct {
	TestSuite
	fundingClient *lnd.LNDWrapper
	service       *service.LndhubService
	userLogin     controllers.CreateUserResponseBody
	userToken     string
}

func (suite *IncomingPaymentTestSuite) SetupSuite() {
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
	users, tokens, err := createUsers(svc, 1)
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
	assert.Equal(suite.T(), 1, len(users))
	assert.Equal(suite.T(), 1, len(tokens))
	suite.userLogin = users[0]
	suite.userToken = tokens[0]
}

func (suite *IncomingPaymentTestSuite) TearDownSuite() {

}

func (suite *IncomingPaymentTestSuite) TestIncomingPayment() {
	var buf bytes.Buffer
	req := httptest.NewRequest(http.MethodGet, "/balance", &buf)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken))
	rec := httptest.NewRecorder()
	suite.echo.Use(tokens.Middleware([]byte(suite.service.Config.JWTSecret)))
	suite.echo.GET("/balance", controllers.NewBalanceController(suite.service).Balance)
	suite.echo.POST("/addinvoice", controllers.NewAddInvoiceController(suite.service).AddInvoice)
	suite.echo.ServeHTTP(rec, req)
	balance := &controllers.BalanceResponse{}
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(&balance))
	//assert the user has no balance to start with
	assert.Equal(suite.T(), int64(0), balance.BTC.AvailableBalance)
	fundingSatAmt := 10
	invoiceResponse := suite.createAddInvoiceReq(fundingSatAmt, "integration test IncomingPaymentTestSuite", suite.userToken)
	//try to pay invoice with external node
	// Prepare the LNRPC call
	sendPaymentRequest := lnrpc.SendRequest{
		PaymentRequest: invoiceResponse.PayReq,
		FeeLimit:       nil,
	}
	_, err := suite.fundingClient.SendPaymentSync(context.Background(), &sendPaymentRequest)
	assert.NoError(suite.T(), err)

	//wait a bit for the callback event to hit
	time.Sleep(100 * time.Millisecond)
	//check balance again
	req = httptest.NewRequest(http.MethodGet, "/balance", &buf)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken))
	suite.echo.ServeHTTP(rec, req)
	balance = &controllers.BalanceResponse{}
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(&balance))
	//assert the balance was added to the user's account
	assert.Equal(suite.T(), int64(fundingSatAmt), balance.BTC.AvailableBalance)

}

func TestIncomingPaymentTestSuite(t *testing.T) {
	suite.Run(t, new(IncomingPaymentTestSuite))
}
