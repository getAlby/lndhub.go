package integration_tests

import (
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
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type GetInfoTestSuite struct {
	TestSuite
	service   *service.LndhubService
	userLogin controllers.CreateUserResponseBody
	userToken string
}

func (suite *GetInfoTestSuite) SetupSuite() {
	svc, err := LndHubTestServiceInit(nil)
	if err != nil {
		log.Fatalf("Error initializing test service: %v", err)
	}
	users, userTokens, err := createUsers(svc, 1)
	if err != nil {
		log.Fatalf("Error creating test users: %v", err)
	}

	suite.service = svc
	e := echo.New()

	e.HTTPErrorHandler = responses.HTTPErrorHandler
	e.Validator = &lib.CustomValidator{Validator: validator.New()}
	suite.echo = e
	assert.Equal(suite.T(), 1, len(users))
	assert.Equal(suite.T(), 1, len(userTokens))
	suite.userLogin = users[0]
	suite.userToken = userTokens[0]
	suite.echo.Use(tokens.Middleware([]byte(suite.service.Config.JWTSecret)))
	suite.echo.GET("/getinfo", controllers.NewGetInfoController(svc).GetInfo)
}

func (suite *GetInfoTestSuite) TestGetInfoWithDefaultAlias() {
	req := httptest.NewRequest(http.MethodGet, "/getinfo", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken))
	rec := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec, req)
	getInfoResponse := &lnrpc.GetInfoResponse{}
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(getInfoResponse))
	assert.NotNil(suite.T(), getInfoResponse)
	assert.Equal(suite.T(), "alby-simnet-lnd1", getInfoResponse.Alias)
}

func (suite *GetInfoTestSuite) TestGetInfoWithGivenAlias() {
	suite.service.Config.CustomName = "test-alias"
	req := httptest.NewRequest(http.MethodGet, "/getinfo", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", suite.userToken))
	rec := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec, req)
	getInfoResponse := &lnrpc.GetInfoResponse{}
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(getInfoResponse))
	assert.NotNil(suite.T(), getInfoResponse)
	assert.Equal(suite.T(), suite.service.Config.CustomName, getInfoResponse.Alias)
}

func (suite *GetInfoTestSuite) TearDownSuite() {}

func TestGetInfoSuite(t *testing.T) {
	suite.Run(t, new(GetInfoTestSuite))
}
