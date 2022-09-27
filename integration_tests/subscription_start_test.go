package integration_tests

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/getAlby/lndhub.go/common"
	"github.com/getAlby/lndhub.go/controllers"
	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/getAlby/lndhub.go/lib/tokens"
	"github.com/getAlby/lndhub.go/lnd"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/uptrace/bun"
	"google.golang.org/grpc"
)

type SubscriptionStartTestSuite struct {
	TestSuite
	service   *service.LndhubService
	userLogin ExpectedCreateUserResponseBody
	userToken string
}

func (suite *SubscriptionStartTestSuite) TearDownSuite() {
	clearTable(suite.service, "invoices")
}

func TestSubscriptionStartTestSuite(t *testing.T) {
	suite.Run(t, new(SubscriptionStartTestSuite))
}
func (suite *SubscriptionStartTestSuite) SetupSuite() {
	svc, err := LndHubTestServiceInit(newDefaultMockLND())
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
	suite.userLogin = users[0]
	suite.userToken = userTokens[0]
	suite.echo.Use(tokens.Middleware([]byte(suite.service.Config.JWTSecret)))
	suite.echo.POST("/addinvoice", controllers.NewAddInvoiceController(suite.service).AddInvoice)

}
func (suite *SubscriptionStartTestSuite) TestAddIndex() {
	addIndexChannel := make(chan uint64)
	//overwrite lnd with mock client
	suite.service.LndClient = &lndSubscriptionStartMockClient{
		addIndexChannel: addIndexChannel,
	}

	ctx, cancel := context.WithCancel(context.Background())
	user, err := suite.service.FindUserByLogin(ctx, suite.userLogin.Login)
	assert.NoError(suite.T(), err)
	//add invoice to database that is already expired
	expiry := time.Hour * 24
	expiredInvoice := models.Invoice{
		Type:      common.InvoiceTypeIncoming,
		AddIndex:  1,
		UserID:    user.ID,
		Amount:    10,
		Memo:      "doesntmatter",
		State:     common.InvoiceStateInitialized,
		ExpiresAt: bun.NullTime{Time: time.Now().Add(-expiry)},
	}
	_, err = suite.service.DB.NewInsert().Model(&expiredInvoice).Exec(ctx)
	assert.NoError(suite.T(), err)

	//add non-expired invoice to database
	nonExpiredInvoice := models.Invoice{
		Type:      common.InvoiceTypeIncoming,
		AddIndex:  5,
		UserID:    user.ID,
		Amount:    10,
		Memo:      "doesntmatter",
		State:     common.InvoiceStateInitialized,
		ExpiresAt: bun.NullTime{Time: time.Now().Add(expiry)},
	}
	_, err = suite.service.DB.NewInsert().Model(&nonExpiredInvoice).Exec(ctx)
	assert.NoError(suite.T(), err)

	go suite.service.InvoiceUpdateSubscription(ctx)
	//check index value in channel to be the one of the non-expired invoice _minus_ one
	actualAddIndex := <-addIndexChannel
	assert.Equal(suite.T(), uint64(nonExpiredInvoice.AddIndex-1), actualAddIndex)
	cancel()
}

type lndSubscriptionStartMockClient struct {
	addIndexChannel chan (uint64)
}

func (mock *lndSubscriptionStartMockClient) ListChannels(ctx context.Context, req *lnrpc.ListChannelsRequest, options ...grpc.CallOption) (*lnrpc.ListChannelsResponse, error) {
	panic("not implemented") // TODO: Implement
}

func (mock *lndSubscriptionStartMockClient) SendPaymentSync(ctx context.Context, req *lnrpc.SendRequest, options ...grpc.CallOption) (*lnrpc.SendResponse, error) {
	panic("not implemented") // TODO: Implement
}

func (mock *lndSubscriptionStartMockClient) AddInvoice(ctx context.Context, req *lnrpc.Invoice, options ...grpc.CallOption) (*lnrpc.AddInvoiceResponse, error) {
	panic("not implemented") // TODO: Implement
}

func (mock *lndSubscriptionStartMockClient) SubscribeInvoices(ctx context.Context, req *lnrpc.InvoiceSubscription, options ...grpc.CallOption) (lnd.SubscribeInvoicesWrapper, error) {
	mock.addIndexChannel <- req.AddIndex
	return mock, nil
}

//wait forever
func (mock *lndSubscriptionStartMockClient) Recv() (*lnrpc.Invoice, error) {
	select {}
}

func (mock *lndSubscriptionStartMockClient) GetInfo(ctx context.Context, req *lnrpc.GetInfoRequest, options ...grpc.CallOption) (*lnrpc.GetInfoResponse, error) {
	panic("not implemented") // TODO: Implement
}

func (mock *lndSubscriptionStartMockClient) DecodeBolt11(ctx context.Context, bolt11 string, options ...grpc.CallOption) (*lnrpc.PayReq, error) {
	panic("not implemented") // TODO: Implement
}

func (mlnd *lndSubscriptionStartMockClient) TrackPayment(ctx context.Context, hash string, options ...grpc.CallOption) (*lnrpc.Payment, error) {
	return nil, nil
}
