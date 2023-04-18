package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

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

	//Look up the user's login to add it to the invoice
	user, err := svc.FindUser(context.Background(), invoice.UserID)
	if err != nil {
		svc.Logger.Error(err)
		return
	}

	payload := new(bytes.Buffer)
	err = json.NewEncoder(payload).Encode(ConvertPayload(invoice, user))
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

type WebhookInvoicePayload struct {
	ID                       int64             `json:"id"`
	Type                     string            `json:"type"`
	UserLogin                string            `json:"user_login"`
	Amount                   int64             `json:"amount"`
	Fee                      int64             `json:"fee"`
	Memo                     string            `json:"memo"`
	DescriptionHash          string            `json:"description_hash,omitempty"`
	PaymentRequest           string            `json:"payment_request"`
	DestinationPubkeyHex     string            `json:"destination_pubkey_hex"`
	DestinationCustomRecords map[uint64][]byte `json:"custom_records,omitempty"`
	RHash                    string            `json:"r_hash"`
	Preimage                 string            `json:"preimage"`
	Keysend                  bool              `json:"keysend"`
	State                    string            `json:"state"`
	ErrorMessage             string            `json:"error_message,omitempty"`
	CreatedAt                time.Time         `json:"created_at"`
	ExpiresAt                time.Time         `json:"expires_at"`
	UpdatedAt                time.Time         `json:"updated_at"`
	SettledAt                time.Time         `json:"settled_at"`
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

func (svc *LndhubService) EncodeInvoiceWithUserLogin(ctx context.Context, w io.Writer, invoice models.Invoice) error {
	user, err := svc.FindUser(ctx, invoice.UserID)
	if err != nil {
		return err
	}

	err = json.NewEncoder(w).Encode(ConvertPayload(invoice, user))
	if err != nil {
		return err
	}

	return nil
}

func ConvertPayload(invoice models.Invoice, user *models.User) (result WebhookInvoicePayload) {
	return WebhookInvoicePayload{
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
	}
}
