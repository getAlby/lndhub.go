package rabbitmq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
	"github.com/lightningnetwork/lnd/lnrpc"
	amqp "github.com/rabbitmq/amqp091-go"
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
	IncomingInvoiceHandler    = func(ctx context.Context, invoice *lnrpc.Invoice) error
	SubscribeToInvoicesFunc   = func() (in chan models.Invoice, out chan models.Invoice, err error)
	EncodeOutgoingInvoiceFunc = func(ctx context.Context, w io.Writer, invoice models.Invoice) error
)

type Client interface {
	SubscribeToLndInvoices(context.Context, IncomingInvoiceHandler) error
	StartPublishInvoices(context.Context, SubscribeToInvoicesFunc, EncodeOutgoingInvoiceFunc) error
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

	logger zerolog.Logger

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

func WithLogger(logger zerolog.Logger) ClientOption {
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

		logger: zerolog.New(os.Stdout).Level(zerolog.DebugLevel).With().Timestamp().Logger(),

		lndInvoiceConsumerQueueName: "lnd_invoice_consumer",
		lndInvoiceExchange:          "lnd_invoice",
		lndHubInvoiceExchange:       "lndhub_invoice",
	}

	for _, opt := range options {
		opt(client)
	}

	return client, nil
}

func (client *DefaultClient) Close() error { return client.conn.Close() }

func (client *DefaultClient) SubscribeToLndInvoices(ctx context.Context, handler IncomingInvoiceHandler) error {
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
		"invoice.incoming.settled",
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

	client.logger.Info().Msg("Starting RabbitMQ consumer loop")
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

	client.logger.Info().Msg("Starting rabbitmq publisher")

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

	client.logger.Debug().Msgf("Successfully published invoice to rabbitmq with RHash %s", invoice.RHash)

	return nil
}

func captureErr(logger zerolog.Logger, err error) {
	logger.Error().Err(err)
	sentry.CaptureException(err)
}
