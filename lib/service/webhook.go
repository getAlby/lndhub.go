package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/getAlby/lndhub.go/common"
	"github.com/getAlby/lndhub.go/db/models"
)

func (svc *LndhubService) StartWebhookSubscription(ctx context.Context, url string) {
	svc.Logger.Infof("Starting webhook subscription with webhook url %s", svc.Config.WebhookUrl)
	incomingInvoices, outgoingInvoices, err := svc.SubscribeIncomingOutgoingInvoices()
	if err != nil {
		svc.Logger.Error(err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case incoming := <-incomingInvoices:
			svc.postToWebhook(incoming, url)
		case outgoing := <-outgoingInvoices:
			svc.postToWebhook(outgoing, url)
		}
	}
}
func (svc *LndhubService) postToWebhook(invoice models.Invoice, url string) {
	payload := new(bytes.Buffer)
	err := svc.AddInvoiceMetadata(context.Background(), payload, invoice)
	if err != nil {
		svc.Logger.Error(err)
		return
	}

	resp, err := http.Post(svc.Config.WebhookUrl, "application/json", payload)
	if err != nil {
		svc.Logger.Error(err)
		return
	}
	if resp.StatusCode != http.StatusOK {
		msg, err := io.ReadAll(resp.Body)
		if err != nil {
			svc.Logger.Error(err)
		}
		svc.Logger.Errorf("Webhook status code was %d, body: %s", resp.StatusCode, msg)
	}
}

func (svc *LndhubService) SubscribeIncomingOutgoingInvoices() (incoming, outgoing chan models.Invoice, err error) {
	incomingInvoices, _, err := svc.InvoicePubSub.Subscribe(common.InvoiceTypeIncoming)
	if err != nil {
		return nil, nil, err
	}
	outgoingInvoices, _, err := svc.InvoicePubSub.Subscribe(common.InvoiceTypeOutgoing)
	if err != nil {
		return nil, nil, err
	}
	return incomingInvoices, outgoingInvoices, nil
}

func (svc *LndhubService) AddInvoiceMetadata(ctx context.Context, w io.Writer, invoice models.Invoice) error {
	user, err := svc.FindUser(ctx, invoice.UserID)
	if err != nil {
		return err
	}

	balance, err := svc.CurrentUserBalance(ctx, invoice.UserID)
	if err != nil {
		return err
	}
	err = json.NewEncoder(w).Encode(ConvertPayload(invoice, user, balance))
	if err != nil {
		return err
	}

	return nil
}

func ConvertPayload(invoice models.Invoice, user *models.User, balance int64) (result models.WebhookInvoicePayload) {
	return models.WebhookInvoicePayload{
		ID:                       invoice.ID,
		Type:                     invoice.Type,
		UserLogin:                user.Login,
		Amount:                   invoice.Amount,
		Fee:                      invoice.Fee,
		Balance:                  balance,
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
	}
}
