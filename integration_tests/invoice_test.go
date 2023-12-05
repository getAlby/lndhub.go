package integration_tests

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getAlby/lndhub.go/common"
	"github.com/getAlby/lndhub.go/controllers"
	v2controllers "github.com/getAlby/lndhub.go/controllers_v2"
	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/getAlby/lndhub.go/lib/tokens"
	"github.com/getAlby/lndhub.go/lnd"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type InvoiceTestSuite struct {
	TestSuite
	service    *service.LndhubService
	aliceLogin ExpectedCreateUserResponseBody
	aliceToken string
}

func (suite *InvoiceTestSuite) SetupSuite() {
	svc, err := LndHubTestServiceInit(newDefaultMockLND())
	if err != nil {
		log.Fatalf("Error initializing test service: %v", err)
	}
	suite.service = svc
	users, userTokens, err := createUsers(svc, 1)
	if err != nil {
		log.Fatalf("Error creating test users: %v", err)
	}
	e := echo.New()

	e.HTTPErrorHandler = responses.HTTPErrorHandler
	e.Validator = &lib.CustomValidator{Validator: validator.New()}
	suite.echo = e
	assert.Equal(suite.T(), 1, len(users))
	assert.Equal(suite.T(), 1, len(userTokens))
	suite.aliceLogin = users[0]
	suite.aliceToken = userTokens[0]
	suite.echo.POST("/invoice/:user_login", controllers.NewInvoiceController(svc).Invoice)
	suite.echo.POST("/v2/invoices", v2controllers.NewInvoiceController(svc).AddInvoice, tokens.Middleware([]byte(suite.service.Config.JWTSecret), &lnd.Limits{}))
}

func (suite *InvoiceTestSuite) TearDownTest() {
	clearTable(suite.service, "invoices")
}

func (suite *InvoiceTestSuite) TestZeroAmtTestSuite() {
	rec := httptest.NewRecorder()
	var buf bytes.Buffer
	assert.NoError(suite.T(), json.NewEncoder(&buf).Encode(&ExpectedV2AddInvoiceRequestBody{
		Amount: 0,
		Memo:   "test zero amount v2 invoice",
	}))
	req := httptest.NewRequest(http.MethodPost, "/v2/invoices", &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", suite.aliceToken))
	suite.echo.ServeHTTP(rec, req)
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
}

func (suite *InvoiceTestSuite) TestAddInvoiceWithoutToken() {
	user, _ := suite.service.FindUserByLogin(context.Background(), suite.aliceLogin.Login)
	invoicesBefore, _ := suite.service.InvoicesFor(context.Background(), user.ID, common.InvoiceTypeIncoming)
	assert.Equal(suite.T(), 0, len(invoicesBefore))

	suite.createInvoiceReq(10, "test invoice without token", suite.aliceLogin.Login)

	// check if invoice is added
	invoicesAfter, _ := suite.service.InvoicesFor(context.Background(), user.ID, common.InvoiceTypeIncoming)
	assert.Equal(suite.T(), 1, len(invoicesAfter))
}

func (suite *InvoiceTestSuite) TestAddInvoiceForNonExistingUser() {
	nonExistingLogin := suite.aliceLogin.Login + "abc"
	suite.createInvoiceReqError(10, "test invoice without token", nonExistingLogin)
}
func (suite *InvoiceTestSuite) TestPreimageEntropy() {
	user, _ := suite.service.FindUserByLogin(context.Background(), suite.aliceLogin.Login)
	preimageChars := map[byte]int{}
	for i := 0; i < 1000; i++ {
		inv, errResp := suite.service.AddIncomingInvoice(context.Background(), user.ID, 10, "test entropy", "")
		assert.Nil(suite.T(), errResp)
		primgBytes, _ := hex.DecodeString(inv.Preimage)
		for _, char := range primgBytes {
			preimageChars[char] += 1
		}
	}
	//check that we use all possible byte values
	assert.Equal(suite.T(), 256, len(preimageChars))
}

func TestInvoiceSuite(t *testing.T) {
	suite.Run(t, new(InvoiceTestSuite))
}
