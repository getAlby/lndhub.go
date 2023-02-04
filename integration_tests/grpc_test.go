package integration_tests

import (
	"context"
	"fmt"
	"log"
	"net"
	"testing"

	"github.com/getAlby/lndhub.go/common"
	"github.com/getAlby/lndhub.go/controllers"
	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/getAlby/lndhub.go/lib/tokens"
	"github.com/getAlby/lndhub.go/lndhubrpc"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GrpcTestSuite struct {
	TestSuite
	service                  *service.LndhubService
	mlnd                     *MockLND
	userLogin                ExpectedCreateUserResponseBody
	userToken                string
	invoiceChan              chan (*lndhubrpc.Invoice)
	grpcClient               lndhubrpc.InvoiceSubscription_SubsribeInvoicesClient
	invoiceUpdateSubCancelFn context.CancelFunc
}

func (suite *GrpcTestSuite) SetupSuite() {
	suite.invoiceChan = make(chan (*lndhubrpc.Invoice))

	mlnd := newDefaultMockLND()
	svc, err := LndHubTestServiceInit(mlnd)
	svc.Config.GRPCPort = 10009
	suite.mlnd = mlnd
	if err != nil {
		log.Fatalf("Error initializing test service: %v", err)
	}

	users, userTokens, err := createUsers(svc, 1)
	if err != nil {
		log.Fatalf("Error creating test users: %v", err)
	}
	// Subscribe to LND invoice updates in the background
	// store cancel func to be called in tear down suite
	ctx, cancel := context.WithCancel(context.Background())
	suite.invoiceUpdateSubCancelFn = cancel
	go svc.InvoiceUpdateSubscription(ctx)

	//start grpc server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", svc.Config.GRPCPort))
	if err != nil {
		svc.Logger.Fatalf("Failed to start grpc server: %v", err)
	}
	grpcServer := svc.NewGrpcServer(ctx)
	go func() {
		err = grpcServer.Serve(lis)
		if err != nil {
			svc.Logger.Error(err)
		}
	}()

	go StartGrpcClient(ctx, svc.Config.GRPCPort, suite.invoiceChan)

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
func (suite *GrpcTestSuite) TestGrpc() {
	// create incoming invoice and fund account
	invoice := suite.createAddInvoiceReq(1000, "integration test grpc", suite.userToken)
	err := suite.mlnd.mockPaidInvoice(invoice, 0, false, nil)
	assert.NoError(suite.T(), err)
	invoiceFromClient := <-suite.invoiceChan
	assert.Equal(suite.T(), "integration test grpc", invoiceFromClient.Memo)
	assert.Equal(suite.T(), common.InvoiceTypeIncoming, invoiceFromClient.Type)
}

func StartGrpcClient(ctx context.Context, port int, invoiceChan chan (*lndhubrpc.Invoice)) error {
	addr := fmt.Sprintf("localhost:%d", port)
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()
	c := lndhubrpc.NewInvoiceSubscriptionClient(conn)
	r, err := c.SubsribeInvoices(context.Background(), &lndhubrpc.SubsribeInvoicesRequest{})
	if err != nil {
		return err
	}
	for {
		result, err := r.Recv()
		if err != nil {
			return err
		}
		invoiceChan <- result
	}
}

func (suite *GrpcTestSuite) TearDownSuite() {
	suite.invoiceUpdateSubCancelFn()
	clearTable(suite.service, "invoices")
}

func TestGrpcSuite(t *testing.T) {
	suite.Run(t, new(GrpcTestSuite))
}
