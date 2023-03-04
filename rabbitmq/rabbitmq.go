package rabbitmq

import (
	"context"
	"encoding/json"
	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getsentry/sentry-go"
	"github.com/labstack/gommon/log"
	"github.com/lightningnetwork/lnd/lnrpc"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/ziflex/lecho/v3"
	"os"
)

type Client interface {
	SubscribeToLndInvoices(context.Context, InvoiceHandler) error
	StartPublishInvoices(context.Context, SubscribeToInvoicesFunc) error
	Close() error
}

type DefaultClient struct {
	conn *amqp.Connection

	// It is recommended that, when possible, publishers and consumers
	// use separate connections so that consumers are isolated from potential
	// flow control measures that may be applied to publishing connections.
	consumeChannel *amqp.Channel
	produceChannel *amqp.Channel

	logger *lecho.Logger

	lndInvoiceConsumerQueueName string
	lndInvoiceExchange          string
	lndhubInvoiceExchange       string
}

type ClientOption = func(client *DefaultClient)

func WithLndInvoiceExchange(exchange string) ClientOption {
	return func(client *DefaultClient) {
		client.lndInvoiceExchange = exchange
	}
}

func WithLndhubInvoiceExchange(exchange string) ClientOption {
	return func(client *DefaultClient) {
		client.lndhubInvoiceExchange = exchange
	}
}

func WithLndInvoiceConsumerQueueName(name string) ClientOption {
	return func(client *DefaultClient) {
		client.lndInvoiceConsumerQueueName = name
	}
}

func WithLogger(logger *lecho.Logger) ClientOption {
	return func(client *DefaultClient) {
		client.logger = logger
	}
}

func Dial(uri string, options ...ClientOption) (Client, error) {
	conn, err := amqp.Dial(uri)
	if err != nil {
		return nil, err
	}

	consumeChannel, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	produceChannel, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	client := &DefaultClient{
		conn: conn,

		consumeChannel: consumeChannel,
		produceChannel: produceChannel,

		logger: lecho.New(
			os.Stdout,
			lecho.WithLevel(log.DEBUG),
			lecho.WithTimestamp(),
		),

		lndInvoiceConsumerQueueName: "lndhub_invoice_consumer",
		lndInvoiceExchange:          "lnd_invoice",
		lndhubInvoiceExchange:       "lndhub_invoice",
	}

	for _, opt := range options {
		opt(client)
	}

	return client, nil
}

func (client *DefaultClient) Close() error { return client.conn.Close() }

type InvoiceHandler = func(ctx context.Context, invoice *lnrpc.Invoice) error

func (client *DefaultClient) SubscribeToLndInvoices(ctx context.Context, handler InvoiceHandler) error {
	queue, err := client.consumeChannel.QueueDeclare(
		client.lndInvoiceConsumerQueueName,
		// Durable and Non-Auto-Deleted queues will survive server restarts and remain
		// declared when there are no remaining bindings.
		true,
		false,
		// None-Exclusive means other consumers can consume from this queue.
		// Messages from queues are spread out and load balanced between consumers.
		// So multiple lndhub.go instances will spread the load of invoices between them
		false,
		// Nowait: We set this to false as we want to wait for a server response
		// to check whether the queue was created successfully
		false,
		nil,
	)
	if err != nil {
		return err
	}

	err = client.consumeChannel.QueueBind(
		queue.Name,
		"#",
		client.lndInvoiceExchange,
		// Nowait: We set this to false as we want to wait for a server response
		// to check whether the queue was created successfully
		false,
		nil,
	)
	if err != nil {
		return err
	}

	deliveryChan, err := client.consumeChannel.Consume(
		queue.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return context.Canceled
		case delivery := <-deliveryChan:
			var invoice lnrpc.Invoice

			err := json.Unmarshal(delivery.Body, &invoice)
			if err != nil {
				client.logger.Error(err)
				sentry.CaptureException(err)

				err = delivery.Nack(false, false)
				if err != nil {
					client.logger.Error(err)
					sentry.CaptureException(err)
				}

				continue
			}

			err = handler(ctx, &invoice)
			if err != nil {
				client.logger.Error(err)
				sentry.CaptureException(err)

				delivery.Nack(false, false)
				continue
			}

			delivery.Ack(false)
		}
	}
}

type SubscribeToInvoicesFunc = func() (in chan models.Invoice, out chan models.Invoice, err error)

func (client *DefaultClient) StartPublishInvoices(ctx context.Context, invoicesSubscribeFunc SubscribeToInvoicesFunc) error {
	err := client.produceChannel.ExchangeDeclare(
		client.lndInvoiceExchange,
		// topic is a type of exchange that allows routing messages to different queue's bases on a routing key
		"topic",
		// Durable and Non-Auto-Deleted exchanges will survive server restarts and remain
		// declared when there are no remaining bindings.
		true,
		false,
		// Non-Internal exchange's accept direct publishing
		false,
		// Nowait: We set this to false as we want to wait for a server response
		// to check wether the exchange was created succesfully
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
			err = client.publishToLndhubExchange(ctx, incomingInvoice)
			if err != nil {
				client.logger.Error(err)
				sentry.CaptureException(err)
			}
		case outgoing := <-out:
			err = client.publishToLndhubExchange(ctx, outgoing)
			if err != nil {
				client.logger.Error(err)
				sentry.CaptureException(err)
			}
		}
	}
}

func (client *DefaultClient) publishToLndhubExchange(ctx context.Context, invoice models.Invoice) error {
	return nil
}
