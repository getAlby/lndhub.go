package rabbitmq_test

import (
	"context"
	"encoding/json"
	"sync"
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

	ch := make(chan amqp.Delivery, 1)
	amqpClient.EXPECT().
		Listen(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		MaxTimes(1).
		Return(ch, nil)

    hash := "69e5f0f0590be75e30f671d56afe1d55"

	invoices := []models.Invoice{
		{
			ID: 0,
            RHash: hash,
		},
	}

	lndHubService.EXPECT().
		GetAllPendingPayments(gomock.Any()).
		MaxTimes(1).
		Return(invoices, nil)

	lndHubService.EXPECT().
		HandleFailedPayment(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		AnyTimes().
		Return(nil)

	lndHubService.EXPECT().
		HandleSuccessfulPayment(gomock.Any(), gomock.Any(), gomock.Any()).
		AnyTimes().
		Return(nil)

	lndHubService.EXPECT().
		GetTransactionEntryByInvoiceId(gomock.Any(), gomock.Eq(invoices[0].ID)).
		AnyTimes().
		Return(models.TransactionEntry{InvoiceID: invoices[0].ID}, nil)

	ctx := context.Background()
    b, err := json.Marshal(&lnrpc.Payment{PaymentHash: hash, Status: lnrpc.Payment_SUCCEEDED})
    if err != nil {
        t.Error(err)
    }

    ch <- amqp.Delivery{Body: b}

    wg := sync.WaitGroup{}

    wg.Add(1)
    go func() {
        err = client.FinalizeInitializedPayments(ctx, lndHubService)

        assert.NoError(t, err)
        wg.Done()
    }()

    waitTimeout(&wg, time.Second * 3, t)
}

// waitTimeout waits for the waitgroup for the specified max timeout.
// Returns true if waiting timed out.
func waitTimeout(wg *sync.WaitGroup, timeout time.Duration, t *testing.T) bool {
    c := make(chan struct{})
    go func() {
        defer close(c)
        wg.Wait()
    }()

    select {
    case <-c:
        return false // completed normally

    case <-time.After(timeout):
        t.Errorf("Waiting on waitgroup timed out during test")

        return true // timed out
    }
}
