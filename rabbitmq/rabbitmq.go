package rabbitmq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getsentry/sentry-go"
	"github.com/labstack/gommon/log"
	"github.com/lightningnetwork/lnd/lnrpc"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/ziflex/lecho/v3"
	"io"
	"os"
	"sync"
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
	contentTypeJSON = "application/json"
)

type (
	InvoiceHandler                  = func(ctx context.Context, invoice *lnrpc.Invoice) error
	SubscribeToInvoicesFunc         = func() (in chan models.Invoice, out chan models.Invoice, err error)
	EncodeWebhookInvoicePayloadFunc = func(ctx context.Context, w io.Writer, invoice models.Invoice) error
)

type Client interface {
	SubscribeToLndInvoices(context.Context, InvoiceHandler) error
	StartPublishInvoices(context.Context, SubscribeToInvoicesFunc, EncodeWebhookInvoicePayloadFunc) error
	// Close will close all connections to rabbitmq
	Close() error
}

type DefaultClient struct {
	conn *amqp.Connection

	// It is recommended that, when possible, publishers and consumers
	// use separate connections so that consumers are isolated from potential
	// flow control measures that may be applied to publishing connections.
	consumeChannel *amqp.Channel
	publishChannel *amqp.Channel

	logger *lecho.Logger

	lndInvoiceConsumerQueueName string
	lndInvoiceExchange          string
	lndHubInvoiceExchange       string
}

type ClientOption = func(client *DefaultClient)

func WithLndInvoiceExchange(exchange string) ClientOption {
	return func(client *DefaultClient) {
		client.lndInvoiceExchange = exchange
	}
}

func WithLndHubInvoiceExchange(exchange string) ClientOption {
	return func(client *DefaultClient) {
		client.lndHubInvoiceExchange = exchange
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

// Dial sets up a connection to rabbitmq with two channels that are ready to produce and consume
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
		publishChannel: produceChannel,

		logger: lecho.New(
			os.Stdout,
			lecho.WithLevel(log.DEBUG),
			lecho.WithTimestamp(),
		),

		lndInvoiceConsumerQueueName: "lndhub_invoice_consumer",
		lndInvoiceExchange:          "lnd_invoice",
		lndHubInvoiceExchange:       "lndhub_invoice",
	}

	for _, opt := range options {
		opt(client)
	}

	return client, nil
}

func (client *DefaultClient) Close() error { return client.conn.Close() }

func (client *DefaultClient) SubscribeToLndInvoices(ctx context.Context, handler InvoiceHandler) error {
	err := client.publishChannel.ExchangeDeclare(
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
		// to check whether the exchange was created succesfully
		false,
		nil,
	)
	if err != nil {
		return err
	}

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

				// If for some reason we can't handle the message we instruct rabbitmq to
				// requeue the message in hopes of finding another consumer that can deal
				// with this message.
				err := delivery.Nack(false, true)
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

func (client *DefaultClient) StartPublishInvoices(ctx context.Context, invoicesSubscribeFunc SubscribeToInvoicesFunc, payloadFunc EncodeWebhookInvoicePayloadFunc) error {
	err := client.publishChannel.ExchangeDeclare(
		client.lndHubInvoiceExchange,
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

func (client *DefaultClient) publishToLndhubExchange(ctx context.Context, invoice models.Invoice, payloadFunc EncodeWebhookInvoicePayloadFunc) error {
	payload := bufPool.Get().(*bytes.Buffer)
	err := payloadFunc(ctx, payload, invoice)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("invoice.%s.%s", invoice.Type, invoice.State)

	err = client.publishChannel.PublishWithContext(ctx,
		client.lndHubInvoiceExchange,
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
