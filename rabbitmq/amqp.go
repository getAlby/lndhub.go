package rabbitmq

import (
	"context"
	"os"
	"time"

	"github.com/labstack/gommon/log"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/ziflex/lecho/v3"
)

const (
	defaultHeartbeat = 10 * time.Second
	defaultLocale    = "en_US"
)

type AMQPClient interface {
	Listen(ctx context.Context, exchange string, routingKey string, queueName string, options ...AMQPListenOptions) (<-chan amqp.Delivery, error)
	PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
	ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error
	Close() error
}

type defaultAMQPCLient struct {
	conn *amqp.Connection
	uri  string

	// It is recommended that, when possible, publishers and consumers
	// use separate connections so that consumers are isolated from potential
	// flow control measures that may be applied to publishing connections.
	consumeChannel *amqp.Channel
	publishChannel *amqp.Channel

	notifyCloseChan chan *amqp.Error

	logger *lecho.Logger
}

type DialOption = func(amqp.Config) amqp.Config

func DialAMQP(uri string) (AMQPClient, error) {
	client := &defaultAMQPCLient{
		uri: uri,
		logger: lecho.New(
			os.Stdout,
			lecho.WithLevel(log.DEBUG),
			lecho.WithTimestamp(),
		),
	}

	err := client.connect()
	return client, err
}

func (c *defaultAMQPCLient) connect() error {
	conn, err := amqp.DialConfig(c.uri, amqp.Config{
		Heartbeat: defaultHeartbeat,
		Locale:    defaultLocale,
		Dial:      amqp.DefaultDial(time.Second * 3),
	})
	if err != nil {
		return err
	}

	consumeChannel, err := conn.Channel()
	if err != nil {
		return err
	}

	publishChannel, err := conn.Channel()
	if err != nil {
		return err
	}

	notifyCloseChan := make(chan *amqp.Error)
	conn.NotifyClose(notifyCloseChan)

	c.conn = conn
	c.consumeChannel = consumeChannel
	c.publishChannel = publishChannel
	c.notifyCloseChan = notifyCloseChan

	return nil
}

func (c *defaultAMQPCLient) Close() error {
    close(c.notifyCloseChan)
	return c.conn.Close()
}

func (c *defaultAMQPCLient) ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error {
    // For now we simply create a short lived channel. If this proves to be a bad approach we can either create a management channel
    // at client create time, or use either the consumer/publishing channels that already exist.
	ch, err := c.conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	return ch.ExchangeDeclare(name, kind, durable, autoDelete, internal, noWait, args)
}

type ListenOptions struct {
	Durable    bool
	AutoDelete bool
	Internal   bool
	Wait       bool
	Exclusive  bool
	AutoAck    bool
}

type AMQPListenOptions = func(opts ListenOptions) ListenOptions

func WithDurable(durable bool) AMQPListenOptions {
	return func(opts ListenOptions) ListenOptions {
		opts.Durable = durable
		return opts
	}
}

func WithAutoDelete(autoDelete bool) AMQPListenOptions {
	return func(opts ListenOptions) ListenOptions {
		opts.AutoDelete = autoDelete
		return opts
	}
}

func WithInternal(internal bool) AMQPListenOptions {
	return func(opts ListenOptions) ListenOptions {
		opts.Internal = internal
		return opts
	}
}

func WithWait(wait bool) AMQPListenOptions {
	return func(opts ListenOptions) ListenOptions {
		opts.Wait = wait
		return opts
	}
}

func WithExclusive(exclusive bool) AMQPListenOptions {
	return func(opts ListenOptions) ListenOptions {
		opts.Exclusive = exclusive
		return opts
	}
}

func WithAutoAck(autoAck bool) AMQPListenOptions {
	return func(opts ListenOptions) ListenOptions {
		opts.AutoAck = autoAck
		return opts
	}
}

func (c *defaultAMQPCLient) Listen(ctx context.Context, exchange string, routingKey string, queueName string, options ...AMQPListenOptions) (<-chan amqp.Delivery, error) {
	deliveries, err := c.consume(ctx, exchange, routingKey, queueName, options...)
	if err != nil {
		return nil, err
	}

	clientChannel := make(chan amqp.Delivery)

    // This routine functions as a wrapper arround the "raw" delivery channel.
    // The happy-path of the select statement, i.e. the last one, is to simply
    // pass on the message we get from the actual amqp channel. If however, an
    // error is send over the NotifyClose channel it means we must try to
    // reconnect if the error is Recoverable. In the meantime the client using
    // the Listen function is non the wiser that this happened. A successful
    // reconnect will make sure we recieve message from a new "raw" delivery
    // channel on the next loop we simply keep sending new messages to the
    // client channel using this new underlying connection/channel.
	go func() {
		for {
			select {
			case <-ctx.Done():
				c.Close()
				return

			case amqpError := <-c.notifyCloseChan:
				c.logger.Error(amqpError.Error())
				if !amqpError.Recover {
                    c.Close()
					return
				}

				c.logger.Info("amqp: trying to reconnect...")

				err := c.connect()
				if err != nil {
					c.logger.Error(err)
                    c.Close()

					return
				}

				d, err := c.consume(ctx, exchange, routingKey, queueName, options...)
				if err != nil {
					c.logger.Error(err)
                    c.Close()

					return
				}

				deliveries = d

			case delivery := <-deliveries:
				clientChannel <- delivery
			}
		}
	}()

	return clientChannel, nil
}

func (c *defaultAMQPCLient) consume(ctx context.Context, exchange string, routingKey string, queueName string, options ...AMQPListenOptions) (<-chan amqp.Delivery, error) {
	opts := ListenOptions{
		Durable:    true,
		AutoDelete: false,
		Internal:   false,
		Wait:       false,
		Exclusive:  false,
		AutoAck:    false,
	}

	for _, opt := range options {
		opts = opt(opts)
	}

	err := c.consumeChannel.ExchangeDeclare(
		exchange,
		// topic is a type of exchange that allows routing messages to different queue's bases on a routing key
		"topic",
		// Durable and Non-Auto-Deleted exchanges will survive server restarts and remain
		// declared when there are no remaining bindings.
		opts.Durable,
		opts.AutoDelete,
		// Non-Internal exchange's accept direct publishing
		opts.Internal,
		// Nowait: We set this to false as we want to wait for a server response
		// to check whether the exchange was created succesfully
		opts.Wait,
		nil,
	)
	if err != nil {
		return nil, err
	}

	queue, err := c.consumeChannel.QueueDeclare(
		queueName,
		// Durable and Non-Auto-Deleted queues will survive server restarts and remain
		// declared when there are no remaining bindings.
		opts.Durable,
		opts.AutoDelete,
		// None-Exclusive means other consumers can consume from this queue.
		// Messages from queues are spread out and load balanced between consumers.
		// So multiple lndhub.go instances will spread the load of invoices between them
		opts.Exclusive,
		// Nowait: We set this to false as we want to wait for a server response
		// to check whether the queue was created successfully
		opts.Wait,
		nil,
	)
	if err != nil {
		return nil, err
	}

	err = c.consumeChannel.QueueBind(
		queue.Name,
		routingKey,
		exchange,
		// Nowait: We set this to false as we want to wait for a server response
		// to check whether the queue was created successfully
		opts.Wait,
		nil,
	)
	if err != nil {
		return nil, err
	}

	return c.consumeChannel.Consume(
		queue.Name,
		"",
		opts.AutoAck,
		opts.Exclusive,
		false,
		opts.Wait,
		nil,
	)
}

func (c *defaultAMQPCLient) PublishWithContext(ctx context.Context, exchange string, key string, mandatory bool, immediate bool, msg amqp.Publishing) error {
    // TODO: Think about race condition here. When a connection retry is in progress the publishing channel will get reassigned as well.
	return c.publishChannel.PublishWithContext(ctx, exchange, key, mandatory, immediate, msg)
}
