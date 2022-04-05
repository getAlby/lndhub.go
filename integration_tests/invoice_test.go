package integration_tests

import (
	"context"
	"log"
	"testing"

	"github.com/getAlby/lndhub.go/common"
	"github.com/getAlby/lndhub.go/controllers"
	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type InvoiceTestSuite struct {
	TestSuite
	service    *service.LndhubService
	aliceLogin ExpectedCreateUserResponseBody
}

func (suite *InvoiceTestSuite) SetupSuite() {
	svc, err := LndHubTestServiceInit(nil)
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
	suite.echo.POST("/invoice/:user_login", controllers.NewInvoiceController(svc).Invoice)
}

func (suite *InvoiceTestSuite) TearDownTest() {
	clearTable(suite.service, "invoices")
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

func TestInvoiceSuite(t *testing.T) {
	suite.Run(t, new(InvoiceTestSuite))
}
