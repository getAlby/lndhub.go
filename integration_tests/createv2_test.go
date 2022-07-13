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

	"math/rand"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/crypto"

	v2controllers "github.com/getAlby/lndhub.go/controllers_v2"
	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/security"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

var (
	PRIV_KEY = []byte{8, 1, 18, 64, 250, 126, 64, 211, 185, 52, 213, 138, 129, 240, 49, 215, 8, 0, 143, 232, 142, 33, 34, 171, 16, 219, 41, 128, 102, 115, 188, 59, 39, 71, 124, 184, 234, 207, 90, 7, 190, 245, 13, 28, 12, 234, 139, 238, 38, 154, 82, 54, 239, 185, 155, 12, 144, 51, 65, 143, 172, 48, 165, 199, 34, 254, 25, 96}
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
	e.Validator = &lib.CustomValidator{Validator: validator.New()}
	priv, err := crypto.UnmarshalPrivateKey(PRIV_KEY)
	assert.NoError(suite.T(), err)
	pub := priv.GetPublic().(*crypto.Ed25519PublicKey)
	pubBytes, _ := pub.Raw()
	messageSigned, err := priv.Sign([]byte(security.LOGIN_MESSAGE))
	assert.NoError(suite.T(), err)
	pid, err := peer.IDFromPublicKey(pub)
	assert.NoError(suite.T(), err)
	mh, err := multihash.Cast([]byte(pid))
	assert.NoError(suite.T(), err)
	testLogin := cid.NewCidV1(security.MHASH_CODEC, mh).String()

	assert.NoError(suite.T(), json.NewEncoder(&buf).Encode(&ExpectedCreateUserRequestBody{
		Login:    testLogin,
		Password: hex.EncodeToString(messageSigned),
	}))
	req := httptest.NewRequest(http.MethodPost, "/v2/create", &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", hex.EncodeToString(pubBytes)))
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
	req2.Header.Add("Authorization", fmt.Sprintf("Bearer %s", hex.EncodeToString(pubBytes)))
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

func (suite *CreateUserV2TestSuite) TestCreateWrongLogin() {
	e := echo.New()
	e.HTTPErrorHandler = responses.HTTPErrorHandler

	var buf bytes.Buffer
	e.Validator = &lib.CustomValidator{Validator: validator.New()}
	priv, err := crypto.UnmarshalPrivateKey(PRIV_KEY)
	assert.NoError(suite.T(), err)
	pub := priv.GetPublic().(*crypto.Ed25519PublicKey)
	pubBytes, _ := pub.Raw()
	messageSigned, err := priv.Sign([]byte(security.LOGIN_MESSAGE))
	assert.NoError(suite.T(), err)
	pid, err := peer.IDFromPublicKey(pub)
	assert.NoError(suite.T(), err)
	mh, err := multihash.Cast([]byte(pid))
	assert.NoError(suite.T(), err)
	testLogin := cid.NewCidV1(security.MHASH_CODEC, mh).String() + "a"
	e.Validator = &lib.CustomValidator{Validator: validator.New()}

	assert.NoError(suite.T(), json.NewEncoder(&buf).Encode(&ExpectedCreateUserRequestBody{
		Login:    testLogin,
		Password: hex.EncodeToString(messageSigned),
	}))
	req := httptest.NewRequest(http.MethodPost, "/v2/create", &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", hex.EncodeToString(pubBytes)))
	rec := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec, req)
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	responseBody := ExpectedCreateUserRequestBody{}
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(&responseBody))
	assert.EqualValues(suite.T(), responseBody.Login, testLogin)
	assert.EqualValues(suite.T(), responseBody.Nickname, testLogin)
	assert.EqualValues(suite.T(), responseBody.Password, hex.EncodeToString(messageSigned))
	user, err := suite.Service.FindUserByLoginOrNickname(context.Background(), testLogin)
	assert.NoError(suite.T(), err)
	assert.EqualValues(suite.T(), user.Nickname, testLogin)
}

func (suite *CreateUserV2TestSuite) TestCreateWrongSignature() {
	e := echo.New()
	e.HTTPErrorHandler = responses.HTTPErrorHandler

	var buf bytes.Buffer
	e.Validator = &lib.CustomValidator{Validator: validator.New()}
	priv, err := crypto.UnmarshalPrivateKey(PRIV_KEY)
	assert.NoError(suite.T(), err)
	pub := priv.GetPublic().(*crypto.Ed25519PublicKey)
	pubBytes, _ := pub.Raw()
	messageSigned := make([]byte, 64)
	rand.Read(messageSigned)
	assert.NoError(suite.T(), err)
	pid, err := peer.IDFromPublicKey(pub)
	assert.NoError(suite.T(), err)
	mh, err := multihash.Cast([]byte(pid))
	assert.NoError(suite.T(), err)
	testLogin := cid.NewCidV1(security.MHASH_CODEC, mh).String()
	e.Validator = &lib.CustomValidator{Validator: validator.New()}

	assert.NoError(suite.T(), json.NewEncoder(&buf).Encode(&ExpectedCreateUserRequestBody{
		Login:    testLogin,
		Password: hex.EncodeToString(messageSigned),
	}))
	req := httptest.NewRequest(http.MethodPost, "/v2/create", &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", hex.EncodeToString(pubBytes)))
	rec := httptest.NewRecorder()
	suite.echo.ServeHTTP(rec, req)
	assert.Equal(suite.T(), http.StatusUnauthorized, rec.Code)
}

func (suite *CreateUserV2TestSuite) TestCreateWithNoSignature() {
	e := echo.New()
	e.HTTPErrorHandler = responses.HTTPErrorHandler

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

func TestCreateUserV2TestSuite(t *testing.T) {
	suite.Run(t, new(CreateUserV2TestSuite))
}
