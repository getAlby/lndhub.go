package rabbitmq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getsentry/sentry-go"
	"github.com/labstack/gommon/log"
	"github.com/lightningnetwork/lnd/lnrpc"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/ziflex/lecho/v3"
)

// bufPool is a classic buffer pool pattern that allows more clever reuse of heap memory.
// Instead of allocating new memory everytime we need to encode the invoices we
// reuse buffers from this buffer pool. If we consume events sequentially there will
// only be one buffer in this pool at all times, but when scaling to multiple go
// routines this memory pool will scale with it.
var bufPool = sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

const (
	contentTypeJSON            = "application/json"
	outgoingPaymentsRoutingKey = "payment.outgoing.*"
)

type (
	IncomingInvoiceHandler    = func(ctx context.Context, invoice *lnrpc.Invoice) error
	SubscribeToInvoicesFunc   = func() (in chan models.Invoice, out chan models.Invoice, err error)
	EncodeOutgoingInvoiceFunc = func(ctx context.Context, w io.Writer, invoice models.Invoice) error
)

type Client interface {
	SubscribeToLndInvoices(context.Context, IncomingInvoiceHandler) error
	StartPublishInvoices(context.Context, SubscribeToInvoicesFunc, EncodeOutgoingInvoiceFunc) error
	FinalizeInitializedPayments(context.Context, LndHubService) error
	// Close will close all connections to rabbitmq
	Close() error
}

type ClientConfig struct {
	lndInvoiceConsumerQueueName string
	lndPaymentConsumerQueueName string
	lndInvoiceExchange          string
	lndPaymentExchange          string
	lndHubInvoiceExchange       string
}

type LndHubService interface {
	HandleFailedPayment(context.Context, *models.Invoice, models.TransactionEntry, error) error
	HandleSuccessfulPayment(context.Context, *models.Invoice, models.TransactionEntry) error
	GetAllPendingPayments(context.Context) ([]models.Invoice, error)
	GetTransactionEntryByInvoiceId(context.Context, int64) (models.TransactionEntry, error)
}

type DefaultClient struct {
	amqpClient AMQPClient
	logger     *lecho.Logger

	config ClientConfig
}

type ClientOption = func(client *DefaultClient)

func WithLndInvoiceExchange(exchange string) ClientOption {
	return func(client *DefaultClient) {
		client.config.lndInvoiceExchange = exchange
	}
}

func WithLndHubInvoiceExchange(exchange string) ClientOption {
	return func(client *DefaultClient) {
		client.config.lndHubInvoiceExchange = exchange
	}
}

func WithLndInvoiceConsumerQueueName(name string) ClientOption {
	return func(client *DefaultClient) {
		client.config.lndInvoiceConsumerQueueName = name
	}
}

func WithLndPaymentConsumerQueueName(name string) ClientOption {
	return func(client *DefaultClient) {
		client.config.lndPaymentConsumerQueueName = name
	}
}

func WithLndPaymentExchange(exchange string) ClientOption {
	return func(client *DefaultClient) {
		client.config.lndPaymentExchange = exchange
	}
}

func WithLogger(logger *lecho.Logger) ClientOption {
	return func(client *DefaultClient) {
		client.logger = logger
	}
}

// Dial sets up a connection to rabbitmq with two channels that are ready to produce and consume
func NewClient(amqpClient AMQPClient, options ...ClientOption) (Client, error) {
	client := &DefaultClient{
		amqpClient: amqpClient,

		logger: lecho.New(
			os.Stdout,
			lecho.WithLevel(log.DEBUG),
			lecho.WithTimestamp(),
		),

		config: ClientConfig{
			lndInvoiceConsumerQueueName: "lnd_invoice_consumer",
			lndPaymentConsumerQueueName: "lnd_payment_consumer",
			lndInvoiceExchange:          "lnd_invoice",
			lndPaymentExchange:          "lnd_payment",
			lndHubInvoiceExchange:       "lndhub_invoice",
		},
	}

	for _, opt := range options {
		opt(client)
	}

	return client, nil
}

func (client *DefaultClient) Close() error { return client.amqpClient.Close() }

func (client *DefaultClient) FinalizeInitializedPayments(ctx context.Context, svc LndHubService) error {
	deliveryChan, err := client.amqpClient.Listen(
		ctx,
		client.config.lndPaymentExchange,
		outgoingPaymentsRoutingKey,
		client.config.lndPaymentConsumerQueueName,
	)
	if err != nil {
		return err
	}

	getInvoicesTable := func(ctx context.Context) (map[string]models.Invoice, error) {
		invoicesByHash := map[string]models.Invoice{}
		pendingInvoices, err := svc.GetAllPendingPayments(ctx)

		if err != nil {
			return invoicesByHash, err
		}

		for _, invoice := range pendingInvoices {
			invoicesByHash[invoice.RHash] = invoice
		}
		return invoicesByHash, nil
	}

	pendingInvoices, err := getInvoicesTable(ctx)
	if err != nil {
		return err
	}

	client.logger.Infof("Payment finalizer: Found %d pending invoices", len(pendingInvoices))

	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	client.logger.Info("Starting payment finalizer rabbitmq consumer")

	for {
		// Shortcircuit if no pending invoices are left
		if len(pendingInvoices) == 0 {
			client.logger.Info("Payment finalizer: Resolved all pending payments, exiting payment finalizer routine")

			return nil
		}

		select {
		case <-ctx.Done():
			return context.Canceled

		case <-ticker.C:
			invoices, err := getInvoicesTable(ctx)
			if err != nil {
				return err
			}

			pendingInvoices = invoices

			client.logger.Infof("Payment finalizer: Found %d pending invoices", len(pendingInvoices))

		case delivery, ok := <-deliveryChan:
			if !ok {
				return fmt.Errorf("Disconnected from RabbitMQ")
			}

			payment := lnrpc.Payment{}

			err := json.Unmarshal(delivery.Body, &payment)
			if err != nil {
				delivery.Nack(false, false)

				continue
			}

			// Check if paymentHash corresponds to one of the pending invoices
			if invoice, ok := pendingInvoices[payment.PaymentHash]; ok {
				t, err := svc.GetTransactionEntryByInvoiceId(ctx, invoice.ID)
				if err != nil {
					captureErr(client.logger, err)
					delivery.Nack(false, false)

					continue
				}

				switch payment.Status {
				case lnrpc.Payment_SUCCEEDED:
					invoice.Fee = payment.FeeSat
					invoice.Preimage = payment.PaymentPreimage

					if err = svc.HandleSuccessfulPayment(ctx, &invoice, t); err != nil {
						captureErr(client.logger, err)
						delivery.Nack(false, false)

						continue
					}

					client.logger.Infof("Payment finalizer: updated successful payment with hash: %s", payment.PaymentHash)
					delete(pendingInvoices, payment.PaymentHash)

				case lnrpc.Payment_FAILED:
					if err = svc.HandleFailedPayment(ctx, &invoice, t, fmt.Errorf(payment.FailureReason.String())); err != nil {
						captureErr(client.logger, err)
						delivery.Nack(false, false)

						continue
					}

					client.logger.Infof("Payment finalizer: updated failed payment with hash: %s", payment.PaymentHash)
					delete(pendingInvoices, payment.PaymentHash)
				}
			}
			delivery.Ack(false)
		}
	}
}

func (client *DefaultClient) SubscribeToLndInvoices(ctx context.Context, handler IncomingInvoiceHandler) error {
	deliveryChan, err := client.amqpClient.Listen(ctx, client.config.lndInvoiceExchange, "invoice.incoming.settled", client.config.lndInvoiceConsumerQueueName)
	if err != nil {
		return err
	}

	client.logger.Info("Starting RabbitMQ invoice consumer loop")
	for {
		select {
		case <-ctx.Done():
			return context.Canceled

		case delivery, ok := <-deliveryChan:
			if !ok {
				return fmt.Errorf("Disconnected from RabbitMQ")
			}
			var invoice lnrpc.Invoice

			err := json.Unmarshal(delivery.Body, &invoice)
			if err != nil {
				captureErr(client.logger, err)

				// If we can't even Unmarshall the message we are dealing with
				// badly formatted events. In that case we simply Nack the message
				// and explicitly do not requeue it.
				err = delivery.Nack(false, false)
				if err != nil {
					captureErr(client.logger, err)
				}

				continue
			}

			err = handler(ctx, &invoice)
			if err != nil {
				captureErr(client.logger, err)

				// If for some reason we can't handle the message we also don't requeue
				// because this can lead to an endless loop that puts pressure on the
				// database and logs.
				err := delivery.Nack(false, false)
				if err != nil {
					captureErr(client.logger, err)
				}

				continue
			}

			err = delivery.Ack(false)
			if err != nil {
				captureErr(client.logger, err)
			}
		}
	}
}

func (client *DefaultClient) StartPublishInvoices(ctx context.Context, invoicesSubscribeFunc SubscribeToInvoicesFunc, payloadFunc EncodeOutgoingInvoiceFunc) error {
	err := client.amqpClient.ExchangeDeclare(
		client.config.lndHubInvoiceExchange,
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
	if err != nil {
		return err
	}

	client.logger.Info("Starting rabbitmq publisher")

	in, out, err := invoicesSubscribeFunc()
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return context.Canceled
		case incomingInvoice := <-in:
			err = client.publishToLndhubExchange(ctx, incomingInvoice, payloadFunc)

			if err != nil {
				captureErr(client.logger, err)
			}
		case outgoing := <-out:
			err = client.publishToLndhubExchange(ctx, outgoing, payloadFunc)

			if err != nil {
				captureErr(client.logger, err)
			}
		}
	}
}

func (client *DefaultClient) publishToLndhubExchange(ctx context.Context, invoice models.Invoice, payloadFunc EncodeOutgoingInvoiceFunc) error {
	payload := bufPool.Get().(*bytes.Buffer)
	err := payloadFunc(ctx, payload, invoice)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("invoice.%s.%s", invoice.Type, invoice.State)

	err = client.amqpClient.PublishWithContext(ctx,
		client.config.lndHubInvoiceExchange,
		key,
		false,
		false,
		amqp.Publishing{
			ContentType: contentTypeJSON,
			Body:        payload.Bytes(),
		},
	)
	if err != nil {
		captureErr(client.logger, err)
		return err
	}

	client.logger.Debugf("Successfully published invoice to rabbitmq with RHash %s", invoice.RHash)

	return nil
}

func captureErr(logger *lecho.Logger, err error) {
	logger.Error(err)
	sentry.CaptureException(err)
}
