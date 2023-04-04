package integration_tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/getAlby/lndhub.go/db"
	"github.com/getAlby/lndhub.go/db/migrations"
	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/getAlby/lndhub.go/lnd"
	"github.com/getAlby/lndhub.go/rabbitmq"
	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/uptrace/bun/migrate"
)

const (
	mockLNDAddress     = "mock.lnd.local"
	mockLNDMacaroonHex = "omnomnom"
)

func LndHubTestServiceInit(lndClientMock lnd.LightningClientWrapper) (svc *service.LndhubService, err error) {
	dbUri := "postgresql://user:password@localhost/lndhub?sslmode=disable"
	c := &service.Config{
		DatabaseUri:             dbUri,
		DatabaseMaxConns:        1,
		DatabaseMaxIdleConns:    1,
		DatabaseConnMaxLifetime: 10,
		JWTSecret:               []byte("SECRET"),
		JWTAccessTokenExpiry:    3600,
		JWTRefreshTokenExpiry:   3600,
		LNDAddress:              mockLNDAddress,
		LNDMacaroonHex:          mockLNDMacaroonHex,
		MaxReceiveAmount:        1000000,
		MaxSendAmount:           100000,
		LnurlDomain:             "testnet.example.com",
	}

	rabbitmqUri, ok := os.LookupEnv("RABBITMQ_URI")
	var rabbitmqClient rabbitmq.Client
	if ok {
		c.RabbitMQUri = rabbitmqUri
		c.RabbitMQLndhubInvoiceExchange = "test_lndhub_invoices"
		c.RabbitMQLndInvoiceExchange = "test_lnd_invoices"
		rabbitmqClient, err = rabbitmq.Dial(c.RabbitMQUri,
			rabbitmq.WithLndInvoiceExchange(c.RabbitMQLndInvoiceExchange),
			rabbitmq.WithLndHubInvoiceExchange(c.RabbitMQLndhubInvoiceExchange),
			rabbitmq.WithLndInvoiceConsumerQueueName(c.RabbitMQInvoiceConsumerQueueName),
		)
		if err != nil {
			return nil, err
		}
	}

	dbConn, err := db.Open(c)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	ctx := context.Background()
	migrator := migrate.NewMigrator(dbConn, migrations.Migrations)
	err = migrator.Init(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to init migrations: %w", err)
	}
	_, err = migrator.Migrate(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate: %w", err)
	}

	logger := lib.Logger(c.LogFilePath)
	svc = &service.LndhubService{
		Config:         c,
		DB:             dbConn,
		LndClient:      lndClientMock,
		Logger:         logger,
		RabbitMQClient: rabbitmqClient,
	}
	getInfo, err := lndClientMock.GetInfo(ctx, &lnrpc.GetInfoRequest{})
	if err != nil {
		logger.Fatalf("Error getting node info: %v", err)
	}
	svc.IdentityPubkey = getInfo.IdentityPubkey

	svc.InvoicePubSub = service.NewPubsub()
	return svc, nil
}

func clearTable(svc *service.LndhubService, tableName string) error {
	dbConn, err := db.Open(svc.Config)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	_, err = dbConn.Exec(fmt.Sprintf("DELETE FROM %s", tableName))
	return err
}

// unsafe parse jwt method to pull out userId claim
// should be used only in integration_tests package
func getUserIdFromToken(token string) int64 {
	parsedToken, _, _ := new(jwt.Parser).ParseUnverified(token, jwt.MapClaims{})
	claims, _ := parsedToken.Claims.(jwt.MapClaims)
	return int64(claims["id"].(float64))
}

func createUsers(svc *service.LndhubService, usersToCreate int) (logins []ExpectedCreateUserResponseBody, tokens []string, err error) {
	logins = []ExpectedCreateUserResponseBody{}
	tokens = []string{}
	for i := 0; i < usersToCreate; i++ {
		user, err := svc.CreateUser(context.Background(), "", "", "")
		if err != nil {
			return nil, nil, err
		}
		var login ExpectedCreateUserResponseBody
		login.Login = user.Login
		login.Password = user.Password
		login.Nickname = user.Nickname
		logins = append(logins, login)
		token, _, err := svc.GenerateToken(context.Background(), login.Login, login.Password, "")
		if err != nil {
			return nil, nil, err
		}
		tokens = append(tokens, token)
	}
	return logins, tokens, nil
}

type TestSuite struct {
	suite.Suite
	echo *echo.Echo
}

func checkErrResponse(suite *TestSuite, rec *httptest.ResponseRecorder) *responses.ErrorResponse {
	errorResponse := &responses.ErrorResponse{}
	assert.Equal(suite.T(), http.StatusBadRequest, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(errorResponse))
	return errorResponse
}

func (suite *TestSuite) createAddInvoiceReq(amt int, memo, token string) *ExpectedAddInvoiceResponseBody {
	rec := httptest.NewRecorder()
	var buf bytes.Buffer
	assert.NoError(suite.T(), json.NewEncoder(&buf).Encode(&ExpectedAddInvoiceRequestBody{
		Amount: amt,
		Memo:   memo,
	}))
	req := httptest.NewRequest(http.MethodPost, "/addinvoice", &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	suite.echo.ServeHTTP(rec, req)
	invoiceResponse := &ExpectedAddInvoiceResponseBody{}
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(invoiceResponse))
	return invoiceResponse
}

func (suite *TestSuite) createInvoiceReq(amt int, memo, userLogin string) *ExpectedAddInvoiceResponseBody {
	rec := httptest.NewRecorder()
	var buf bytes.Buffer
	assert.NoError(suite.T(), json.NewEncoder(&buf).Encode(&ExpectedAddInvoiceRequestBody{
		Amount: amt,
		Memo:   memo,
	}))
	req := httptest.NewRequest(http.MethodPost, "/invoice/"+userLogin, &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	suite.echo.ServeHTTP(rec, req)
	invoiceResponse := &ExpectedAddInvoiceResponseBody{}
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(invoiceResponse))
	return invoiceResponse
}

func (suite *TestSuite) createInvoiceReqError(amt int, memo, userLogin string) *responses.ErrorResponse {
	rec := httptest.NewRecorder()
	var buf bytes.Buffer
	assert.NoError(suite.T(), json.NewEncoder(&buf).Encode(&ExpectedAddInvoiceRequestBody{
		Amount: amt,
		Memo:   memo,
	}))
	req := httptest.NewRequest(http.MethodPost, "/invoice/"+userLogin, &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	suite.echo.ServeHTTP(rec, req)
	return checkErrResponse(suite, rec)
}

func (suite *TestSuite) createKeySendReq(amount int64, memo, destination, token string) *ExpectedKeySendResponseBody {
	rec := httptest.NewRecorder()
	var buf bytes.Buffer
	assert.NoError(suite.T(), json.NewEncoder(&buf).Encode(ExpectedKeySendRequestBody{
		Amount:      amount,
		Destination: destination,
		Memo:        memo,
		//add memo as WHATSAT_MESSAGE custom record
		CustomRecords: map[string]string{fmt.Sprint(service.TLV_WHATSAT_MESSAGE): memo},
	}))
	req := httptest.NewRequest(http.MethodPost, "/keysend", &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	suite.echo.ServeHTTP(rec, req)

	keySendResponse := &ExpectedKeySendResponseBody{}
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(keySendResponse))
	return keySendResponse
}

func (suite *TestSuite) createKeySendReqError(amount int64, memo, destination, token string) *responses.ErrorResponse {
	rec := httptest.NewRecorder()
	var buf bytes.Buffer
	assert.NoError(suite.T(), json.NewEncoder(&buf).Encode(ExpectedKeySendRequestBody{
		Amount:      amount,
		Destination: destination,
		Memo:        memo,
	}))
	req := httptest.NewRequest(http.MethodPost, "/keysend", &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	suite.echo.ServeHTTP(rec, req)
	return checkErrResponse(suite, rec)
}

func (suite *TestSuite) createPayInvoiceReq(payReq *ExpectedPayInvoiceRequestBody, token string) *ExpectedPayInvoiceResponseBody {
	rec := httptest.NewRecorder()
	var buf bytes.Buffer
	assert.NoError(suite.T(), json.NewEncoder(&buf).Encode(payReq))
	req := httptest.NewRequest(http.MethodPost, "/payinvoice", &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	suite.echo.ServeHTTP(rec, req)

	payInvoiceResponse := &ExpectedPayInvoiceResponseBody{}
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	assert.NoError(suite.T(), json.NewDecoder(rec.Body).Decode(payInvoiceResponse))
	return payInvoiceResponse
}

func (suite *TestSuite) createPayInvoiceReqError(payReq string, token string) *responses.ErrorResponse {
	rec := httptest.NewRecorder()
	var buf bytes.Buffer
	assert.NoError(suite.T(), json.NewEncoder(&buf).Encode(&ExpectedPayInvoiceRequestBody{
		Invoice: payReq,
	}))
	req := httptest.NewRequest(http.MethodPost, "/payinvoice", &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	suite.echo.ServeHTTP(rec, req)
	return checkErrResponse(suite, rec)
}

func (suite *TestSuite) createPayInvoiceReqWithCancel(payReq string, token string) {
	rec := httptest.NewRecorder()
	var buf bytes.Buffer
	assert.NoError(suite.T(), json.NewEncoder(&buf).Encode(&ExpectedPayInvoiceRequestBody{
		Invoice: payReq,
	}))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req := httptest.NewRequest(http.MethodPost, "/payinvoice", &buf).WithContext(ctx)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	suite.echo.ServeHTTP(rec, req)
}
