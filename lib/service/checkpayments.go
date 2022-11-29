package service

import (
	"context"
	"encoding/hex"

	"github.com/getAlby/lndhub.go/db/models"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnrpc/routerrpc"
)

func (svc *LndhubService) CheckAllPendingOutgoingPayments(ctx context.Context) (err error) {
	//check database for all pending payments
	pendingPayments := []models.Invoice{}
	err = svc.DB.NewSelect().Model(&pendingPayments).Where("state = 'initialized'").Where("type = 'outgoing'").Scan(ctx)
	if err != nil {
		return err
	}
	svc.Logger.Infof("Found %d pending payments, spawning trackers", len(pendingPayments))
	//call trackoutgoingpaymentstatus for each one
	for _, inv := range pendingPayments {
		//spawn goroutines
		go svc.TrackOutgoingPaymentstatus(ctx, &inv)
	}
	return nil
}

// Should be called in a goroutine as the tracking can potentially take a long time
func (svc *LndhubService) TrackOutgoingPaymentstatus(ctx context.Context, invoice *models.Invoice) {

	//fetch the tx entry for the invoice
	entry := models.TransactionEntry{}
	err := svc.DB.NewSelect().Model(&entry).Where("invoice_id = ?", invoice.ID).Limit(1).Scan(ctx)
	if err != nil {
		svc.Logger.Errorf("Error tracking payment %v: %s", invoice, err.Error())
		return

	}
	if entry.UserID != invoice.UserID {
		svc.Logger.Errorf("User ID's don't match : entry %v, invoice %v", entry, invoice)
		return
	}
	//ask lnd using TrackPaymentV2 by hash of payment
	rawHash, err := hex.DecodeString(invoice.RHash)
	if err != nil {
		svc.Logger.Errorf("Error tracking payment %v: %s", invoice, err.Error())
		return
	}
	paymentTracker, err := svc.LndClient.SubscribePayment(ctx, &routerrpc.TrackPaymentRequest{
		PaymentHash:       rawHash,
		NoInflightUpdates: true,
	})
	if err != nil {
		svc.Logger.Errorf("Error tracking payment %v: %s", invoice, err.Error())
		return
	}
	//call HandleFailedPayment or HandleSuccesfulPayment
	for {
		payment, err := paymentTracker.Recv()
		if err != nil {
			svc.Logger.Errorf("Error tracking payment %v: %s", invoice, err.Error())
			return
		}
		if payment.Status == lnrpc.Payment_FAILED {
			svc.Logger.Infof("Failed payment detected: hash %s, reason %s", payment.PaymentHash, payment.FailureReason)
			//todo handle failed payment
			//return svc.HandleFailedPayment(ctx, invoice, entry, fmt.Errorf(payment.FailureReason.String()))
			return
		}
		if payment.Status == lnrpc.Payment_SUCCEEDED {
			invoice.Fee = payment.FeeSat
			invoice.Preimage = payment.PaymentPreimage
			svc.Logger.Infof("Completed payment detected: hash %s", payment.PaymentHash)
			//todo handle succesful payment
			//return svc.HandleSuccessfulPayment(ctx, invoice, entry)
			return
		}
		//Since we shouldn't get in-flight updates we shouldn't get here
		svc.Logger.Warnf("Got an unexpected in-flight update %v", payment)
	}
}
