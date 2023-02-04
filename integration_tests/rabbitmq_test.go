package integration_tests

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"testing"

	"github.com/getAlby/lndhub.go/controllers"
	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/getAlby/lndhub.go/lib/tokens"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type RabbitMQTestSuite struct {
	TestSuite
	mlnd                     *MockLND
	invoiceUpdateSubCancelFn context.CancelFunc
	userToken                string
	svc                      *service.LndhubService
	testQueueName            string
}

func (suite *RabbitMQTestSuite) SetupSuite() {
	mlnd := newDefaultMockLND()
	svc, err := LndHubTestServiceInit(mlnd)
	if err != nil {
		log.Fatalf("could not initialize test service: %v", err)
	}

	suite.mlnd = mlnd
	suite.testQueueName = "test_invoice"

	_, userTokens, err := createUsers(svc, 1)
	if err != nil {
		log.Fatalf("error creating test users: %v", err)
	}
	suite.userToken = userTokens[0]

	ctx, cancel := context.WithCancel(context.Background())
	suite.invoiceUpdateSubCancelFn = cancel
	go svc.InvoiceUpdateSubscription(ctx)
	suite.svc = svc

	e := echo.New()
	e.HTTPErrorHandler = responses.HTTPErrorHandler
	e.Validator = &lib.CustomValidator{Validator: validator.New()}

	suite.echo = e
	suite.echo.Use(tokens.Middleware(suite.svc.Config.JWTSecret))
	suite.echo.POST("/addinvoice", controllers.NewAddInvoiceController(suite.svc).AddInvoice)
	go svc.StartRabbitMqPublisher(ctx)
}

func (suite *RabbitMQTestSuite) TestPublishInvoice() {
	conn, err := amqp.Dial(suite.svc.Config.RabbitMQUri)
	assert.NoError(suite.T(), err)
	defer conn.Close()

	ch, err := conn.Channel()
	assert.NoError(suite.T(), err)
	defer ch.Close()

	q, err := ch.QueueDeclare(
		suite.testQueueName,
		true,
		false,
		false,
		false,
		nil,
	)
	assert.NoError(suite.T(), err)

	err = ch.QueueBind(q.Name, "#", suite.svc.Config.RabbitMQInvoiceExchange, false, nil)
	assert.NoError(suite.T(), err)

	invoice := suite.createAddInvoiceReq(1000, "integration test rabbitmq", suite.userToken)
	err = suite.mlnd.mockPaidInvoice(invoice, 0, false, nil)
	assert.NoError(suite.T(), err)

	m, err := ch.Consume(
		q.Name,
		"#.#.invoice",
		true,
		false,
		false,
		false,
		nil,
	)
	assert.NoError(suite.T(), err)

	msg := <-m

	var recievedInvoice models.Invoice
	r := bytes.NewReader(msg.Body)
	err = json.NewDecoder(r).Decode(&recievedInvoice)
	assert.NoError(suite.T(), err)

	assert.Equal(suite.T(), invoice.RHash, recievedInvoice.RHash)
}

func (suite *RabbitMQTestSuite) TearDownSuite() {
	conn, err := amqp.Dial(suite.svc.Config.RabbitMQUri)
	assert.NoError(suite.T(), err)
	defer conn.Close()

	ch, err := conn.Channel()
	assert.NoError(suite.T(), err)
	defer ch.Close()

	_, err = ch.QueueDelete(suite.testQueueName, false, false, false)
	assert.NoError(suite.T(), err)

	err = ch.ExchangeDelete(suite.svc.Config.RabbitMQInvoiceExchange, true, false)
	assert.NoError(suite.T(), err)
}

func TestRabbitMQTestSuite(t *testing.T) {
	suite.Run(t, new(RabbitMQTestSuite))
}
