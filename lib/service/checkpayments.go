package service

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/getAlby/lndhub.go/db/models"
	"github.com/lightningnetwork/lnd/lnrpc"
)

func (svc *LndhubService) CheckAllPendingOutgoingPayments(ctx context.Context) (err error) {
	//check database for all pending payments
	pendingPayments := []models.Invoice{}
	err = svc.DB.NewSelect().Model(&pendingPayments).Where("state = 'initialized'").Where("type = 'outgoing'").Scan(ctx)
	if err != nil {
		return err
	}
	svc.Logger.Infof("Found %d pending payments", len(pendingPayments))
	//call trackoutgoingpaymentstatus for each one
	for _, inv := range pendingPayments {
		err = svc.TrackOutgoingPaymentstatus(ctx, &inv)
		if err != nil {
			svc.Logger.Error(err)
		}
	}
	return nil
}

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
	rawHash, err := hex.DecodeString(invoice.RHash)
	if err != nil {
		return err
	}
	payment, err := svc.LndClient.TrackPayment(ctx, rawHash)
	if err != nil {
		return err
	}
	//call HandleFailedPayment or HandleSuccesfulPayment
	if payment.Status == lnrpc.Payment_FAILED {
		svc.Logger.Infof("Failed payment detected: %v", payment)
		//todo handle failed payment
		//return svc.HandleFailedPayment(ctx, invoice, entry, fmt.Errorf(payment.FailureReason.String()))
		return nil
	}
	if payment.Status == lnrpc.Payment_SUCCEEDED {
		invoice.Fee = payment.FeeSat
		invoice.Preimage = payment.PaymentPreimage
		svc.Logger.Infof("Completed payment detected: %v", payment)
		//todo handle succesful payment
		//return svc.HandleSuccessfulPayment(ctx, invoice, entry)
		return nil
	}
	if payment.Status == lnrpc.Payment_IN_FLIGHT {
		//todo handle inflight payment
		svc.Logger.Infof("In-flight payment detected: %v", payment)
		return nil
	}
	return nil
}
