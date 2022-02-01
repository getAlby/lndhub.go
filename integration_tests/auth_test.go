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
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type UserAuthTestSuite struct {
	suite.Suite
	Service   *service.LndhubService
	echo      *echo.Echo
	userLogin *controllers.CreateUserResponseBody
}

func (suite *UserAuthTestSuite) SetupSuite() {
	svc, err := LndHubTestServiceInit()
	if err != nil {
		log.Fatalf("Error initializing test service: %v", err)
	}
	suite.Service = svc
	e := echo.New()

	e.HTTPErrorHandler = responses.HTTPErrorHandler
	e.Validator = &lib.CustomValidator{Validator: validator.New()}
	suite.echo = e
	//create user
	req := httptest.NewRequest(http.MethodPost, "/create", bytes.NewReader([]byte{}))
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	controller := controllers.NewCreateUserController(suite.Service)
	responseBody := controllers.CreateUserResponseBody{}
	assert.NoError(suite.T(), controller.CreateUser(c))
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(&responseBody))
	suite.userLogin = &responseBody
}

func (suite *UserAuthTestSuite) TearDownSuite() {

}

func (suite *UserAuthTestSuite) TestAuth() {
	var buf bytes.Buffer
	assert.NoError(suite.T(), json.NewEncoder(&buf).Encode(&controllers.AuthRequestBody{
		Login:    suite.userLogin.Login,
		Password: suite.userLogin.Password,
	}))
	req := httptest.NewRequest(http.MethodPost, "/auth", &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := suite.echo.NewContext(req, rec)
	controller := controllers.NewAuthController(suite.Service)
	responseBody := &controllers.AuthResponseBody{}
	assert.NoError(suite.T(), controller.Auth(c))
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(&responseBody))
	assert.NotEmpty(suite.T(), responseBody.AccessToken)
	assert.NotEmpty(suite.T(), responseBody.RefreshToken)
	fmt.Printf("Succesfully got a token: %s\n", responseBody.AccessToken)
}

func TestUserAuthTestSuite(t *testing.T) {
	suite.Run(t, new(UserAuthTestSuite))
}
