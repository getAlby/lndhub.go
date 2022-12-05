package service

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getsentry/sentry-go"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnrpc/routerrpc"
)

func (svc *LndhubService) CheckAllPendingOutgoingPayments(ctx context.Context) (err error) {
	//check database for all pending payments
	pendingPayments := []models.Invoice{}
	//since this part is synchronously executed before the main server starts, we should not get into race conditions
	//only fetch invoices from the last 2 weeks which should be a safe timeframe for hodl invoices to avoid refetching old invoices again and again
	err = svc.DB.NewSelect().Model(&pendingPayments).Where("state = 'initialized'").Where("type = 'outgoing'").Where("created_at >= (now() - interval '2 weeks') ").Scan(ctx)
	if err != nil {
		return err
	}
	svc.Logger.Infof("Found %d pending payments", len(pendingPayments))
	//call trackoutgoingpaymentstatus for each one
	for _, inv := range pendingPayments {
		//spawn goroutines
		//https://go.dev/doc/faq#closures_and_goroutines
		inv := inv
		svc.Logger.Infof("Spawning tracker for payment with hash %s", inv.RHash)
		go svc.TrackOutgoingPaymentstatus(ctx, &inv)
	}
	return nil
}

// Should be called in a goroutine as the tracking can potentially take a long time
func (svc *LndhubService) TrackOutgoingPaymentstatus(ctx context.Context, invoice *models.Invoice) {
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
	//fetch the tx entry for the invoice
	entry := models.TransactionEntry{}
	err = svc.DB.NewSelect().Model(&entry).Where("invoice_id = ?", invoice.ID).Limit(1).Scan(ctx)
	if err != nil {
		svc.Logger.Errorf("Error tracking payment %v: %s", invoice, err.Error())
		return

	}
	if entry.UserID != invoice.UserID {
		svc.Logger.Errorf("User ID's don't match : entry %v, invoice %v", entry, invoice)
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
			err = svc.HandleFailedPayment(ctx, invoice, entry, fmt.Errorf(payment.FailureReason.String()))
			if err != nil {
				sentry.CaptureException(err)
				svc.Logger.Errorf("Error handling failed payment %v: %s", invoice, err.Error())
				return
			}
			svc.Logger.Infof("Updated failed payment: hash %s, reason %s", payment.PaymentHash, payment.FailureReason)
			return
		}
		if payment.Status == lnrpc.Payment_SUCCEEDED {
			invoice.Fee = payment.FeeSat
			invoice.Preimage = payment.PaymentPreimage
			svc.Logger.Infof("Completed payment detected: hash %s", payment.PaymentHash)
			err = svc.HandleSuccessfulPayment(ctx, invoice, entry)
			if err != nil {
				sentry.CaptureException(err)
				svc.Logger.Errorf("Error handling successful payment %v: %s", invoice, err.Error())
				return
			}
			svc.Logger.Infof("Updated completed payment: hash %s", payment.PaymentHash)
			return
		}
		//Since we shouldn't get in-flight updates we shouldn't get here
		sentry.CaptureException(fmt.Errorf("Got an unexpected payment update %v", payment))
		svc.Logger.Warnf("Got an unexpected in-flight update %v", payment)
	}
}
