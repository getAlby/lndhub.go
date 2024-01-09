package integration_tests

import (
	"context"
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

type ValidateUserSuite struct {
	TestSuite
	Service                  *service.LndhubService
	userLogin                ExpectedCreateUserResponseBody
	userToken                string
	mockLND                  *MockLND
	invoiceUpdateSubCancelFn context.CancelFunc
}

func (suite *ValidateUserSuite) SetupSuite() {
	mockLND := newDefaultMockLND()
	svc, err := LndHubTestServiceInit(mockLND)
	if err != nil {
		log.Fatalf("Error initializing test service: %v", err)
	}
	users, userTokens, err := createUsers(svc, 1)
	if err != nil {
		log.Fatalf("Error creating test users %v", err)
	}
	suite.Service = svc
	suite.mockLND = mockLND
	e := echo.New()

	e.HTTPErrorHandler = responses.HTTPErrorHandler
	e.Validator = &lib.CustomValidator{Validator: validator.New()}
	suite.echo = e
	suite.echo.Use(tokens.Middleware([]byte(suite.Service.Config.JWTSecret)))
	suite.echo.Use(svc.ValidateUserMiddleware())
	suite.echo.GET("/gettxs", controllers.NewGetTXSController(suite.Service).GetTXS)
	suite.echo.GET("/getuserinvoices", controllers.NewGetTXSController(svc).GetUserInvoices)
	suite.echo.POST("/addinvoice", controllers.NewAddInvoiceController(suite.Service).AddInvoice)
	suite.echo.POST("/payinvoice", controllers.NewPayInvoiceController(suite.Service).PayInvoice)

	assert.Equal(suite.T(), 1, len(users))
	suite.userLogin = users[0]
	suite.userToken = userTokens[0]
}

func (suite *ValidateUserSuite) TearDownSuite() {
}

func (suite *ValidateUserSuite) TestDeletedUserValidation() {
	_, err := suite.Service.DB.NewUpdate().Table("users").Set("deleted = ?", true).Where("login = ?", suite.userLogin.Login).Exec(context.TODO())
	assert.NoError(suite.T(), err)
	req := httptest.NewRequest(http.MethodGet, "/gettxs", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken))
	rec := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec, req)
	assert.Equal(suite.T(), http.StatusUnauthorized, rec.Code)

	_, err = suite.Service.DB.NewUpdate().Table("users").Set("deleted = ?", false).Where("login = ?", suite.userLogin.Login).Exec(context.TODO())
	assert.NoError(suite.T(), err)
	req = httptest.NewRequest(http.MethodGet, "/gettxs", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken))
	rec = httptest.NewRecorder()
	suite.echo.ServeHTTP(rec, req)
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
}

func (suite *ValidateUserSuite) TestDeactivatedUserValidation() {
	_, err := suite.Service.DB.NewUpdate().Table("users").Set("deactivated = ?, deleted = false", true).Where("login = ?", suite.userLogin.Login).Exec(context.TODO())
	assert.NoError(suite.T(), err)
	req := httptest.NewRequest(http.MethodGet, "/gettxs", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken))
	rec := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec, req)
	assert.Equal(suite.T(), http.StatusUnauthorized, rec.Code)

	_, err = suite.Service.DB.NewUpdate().Table("users").Set("deactivated = ?, deleted = false", false).Where("login = ?", suite.userLogin.Login).Exec(context.TODO())
	assert.NoError(suite.T(), err)
	req = httptest.NewRequest(http.MethodGet, "/gettxs", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken))
	rec = httptest.NewRecorder()
	suite.echo.ServeHTTP(rec, req)
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
}

func TestValidateUserSuite(t *testing.T) {
	suite.Run(t, new(ValidateUserSuite))
}
