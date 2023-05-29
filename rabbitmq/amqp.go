package rabbitmq

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

type AMQPClient interface {
	Listen(ctx context.Context, exchange string, routingKey string, queueName string, options ...AMQPListenOptions) (<-chan amqp.Delivery, error)
	PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
    ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error
    Close() error
}

type DefaultAMQPCLient struct {
	conn *amqp.Connection

	// It is recommended that, when possible, publishers and consumers
	// use separate connections so that consumers are isolated from potential
	// flow control measures that may be applied to publishing connections.
	consumeChannel *amqp.Channel
	publishChannel *amqp.Channel
}

func (c *DefaultAMQPCLient) Close() error { return c.conn.Close() }

func (c *DefaultAMQPCLient) ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error {
    // TODO: Seperate management channel? Or provide way to select channel?
    ch, err := c.conn.Channel()
    if err != nil {
        return err
    }
    defer ch.Close()


    return ch.ExchangeDeclare(name, kind, durable, autoDelete, internal, noWait, args)
}


type listenOptions struct {
	Durable    bool
	AutoDelete bool
	Internal   bool
	Wait       bool
	Exclusive  bool
	AutoAck    bool
}

type AMQPListenOptions = func(opts listenOptions) listenOptions

func WithDurable(durable bool) AMQPListenOptions {
	return func(opts listenOptions) listenOptions {
		opts.Durable = durable
		return opts
	}
}

func WithAutoDelete(autoDelete bool) AMQPListenOptions {
	return func(opts listenOptions) listenOptions {
		opts.AutoDelete = autoDelete
		return opts
	}
}

func WithInternal(internal bool) AMQPListenOptions {
	return func(opts listenOptions) listenOptions {
		opts.Internal = internal
		return opts
	}
}

func WithWait(wait bool) AMQPListenOptions {
	return func(opts listenOptions) listenOptions {
		opts.Wait = wait
		return opts
	}
}

func WithExclusive(exclusive bool) AMQPListenOptions {
	return func(opts listenOptions) listenOptions {
		opts.Exclusive = exclusive
		return opts
	}
}

func WithAutoAck(autoAck bool) AMQPListenOptions {
	return func(opts listenOptions) listenOptions {
		opts.AutoAck = autoAck
		return opts
	}
}

func (c *DefaultAMQPCLient) Listen(ctx context.Context, exchange string, routingKey string, queueName string, options ...AMQPListenOptions) (<-chan amqp.Delivery, error) {
	opts := listenOptions{
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

func (c *DefaultAMQPCLient) PublishWithContext(ctx context.Context, exchange string, key string, mandatory bool, immediate bool, msg amqp.Publishing) error {
	return c.publishChannel.PublishWithContext(ctx, exchange, key, mandatory, immediate, msg)
}

func Dial(uri string) (AMQPClient, error) {
	conn, err := amqp.Dial(uri)
	if err != nil {
		return nil, err
	}

	consumeChannel, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	publishChannel, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	return &DefaultAMQPCLient{
		conn,
		consumeChannel,
		publishChannel,
	}, nil
}
