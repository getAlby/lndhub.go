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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/uptrace/bun/migrate"
)

//not used anymore
// const (
// 	lnd1RegtestAddress     = "rpc.lnd1.regtest.getalby.com:443"
// 	lnd1RegtestMacaroonHex = "0201036c6e6402f801030a10e2133a1cac2c5b4d56e44e32dc64c8551201301a160a0761646472657373120472656164120577726974651a130a04696e666f120472656164120577726974651a170a08696e766f69636573120472656164120577726974651a210a086d616361726f6f6e120867656e6572617465120472656164120577726974651a160a076d657373616765120472656164120577726974651a170a086f6666636861696e120472656164120577726974651a160a076f6e636861696e120472656164120577726974651a140a057065657273120472656164120577726974651a180a067369676e6572120867656e657261746512047265616400000620c4f9783e0873fa50a2091806f5ebb919c5dc432e33800b401463ada6485df0ed"
// 	lnd2RegtestAddress     = "rpc.lnd2.regtest.getalby.com:443"
// 	lnd2RegtestMacaroonHex = "0201036C6E6402F801030A101782922F4358E80655920FC7A7C3E9291201301A160A0761646472657373120472656164120577726974651A130A04696E666F120472656164120577726974651A170A08696E766F69636573120472656164120577726974651A210A086D616361726F6F6E120867656E6572617465120472656164120577726974651A160A076D657373616765120472656164120577726974651A170A086F6666636861696E120472656164120577726974651A160A076F6E636861696E120472656164120577726974651A140A057065657273120472656164120577726974651A180A067369676E6572120867656E657261746512047265616400000620628FFB2938C8540DD3AA5E578D9B43456835FAA176E175FFD4F9FBAE540E3BE9"
// 	// Use lnd3 for a funding client when testing out fee handling for payments done by lnd-1, since lnd3 doesn't have a direct channel to lnd1.
// 	// This will cause payment to be routed through lnd2, which will charge a fee (lnd default fee 1 sat base + 1 ppm).
// 	lnd3RegtestAddress     = "rpc.lnd3.regtest.getalby.com:443"
// 	lnd3RegtestMacaroonHex = "0201036c6e6402f801030a102a5aa69a5efdf4b4a55a5304b164641f1201301a160a0761646472657373120472656164120577726974651a130a04696e666f120472656164120577726974651a170a08696e766f69636573120472656164120577726974651a210a086d616361726f6f6e120867656e6572617465120472656164120577726974651a160a076d657373616765120472656164120577726974651a170a086f6666636861696e120472656164120577726974651a160a076f6e636861696e120472656164120577726974651a140a057065657273120472656164120577726974651a180a067369676e6572120867656e657261746512047265616400000620defbb5a809262297fd661a9ab6d3deb4b7acca4f1309c79addb952f0dc2d8c82"
// 	simnetLnd1PubKey       = "0242898f86064c2fd72de22059c947a83ba23e9d97aedeae7b6dba647123f1d71b"
// 	simnetLnd2PubKey       = "025c1d5d1b4c983cc6350fc2d756fbb59b4dc365e45e87f8e3afe07e24013e8220"
// 	simnetLnd3PubKey       = "03c7092d076f799ab18806743634b4c9bb34e351bdebc91d5b35963f3dc63ec5aa"
// )

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
		MaxFeeAmount:            1e6, //todo: add max fee test
		JWTSecret:               []byte("SECRET"),
		JWTAccessTokenExpiry:    3600,
		JWTRefreshTokenExpiry:   3600,
	}

	rabbitmqUri, ok := os.LookupEnv("RABBITMQ_URI")
	var rabbitmqClient rabbitmq.Client
	if ok {
		c.RabbitMQUri = rabbitmqUri
		c.RabbitMQLndhubInvoiceExchange = "test_lndhub_invoices"
		c.RabbitMQLndInvoiceExchange = "test_lnd_invoices"

		amqpClient, err := rabbitmq.DialAMQP(c.RabbitMQUri)
		if err != nil {
			return nil, err
		}

		rabbitmqClient, err = rabbitmq.NewClient(amqpClient,
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
		user, err := svc.CreateUser(context.Background(), "", "")
		if err != nil {
			return nil, nil, err
		}
		var login ExpectedCreateUserResponseBody
		login.Login = user.Login
		login.Password = user.Password
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
