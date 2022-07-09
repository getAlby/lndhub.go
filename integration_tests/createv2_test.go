package integration_tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
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

type CreateUserV2TestSuite struct {
	suite.Suite
	Service *service.LndhubService
}

func (suite *CreateUserV2TestSuite) SetupSuite() {
	svc, err := LndHubTestServiceInit(newDefaultMockLND())
	if err != nil {
		log.Fatalf("Error initializing test service: %v", err)
	}
	suite.Service = svc
}

func (suite *CreateUserV2TestSuite) TearDownSuite() {

}

func (suite *CreateUserV2TestSuite) TearDownTest() {
	err := clearTable(suite.Service, "users")
	if err != nil {
		fmt.Printf("Tear down test error %v\n", err.Error())
		return
	}
	fmt.Println("Tear down test success")
}

func (suite *CreateUserV2TestSuite) TestCreate() {
	e := echo.New()
	e.HTTPErrorHandler = responses.HTTPErrorHandler
	e.Validator = &lib.CustomValidator{Validator: validator.New()}
	req := httptest.NewRequest(http.MethodPost, "/create", bytes.NewReader([]byte{}))
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	controller := v2controllers.NewCreateUserController(suite.Service)
	responseBody := ExpectedCreateUser{}
	if assert.NoError(suite.T(), controller.CreateUser(c)) {
		assert.Equal(suite.T(), http.StatusOK, rec.Code)
		assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(&responseBody))
		assert.NotEmpty(suite.T(), responseBody.Login)
		assert.NotEmpty(suite.T(), responseBody.Password)
		fmt.Printf("Successfully created user with login %s\n", responseBody.Login)
	}
}

func (suite *CreateUserV2TestSuite) TestCreateWrongWithProvidedLoginAndPassword() {
	e := echo.New()
	e.HTTPErrorHandler = responses.HTTPErrorHandler
	e.Validator = &lib.CustomValidator{Validator: validator.New()}
	var buf bytes.Buffer
	const testLogin = "12D3KooWNmjM4sMbSkDEA6ShvjTgkrJHjMya46fhZ9PjKZ4KVZYq"
	const testPassword = "wrong signature"
	const testNickname = "test nickname"
	json.NewEncoder(&buf).Encode(&ExpectedCreateUser{
		Login:    testLogin,
		Password: testPassword,
		Nickname: testNickname,
	})
	req := httptest.NewRequest(http.MethodPost, "/v2/create", &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	controller := v2controllers.NewCreateUserController(suite.Service)
	responseBody := ExpectedCreateUserResponseBody{}
	if assert.NoError(suite.T(), controller.CreateUser(c)) {
		assert.Equal(suite.T(), http.StatusBadRequest, rec.Code)
		assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(&responseBody))
	}
}

func TestCreateUserV2TestSuite(t *testing.T) {
	suite.Run(t, new(CreateUserV2TestSuite))
}
