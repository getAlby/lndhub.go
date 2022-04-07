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
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type UserAuthTestSuite struct {
	suite.Suite
	Service   *service.LndhubService
	echo      *echo.Echo
	userLogin controllers.CreateUserResponseBody
}

func (suite *UserAuthTestSuite) SetupSuite() {
	svc, err := LndHubTestServiceInit(nil)
	if err != nil {
		log.Fatalf("Error initializing test service: %v", err)
	}
	users, _, err := createUsers(svc, 1)
	if err != nil {
		log.Fatalf("Error creating test users %v", err)
	}
	suite.Service = svc
	e := echo.New()

	e.HTTPErrorHandler = responses.HTTPErrorHandler
	e.Validator = &lib.CustomValidator{Validator: validator.New()}
	suite.echo = e
	assert.Equal(suite.T(), 1, len(users))
	suite.userLogin = users[0]
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
	fmt.Printf("Succesfully got a token using login and password: %s\n", responseBody.AccessToken)

	// login again with only refresh token
	assert.NoError(suite.T(), json.NewEncoder(&buf).Encode(&controllers.AuthRequestBody{
		RefreshToken: responseBody.RefreshToken,
	}))
	req = httptest.NewRequest(http.MethodPost, "/auth", &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = suite.echo.NewContext(req, rec)
	controller = controllers.NewAuthController(suite.Service)
	responseBody = &controllers.AuthResponseBody{}
	assert.NoError(suite.T(), controller.Auth(c))
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(&responseBody))
	assert.NotEmpty(suite.T(), responseBody.AccessToken)
	assert.NotEmpty(suite.T(), responseBody.RefreshToken)
	fmt.Printf("Succesfully got a token using refresh token only: %s\n", responseBody.AccessToken)
}

func (suite *UserAuthTestSuite) TestAuthWithExpiredRefreshToken() {
	// log in with login and password
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

	// get user
	userId := getUserIdFromToken(responseBody.AccessToken)
	user, _ := suite.Service.FindUser(context.Background(), userId)

	// expire in 0 seconds, with correct secret and user
	expiredRefreshToken, _ := tokens.GenerateRefreshToken(suite.Service.Config.JWTSecret, 0, user)

	// login again with only expired refresh token
	assert.NoError(suite.T(), json.NewEncoder(&buf).Encode(&controllers.AuthRequestBody{
		RefreshToken: expiredRefreshToken,
	}))

	// just to make sure that token expires
	time.Sleep(1 * time.Second)

	req = httptest.NewRequest(http.MethodPost, "/auth", &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = suite.echo.NewContext(req, rec)
	controller = controllers.NewAuthController(suite.Service)
	assert.NoError(suite.T(), controller.Auth(c))
	errorResponse := &responses.ErrorResponse{}
	assert.Equal(suite.T(), http.StatusUnauthorized, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(errorResponse))
	assert.Equal(suite.T(), responses.BadAuthError.Code, errorResponse.Code)
	assert.Equal(suite.T(), responses.BadAuthError.Message, errorResponse.Message)
	assert.Equal(suite.T(), responses.BadAuthError.Error, errorResponse.Error)
}

func (suite *UserAuthTestSuite) TestAuthWithInvalidSecretRefreshToken() {
	// log in with login and password
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

	// get user
	userId := getUserIdFromToken(responseBody.AccessToken)
	user, _ := suite.Service.FindUser(context.Background(), userId)

	// only secret is invalid here
	expiredRefreshToken, _ := tokens.GenerateRefreshToken([]byte("INVALID SECRET"), suite.Service.Config.JWTRefreshTokenExpiry, user)

	// login again with only refresh token
	assert.NoError(suite.T(), json.NewEncoder(&buf).Encode(&controllers.AuthRequestBody{
		RefreshToken: expiredRefreshToken,
	}))

	req = httptest.NewRequest(http.MethodPost, "/auth", &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = suite.echo.NewContext(req, rec)
	controller = controllers.NewAuthController(suite.Service)
	assert.NoError(suite.T(), controller.Auth(c))
	errorResponse := &responses.ErrorResponse{}
	assert.Equal(suite.T(), http.StatusUnauthorized, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(errorResponse))
	assert.Equal(suite.T(), responses.BadAuthError.Code, errorResponse.Code)
	assert.Equal(suite.T(), responses.BadAuthError.Message, errorResponse.Message)
	assert.Equal(suite.T(), responses.BadAuthError.Error, errorResponse.Error)
}

func (suite *UserAuthTestSuite) TestAuthWithInvalidUserIdRefreshToken() {
	// log in with login and password
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

	// get not existing user (userId + 1)
	userId := getUserIdFromToken(responseBody.AccessToken)
	user, _ := suite.Service.FindUser(context.Background(), userId+1)

	expiredRefreshToken, _ := tokens.GenerateRefreshToken(suite.Service.Config.JWTSecret, suite.Service.Config.JWTRefreshTokenExpiry, user)

	// login again with only refresh token
	assert.NoError(suite.T(), json.NewEncoder(&buf).Encode(&controllers.AuthRequestBody{
		RefreshToken: expiredRefreshToken,
	}))

	req = httptest.NewRequest(http.MethodPost, "/auth", &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = suite.echo.NewContext(req, rec)
	controller = controllers.NewAuthController(suite.Service)
	assert.NoError(suite.T(), controller.Auth(c))
	errorResponse := &responses.ErrorResponse{}
	assert.Equal(suite.T(), http.StatusUnauthorized, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(errorResponse))
	assert.Equal(suite.T(), responses.BadAuthError.Code, errorResponse.Code)
	assert.Equal(suite.T(), responses.BadAuthError.Message, errorResponse.Message)
	assert.Equal(suite.T(), responses.BadAuthError.Error, errorResponse.Error)
}

func (suite *UserAuthTestSuite) TestAuthWithAccessToken() {
	// log in with login and password
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

	// login again with only access token
	assert.NoError(suite.T(), json.NewEncoder(&buf).Encode(&controllers.AuthRequestBody{
		RefreshToken: responseBody.AccessToken,
	}))

	req = httptest.NewRequest(http.MethodPost, "/auth", &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = suite.echo.NewContext(req, rec)
	controller = controllers.NewAuthController(suite.Service)
	assert.NoError(suite.T(), controller.Auth(c))
	errorResponse := &responses.ErrorResponse{}
	assert.Equal(suite.T(), http.StatusUnauthorized, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(errorResponse))
	assert.Equal(suite.T(), responses.BadAuthError.Code, errorResponse.Code)
	assert.Equal(suite.T(), responses.BadAuthError.Message, errorResponse.Message)
	assert.Equal(suite.T(), responses.BadAuthError.Error, errorResponse.Error)
}

func (suite *UserAuthTestSuite) TestAuthWithNotParseableRefreshToken() {
	var buf bytes.Buffer
	// login with random not parseable refresh token
	assert.NoError(suite.T(), json.NewEncoder(&buf).Encode(&controllers.AuthRequestBody{
		RefreshToken: "12345",
	}))

	req := httptest.NewRequest(http.MethodPost, "/auth", &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := suite.echo.NewContext(req, rec)
	controller := controllers.NewAuthController(suite.Service)
	assert.NoError(suite.T(), controller.Auth(c))
	errorResponse := &responses.ErrorResponse{}
	assert.Equal(suite.T(), http.StatusUnauthorized, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(errorResponse))
	assert.Equal(suite.T(), responses.BadAuthError.Code, errorResponse.Code)
	assert.Equal(suite.T(), responses.BadAuthError.Message, errorResponse.Message)
	assert.Equal(suite.T(), responses.BadAuthError.Error, errorResponse.Error)
}

func TestUserAuthTestSuite(t *testing.T) {
	suite.Run(t, new(UserAuthTestSuite))
}
