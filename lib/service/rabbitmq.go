package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/getAlby/lndhub.go/db/models"
	amqp "github.com/rabbitmq/amqp091-go"
)

var bufPool sync.Pool = sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

func (svc *LndhubService) StartRabbitMqPublisher(ctx context.Context) error {
	// It is recommended that, when possible, publishers and consumers
	// use separate connections so that consumers are isolated from potential
	// flow control messures that may be applied to publishing connections.
	// We therefore start a single publishing connection here instead of storing
	// one on the service object.
	conn, err := amqp.Dial(svc.Config.RabbitMQUri)
	if err != nil {
		return err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	err = ch.ExchangeDeclare(
		// For the time being we simply declare a single exchange and start pushing to it.
		// Towards the future however this might become a more involved setup.
		svc.Config.RabbitMQInvoiceExchange,
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

	svc.Logger.Infof("Starting rabbitmq publisher")

	incomingInvoices, outgoingInvoices, err := svc.subscribeIncomingOutgoingInvoices()
	if err != nil {
		svc.Logger.Error(err)
	}
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled")
		case incoming := <-incomingInvoices:
			svc.publishInvoice(ctx, incoming, ch)
		case outgoing := <-outgoingInvoices:
			svc.publishInvoice(ctx, outgoing, ch)
		}
	}
}

func (svc *LndhubService) publishInvoice(ctx context.Context, invoice models.Invoice, ch *amqp.Channel) {
	key := fmt.Sprintf("%s.%s.invoice", invoice.Type, invoice.State)

	user, err := svc.FindUser(context.Background(), invoice.UserID)
	if err != nil {
		svc.Logger.Error(err)
		return
	}

	payload := bufPool.Get().(*bytes.Buffer)
	err = json.NewEncoder(payload).Encode(convertPayload(invoice, user))
	if err != nil {
		svc.Logger.Error(err)
		return
	}

	err = ch.PublishWithContext(ctx,
		svc.Config.RabbitMQInvoiceExchange,
		key,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        payload.Bytes(),
		},
	)
	if err != nil {
		svc.Logger.Error(err)
		return
	}
	svc.Logger.Debugf("Succesfully published invoice to rabbitmq with RHash %s", invoice.RHash)
}
