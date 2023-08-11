package rabbitmq_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/rabbitmq"
	"github.com/getAlby/lndhub.go/rabbitmq/mock_rabbitmq"
	"github.com/golang/mock/gomock"
	"github.com/lightningnetwork/lnd/lnrpc"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
)

//go:generate mockgen -destination=./mock_rabbitmq/rabbitmq.go github.com/getAlby/lndhub.go/rabbitmq LndHubService,AMQPClient

func TestFinalizedInitializedPayments(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lndHubService := mock_rabbitmq.NewMockLndHubService(ctrl)
	amqpClient := mock_rabbitmq.NewMockAMQPClient(ctrl)

	client, err := rabbitmq.NewClient(amqpClient)
	assert.NoError(t, err)

	ch := make(chan amqp.Delivery, 2)
	amqpClient.EXPECT().
		Listen(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		Return(ch, nil)

	firstHash := "69e5f0f0590be75e30f671d56afe1d55"
	secondHash := "ffff0f0590be75e30f671d56afe1d55"

	invoices := []models.Invoice{
		{
			ID:    0,
			RHash: firstHash,
		},
		{
			ID:    1,
			RHash: secondHash,
		},
	}

	lndHubService.EXPECT().
		GetAllPendingPayments(gomock.Any()).
		Times(1).
		Return(invoices, nil)

	lndHubService.EXPECT().
		HandleSuccessfulPayment(gomock.Any(), gomock.Eq(&invoices[0]), gomock.Any()).
		Times(1).
		Return(nil)

	lndHubService.EXPECT().
		HandleFailedPayment(gomock.Any(), gomock.Eq(&invoices[1]), gomock.Any(), gomock.Any()).
		Times(1).
		Return(nil)

	lndHubService.EXPECT().
		GetTransactionEntryByInvoiceId(gomock.Any(), gomock.Eq(invoices[0].ID)).
		AnyTimes().
		Return(models.TransactionEntry{InvoiceID: invoices[0].ID}, nil)

	lndHubService.EXPECT().
		GetTransactionEntryByInvoiceId(gomock.Any(), gomock.Eq(invoices[1].ID)).
		AnyTimes().
		Return(models.TransactionEntry{InvoiceID: invoices[1].ID}, nil)

	ctx := context.Background()
	successPayment, err := json.Marshal(&lnrpc.Payment{PaymentHash: firstHash, Status: lnrpc.Payment_SUCCEEDED})
	if err != nil {
		t.Error(err)
	}

	failedPayment, err := json.Marshal(&lnrpc.Payment{PaymentHash: secondHash, Status: lnrpc.Payment_FAILED})
	if err != nil {
		t.Error(err)
	}

	ch <- amqp.Delivery{Body: successPayment}
	ch <- amqp.Delivery{Body: failedPayment}


	go func() {
		err = client.FinalizeInitializedPayments(ctx, lndHubService)

		assert.NoError(t, err)
	}()

	//wait a bit for payments to be processed
	time.Sleep(time.Second)
}
