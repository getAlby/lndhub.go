package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/getAlby/lndhub.go/common"
	"github.com/getAlby/lndhub.go/db/models"
	amqp "github.com/rabbitmq/amqp091-go"
)

func (svc *LndhubService) StartRabbitMqPublisher(ctx context.Context) error {
	conn, err := amqp.Dial(svc.Config.RabbitMQUri)
	if err != nil {
		return err
	}
	svc.RabbitMqConn = conn
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	err = ch.ExchangeDeclare(
		//TODO: review exchange config
		svc.Config.RabbitMQInvoiceExchange,
		"topic", // type
		true,    // durable
		false,   // auto-deleted
		false,   // internal
		false,   // no-wait
		nil,     // arguments
	)
	if err != nil {
		return err
	}

	svc.Logger.Infof("Starting rabbitmq publisher")
	incomingInvoices := make(chan models.Invoice)
	outgoingInvoices := make(chan models.Invoice)
	_, err = svc.InvoicePubSub.Subscribe(common.InvoiceTypeIncoming, incomingInvoices)
	if err != nil {
		svc.Logger.Error(err.Error())
	}
	_, err = svc.InvoicePubSub.Subscribe(common.InvoiceTypeOutgoing, outgoingInvoices)
	if err != nil {
		svc.Logger.Error(err.Error())
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

	//Look up the user's login to add it to the invoice
	user, err := svc.FindUser(context.Background(), invoice.UserID)
	if err != nil {
		svc.Logger.Error(err)
		return
	}

	payload := new(bytes.Buffer)
	err = json.NewEncoder(payload).Encode(WebhookInvoicePayload{
		ID:                       invoice.ID,
		Type:                     invoice.Type,
		UserLogin:                user.Login,
		Amount:                   invoice.Amount,
		Fee:                      invoice.Fee,
		Memo:                     invoice.Memo,
		DescriptionHash:          invoice.DescriptionHash,
		PaymentRequest:           invoice.PaymentRequest,
		DestinationPubkeyHex:     invoice.DestinationPubkeyHex,
		DestinationCustomRecords: invoice.DestinationCustomRecords,
		RHash:                    invoice.RHash,
		Preimage:                 invoice.Preimage,
		Keysend:                  invoice.Keysend,
		State:                    invoice.State,
		ErrorMessage:             invoice.ErrorMessage,
		CreatedAt:                invoice.CreatedAt,
		ExpiresAt:                invoice.ExpiresAt.Time,
		UpdatedAt:                invoice.UpdatedAt.Time,
		SettledAt:                invoice.SettledAt.Time,
	})
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
	svc.Logger.Debugf("Succesfully published %s", payload.String())
}
