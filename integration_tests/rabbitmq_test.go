package integration_tests

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
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
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/lightningnetwork/lnd/lnrpc"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type RabbitMQTestSuite struct {
	TestSuite
	mlnd                     *MockLND
	externalLnd              *MockLND
	invoiceUpdateSubCancelFn context.CancelFunc
	userToken                string
	svc                      *service.LndhubService
	testQueueName            string
}

func (suite *RabbitMQTestSuite) SetupSuite() {
	mlnd := newDefaultMockLND()
	//needs different pubkey
	//to allow for "external" payments
	externalLnd, err := NewMockLND("1234567890abcdef1234", 0, make(chan (*lnrpc.Invoice)))
	assert.NoError(suite.T(), err)
	svc, err := LndHubTestServiceInit(mlnd)
	if err != nil {
		log.Fatalf("could not initialize test service: %v", err)
	}

	suite.mlnd = mlnd
	suite.externalLnd = externalLnd
	suite.testQueueName = "test_invoice"

	_, userTokens, err := createUsers(svc, 1)
	if err != nil {
		log.Fatalf("error creating test users: %v", err)
	}
	suite.userToken = userTokens[0]

	ctx, cancel := context.WithCancel(context.Background())
	suite.invoiceUpdateSubCancelFn = cancel
	suite.svc = svc

	e := echo.New()
	e.HTTPErrorHandler = responses.HTTPErrorHandler
	e.Validator = &lib.CustomValidator{Validator: validator.New()}

	suite.echo = e
	suite.echo.Use(tokens.Middleware(suite.svc.Config.JWTSecret))
	suite.echo.POST("/addinvoice", controllers.NewAddInvoiceController(suite.svc).AddInvoice)
	suite.echo.POST("/payinvoice", controllers.NewPayInvoiceController(suite.svc).PayInvoice)
	go func() {
		err = svc.RabbitMQClient.StartPublishInvoices(ctx, svc.SubscribeIncomingOutgoingInvoices, svc.AddInvoiceMetadata)
		assert.NoError(suite.T(), err)
	}()
}

func (suite *RabbitMQTestSuite) TestConsumeAndPublishInvoice() {
	conn, err := amqp.Dial(suite.svc.Config.RabbitMQUri)
	assert.NoError(suite.T(), err)
	defer conn.Close()

	ch, err := conn.Channel()
	assert.NoError(suite.T(), err)

	//listen to outgoing lndhub invoice channel to test e2e
	q, err := ch.QueueDeclare(
		suite.testQueueName,
		true,
		false,
		false,
		false,
		nil,
	)
	assert.NoError(suite.T(), err)

	err = ch.QueueBind(q.Name, "#", suite.svc.Config.RabbitMQLndhubInvoiceExchange, false, nil)
	assert.NoError(suite.T(), err)
	defer ch.Close()
	err = ch.ExchangeDeclare(
		suite.svc.Config.RabbitMQLndInvoiceExchange,
		// topic is a type of exchange that allows routing messages to different queue's bases on a routing key
		"topic",
		// Durable and Non-Auto-Deleted exchanges will survive server restarts and remain
		// declared when there are no remaining bindings.
		true,
		false,
		// Non-Internal exchange's accept direct publishing
		false,
		// Nowait: We set this to false as we want to wait for a server response
		// to check whether the exchange was created succesfully
		false,
		nil,
	)
	assert.NoError(suite.T(), err)

	go func() {
		err = suite.svc.RabbitMQClient.SubscribeToLndInvoices(context.Background(), suite.svc.ProcessInvoiceUpdate)
		assert.NoError(suite.T(), err)
	}()
	time.Sleep(100 * time.Millisecond)

	//create payload
	invoice := suite.createAddInvoiceReq(1000, "integration test rabbitmq", suite.userToken)
	hash, err := hex.DecodeString(invoice.RHash)
	assert.NoError(suite.T(), err)
	payload := &lnrpc.Invoice{
		RHash:      hash,
		AmtPaidSat: 1000,
		Settled:    true,
		SettleDate: time.Now().Unix(),
	}
	payloadBytes := new(bytes.Buffer)
	err = json.NewEncoder(payloadBytes).Encode(payload)
	assert.NoError(suite.T(), err)
	err = ch.Publish(suite.svc.Config.RabbitMQLndInvoiceExchange, "invoice.incoming.settled", false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        payloadBytes.Bytes(),
	})
	assert.NoError(suite.T(), err)

	m, err := ch.Consume(
		q.Name,
		"invoice.*.*",
		true,
		false,
		false,
		false,
		nil,
	)
	assert.NoError(suite.T(), err)

	msg := <-m

	var receivedInvoice models.WebhookInvoicePayload
	r := bytes.NewReader(msg.Body)
	err = json.NewDecoder(r).Decode(&receivedInvoice)
	assert.NoError(suite.T(), err)

	assert.Equal(suite.T(), invoice.RHash, receivedInvoice.RHash)
	assert.Equal(suite.T(), common.InvoiceTypeIncoming, receivedInvoice.Type)
	assert.Equal(suite.T(), int64(1000), receivedInvoice.Balance)
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

	err = ch.QueueBind(q.Name, "#", suite.svc.Config.RabbitMQLndhubInvoiceExchange, false, nil)
	assert.NoError(suite.T(), err)

	go suite.svc.InvoiceUpdateSubscription(context.Background())
	time.Sleep(100 * time.Microsecond)
	invoice := suite.createAddInvoiceReq(1000, "integration test rabbitmq", suite.userToken)
	err = suite.mlnd.mockPaidInvoice(invoice, 0, false, nil)
	assert.NoError(suite.T(), err)

	m, err := ch.Consume(
		q.Name,
		"invoice.*.*",
		true,
		false,
		false,
		false,
		nil,
	)
	assert.NoError(suite.T(), err)

	msg := <-m

	var receivedInvoice models.Invoice
	r := bytes.NewReader(msg.Body)
	err = json.NewDecoder(r).Decode(&receivedInvoice)
	assert.NoError(suite.T(), err)

	assert.Equal(suite.T(), invoice.RHash, receivedInvoice.RHash)
	assert.Equal(suite.T(), common.InvoiceTypeIncoming, receivedInvoice.Type)

	//check if outgoing invoices also get published
	outgoingInvoiceValue := 500
	outgoingInvoiceDescription := "test rabbit outgoing invoice"
	preimage, err := makePreimageHex()
	assert.NoError(suite.T(), err)
	outgoingInv, err := suite.externalLnd.AddInvoice(context.Background(), &lnrpc.Invoice{Value: int64(outgoingInvoiceValue), Memo: outgoingInvoiceDescription, RPreimage: preimage})
	assert.NoError(suite.T(), err)
	//pay invoice
	suite.createPayInvoiceReq(&ExpectedPayInvoiceRequestBody{
		Invoice: outgoingInv.PaymentRequest,
	}, suite.userToken)
	msg = <-m

	var receivedPayment models.Invoice
	r = bytes.NewReader(msg.Body)
	err = json.NewDecoder(r).Decode(&receivedPayment)
	assert.NoError(suite.T(), err)

	assert.Equal(suite.T(), common.InvoiceTypeOutgoing, receivedPayment.Type)
	assert.Equal(suite.T(), int64(outgoingInvoiceValue), receivedPayment.Amount)
	assert.Equal(suite.T(), outgoingInvoiceDescription, receivedPayment.Memo)

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

	err = ch.ExchangeDelete(suite.svc.Config.RabbitMQLndhubInvoiceExchange, true, false)
	assert.NoError(suite.T(), err)
}

func TestRabbitMQTestSuite(t *testing.T) {
	suite.Run(t, new(RabbitMQTestSuite))
}
