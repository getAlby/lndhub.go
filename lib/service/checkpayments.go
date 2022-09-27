package service

import (
	"context"
	"fmt"

	"github.com/getAlby/lndhub.go/db/models"
	"github.com/lightningnetwork/lnd/lnrpc"
)

func (svc *LndhubService) TrackOutgoingPaymentstatus(ctx context.Context, invoice *models.Invoice) error {

	//fetch the tx entry for the invoice
	entry := models.TransactionEntry{}
	err := svc.DB.NewSelect().Model(&entry).Where("invoice_id = ?", invoice.ID).Limit(1).Scan(ctx)
	if err != nil {
		return err
	}
	if entry.UserID != invoice.UserID {
		return fmt.Errorf("User ID's don't match : entry %v, invoice %v", entry, invoice)
	}
	//ask lnd using TrackPaymentV2 by hash of payment
	payment, err := svc.LndClient.TrackPayment(ctx, invoice.RHash)
	if err != nil {
		return err
	}
	//call HandleFailedPayment or HandleSuccesfulPayment
	if payment.Status == lnrpc.Payment_FAILED {
		return svc.HandleFailedPayment(ctx, invoice, entry, fmt.Errorf(payment.FailureReason.String()))
	}
	if payment.Status == lnrpc.Payment_SUCCEEDED {
		invoice.Fee = payment.FeeSat
		invoice.Preimage = payment.PaymentPreimage
		//is it needed to update the hash here?
		return svc.HandleSuccessfulPayment(ctx, invoice, entry)
	}
	if payment.Status == lnrpc.Payment_IN_FLIGHT {
		//todo, we need to keep calling Recv() in this case, in a seperate goroutine maybe?
		return nil
	}
	return nil
}
