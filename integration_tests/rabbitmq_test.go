package integration_tests

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"testing"

	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/lib/service"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
)

type RabbitMQTestSuite struct {
	TestSuite
	mlnd                     *MockLND
	invoiceUpdateSubCancelFn context.CancelFunc
	userToken                string
	svc                      *service.LndhubService
}

func (suite *RabbitMQTestSuite) SetupSuite() {
	mlnd := newDefaultMockLND()
	svc, err := LndHubTestServiceInit(mlnd)
	if err != nil {
		log.Fatalf("could not initialize test service: %v", err)
	}

	suite.mlnd = mlnd

	_, userTokens, err := createUsers(svc, 1)
	if err != nil {
		log.Fatalf("Error creating test users: %v", err)
	}
	suite.userToken = userTokens[0]

	ctx, cancel := context.WithCancel(context.Background())
	suite.invoiceUpdateSubCancelFn = cancel
	go svc.InvoiceUpdateSubscription(ctx)

	go svc.StartRabbitMqPublisher(ctx)
}

func (suite *RabbitMQTestSuite) TestPublishInvoice(t *testing.T) {
	// create incoming invoice and fund account
	invoice := suite.createAddInvoiceReq(1000, "integration test webhook", suite.userToken)
	err := suite.mlnd.mockPaidInvoice(invoice, 0, false, nil)
	assert.NoError(suite.T(), err)

	// Check consume from rabbit
	conn, err := amqp.Dial(suite.svc.Config.RabbitMQUri)
	if err != nil {
		t.Error(err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		t.Error(err)
	}

	q, err := ch.QueueDeclare(
		"test_invoice",
		false,
		true,
		false,
		false,
		nil,
	)
	if err != nil {
		t.Error(err)
	}

	m, err := ch.Consume(
		q.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		t.Error(err)
	}

	t.Log("Blocking on message channel")
	msg := <-m
	t.Logf("%s/n", string(msg.Body))

	var recievedInvoice models.Invoice
	r := bytes.NewReader(msg.Body)
	json.NewDecoder(r).Decode(&recievedInvoice)

	assert.Equal(t, invoice.RHash, recievedInvoice.RHash)
}
