package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/getAlby/lndhub.go/common"
	"github.com/getAlby/lndhub.go/db/models"
)

func (svc *LndhubService) StartWebhookSubscribtion(ctx context.Context) {

	svc.Logger.Infof("Starting webhook subscription with webhook url %s", svc.Config.WebhookUrl)
	incomingInvoices := make(chan models.Invoice)
	outgoingInvoices := make(chan models.Invoice)
	svc.InvoicePubSub.Subscribe(common.InvoiceTypeIncoming, incomingInvoices)
	svc.InvoicePubSub.Subscribe(common.InvoiceTypeOutgoing, outgoingInvoices)
	for {
		select {
		case <-ctx.Done():
			return
		case incoming := <-incomingInvoices:
			svc.postToWebhook(incoming)
		case outgoing := <-outgoingInvoices:
			svc.postToWebhook(outgoing)
		}
	}
}
func (svc *LndhubService) postToWebhook(invoice models.Invoice) {

	payload := new(bytes.Buffer)
	err := json.NewEncoder(payload).Encode(invoice)
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
