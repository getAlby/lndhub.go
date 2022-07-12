package integration_tests

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	v2controllers "github.com/getAlby/lndhub.go/controllers_v2"
	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/security"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type CreateUserV2TestSuite struct {
	TestSuite
	Service *service.LndhubService
}

func (suite *CreateUserV2TestSuite) SetupSuite() {
	svc, err := LndHubTestServiceInit(newDefaultMockLND())
	if err != nil {
		log.Fatalf("Error initializing test service: %v", err)
	}
	suite.Service = svc
	e := echo.New()

	e.HTTPErrorHandler = responses.HTTPErrorHandler
	e.Validator = &lib.CustomValidator{Validator: validator.New()}
	suite.echo = e
	suite.echo.POST("/v2/create", v2controllers.NewCreateUserController(suite.Service).CreateUser, security.SignatureMiddleware())
}

func (suite *CreateUserV2TestSuite) TearDownTest() {
	err := clearTable(suite.Service, "users")
	if err != nil {
		fmt.Printf("Tear down test error %v\n", err.Error())
		return
	}
	fmt.Println("Tear down test success")
}

func (suite *CreateUserV2TestSuite) TestCreateAndChangeNickname() {
	e := echo.New()
	e.HTTPErrorHandler = responses.HTTPErrorHandler

	var buf bytes.Buffer
	pubKey, privKey, err := ed25519.GenerateKey(nil)
	assert.NoError(suite.T(), err)
	messageSigned := ed25519.Sign(privKey, []byte(security.LOGIN_MESSAGE))
	e.Validator = &lib.CustomValidator{Validator: validator.New()}
	const testLogin = "12D3KooWNmjM4sMbSkDEA6ShvjTgkrJHjMya46fhZ9PjKZ4KVZYq"
	assert.NoError(suite.T(), json.NewEncoder(&buf).Encode(&ExpectedCreateUserRequestBody{
		Login:    testLogin,
		Password: hex.EncodeToString(messageSigned),
	}))
	req := httptest.NewRequest(http.MethodPost, "/v2/create", &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", hex.EncodeToString(pubKey)))
	rec := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec, req)
	responseBody := ExpectedCreateUserRequestBody{}
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(&responseBody))
	assert.EqualValues(suite.T(), responseBody.Login, testLogin)
	assert.EqualValues(suite.T(), responseBody.Nickname, testLogin)
	assert.EqualValues(suite.T(), responseBody.Password, hex.EncodeToString(messageSigned))
	user, err := suite.Service.FindUserByLoginOrNickname(context.Background(), testLogin)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), user.Nickname, testLogin)

	const newNickname = "newNickname"
	assert.NoError(suite.T(), json.NewEncoder(&buf).Encode(&ExpectedCreateUserRequestBody{
		Login:    testLogin,
		Password: hex.EncodeToString(messageSigned),
		Nickname: newNickname,
	}))
	req2 := httptest.NewRequest(http.MethodPost, "/v2/create", &buf)
	req2.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec2 := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec2, req2)
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec2.Body).Decode(&responseBody))
	assert.EqualValues(suite.T(), responseBody.Login, testLogin)
	assert.EqualValues(suite.T(), responseBody.Nickname, newNickname)
	assert.EqualValues(suite.T(), responseBody.Password, hex.EncodeToString(messageSigned))
	user, err = suite.Service.FindUserByLoginOrNickname(context.Background(), newNickname)
	assert.NoError(suite.T(), err)
	assert.EqualValues(suite.T(), user.Nickname, newNickname)
}

func (suite *CreateUserV2TestSuite) TestCreateWrongSignature() {
	e := echo.New()
	e.HTTPErrorHandler = responses.HTTPErrorHandler

	var buf bytes.Buffer
	pubKey, _, err := ed25519.GenerateKey(nil)
	assert.NoError(suite.T(), err)
	_, privKey, err := ed25519.GenerateKey(nil)
	assert.NoError(suite.T(), err)
	messageSigned := ed25519.Sign(privKey, []byte(security.LOGIN_MESSAGE))
	e.Validator = &lib.CustomValidator{Validator: validator.New()}
	const testLogin = "12D3KooWNmjM4sMbSkDEA6ShvjTgkrJHjMya46fhZ9PjKZ4KVZYq"
	assert.NoError(suite.T(), json.NewEncoder(&buf).Encode(&ExpectedCreateUserRequestBody{
		Login:    testLogin,
		Password: hex.EncodeToString(messageSigned),
	}))
	req := httptest.NewRequest(http.MethodPost, "/v2/create", &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", hex.EncodeToString(pubKey)))
	rec := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec, req)
	assert.Equal(suite.T(), http.StatusUnauthorized, rec.Code)
}

func (suite *CreateUserV2TestSuite) TestCreateWithNoSignature() {
	e := echo.New()
	e.HTTPErrorHandler = responses.HTTPErrorHandler
	e.Validator = &lib.CustomValidator{Validator: validator.New()}
	var buf bytes.Buffer
	const testLogin = "Testlogin"
	const testPassword = "testPass"
	const testNickname = "testNickname"
	json.NewEncoder(&buf).Encode(&ExpectedCreateUserResponseBody{
		Login:    testLogin,
		Password: testPassword,
		Nickname: testNickname,
	})
	req := httptest.NewRequest(http.MethodPost, "/v2/create", &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec, req)
	responseBody := ExpectedCreateUserRequestBody{}
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(&responseBody))
	assert.EqualValues(suite.T(), responseBody.Login, testLogin)
	assert.EqualValues(suite.T(), responseBody.Nickname, testNickname)
	assert.EqualValues(suite.T(), responseBody.Password, testPassword)
}

func (suite *CreateUserV2TestSuite) TestCreateWrongNickname() {
	e := echo.New()
	e.HTTPErrorHandler = responses.HTTPErrorHandler
	e.Validator = &lib.CustomValidator{Validator: validator.New()}
	var buf bytes.Buffer
	const testLogin = "Testlogin"
	const testPassword = "testPass"
	const testNickname = " testNickname"
	json.NewEncoder(&buf).Encode(&ExpectedCreateUserResponseBody{
		Login:    testLogin,
		Password: testPassword,
		Nickname: testNickname,
	})
	req := httptest.NewRequest(http.MethodPost, "/v2/create", &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec, req)
	assert.Equal(suite.T(), http.StatusBadRequest, rec.Code)
}

func (suite *CreateUserV2TestSuite) TestCreateWithProvidedLoginAndPassword() {
	assert.NoError(suite.T(), nil)
	/*
		// TODO: uncomment once the test is ready
		user, err := suite.Service.FindUserByLoginOrNickname(context.Background(), testNickname)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), user.Login, testLogin)
	*/
}
func TestCreateUserV2TestSuite(t *testing.T) {
	suite.Run(t, new(CreateUserV2TestSuite))
}
