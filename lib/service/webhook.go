package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/getAlby/lndhub.go/common"
	"github.com/getAlby/lndhub.go/db/models"
)

func (svc *LndhubService) StartWebhookSubscribtion(ctx context.Context, url string) {

	svc.Logger.Infof("Starting webhook subscription with webhook url %s", svc.Config.WebhookUrl)
	incomingInvoices := make(chan models.Invoice)
	outgoingInvoices := make(chan models.Invoice)
	_, err := svc.InvoicePubSub.Subscribe(common.InvoiceTypeIncoming, incomingInvoices)
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

	resp, err := http.Post(svc.Config.WebhookUrl, "application/json", payload)
	if err != nil {
		svc.Logger.Error(err)
		return
	}
	if resp.StatusCode != http.StatusOK {
		msg, err := ioutil.ReadAll(resp.Body)
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
