package service

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"

	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getsentry/sentry-go"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnrpc/routerrpc"
)

func (svc *LndhubService) GetAllPendingPayments(ctx context.Context) ([]models.Invoice, error) {
	payments := []models.Invoice{}
	err := svc.DB.NewSelect().Model(&payments).Where("state = 'initialized'").Where("type = 'outgoing'").Where("r_hash != ''").Where("created_at >= (now() - interval '2 weeks') ").Scan(ctx)
	return payments, err
}
func (svc *LndhubService) CheckPendingOutgoingPayments(ctx context.Context, pendingPayments []models.Invoice) (err error) {
	svc.Logger.Infof("Found %d pending payments", len(pendingPayments))
	//call trackoutgoingpaymentstatus for each one
	var wg sync.WaitGroup
	for _, inv := range pendingPayments {
		wg.Add(1)
		//spawn goroutines
		//https://go.dev/doc/faq#closures_and_goroutines
		inv := inv
		svc.Logger.Infof("Spawning tracker for payment with hash %s", inv.RHash)
		go func() {
			svc.TrackOutgoingPaymentstatus(ctx, &inv)
			wg.Done()
		}()
	}
	wg.Wait()
	return nil
}

func (svc *LndhubService) GetTransactionEntryByInvoiceId(ctx context.Context, id int64) (models.TransactionEntry, error) {
	entry := models.TransactionEntry{}
	feeReserveEntry := models.TransactionEntry{}

	err := svc.DB.NewSelect().Model(&entry).Where("invoice_id = ? and entry_type = ?", id, models.EntryTypeOutgoing).Limit(1).Scan(ctx)
	if err != nil {
		//migration issue: pre-feereserve payment will cause a "no rows in result set" error.
		//in this case, we also look for the entries without the outgoing check, and do not add the fee reserve
		//we can remove this later when all relevant payments will have an entry_type and a fee_reserve tx
		if errors.Is(err, sql.ErrNoRows) {
			//check again with legacy query
			err = svc.DB.NewSelect().Model(&entry).Where("invoice_id = ?", id).Limit(1).Scan(ctx)
			if err == nil {
				return entry, nil
			}
		}
		return entry, err
	}
	err = svc.DB.NewSelect().Model(&feeReserveEntry).Where("invoice_id = ? and entry_type = ?", id, models.EntryTypeFeeReserve).Limit(1).Scan(ctx)
	if err != nil {
		return entry, err
	}
	entry.FeeReserve = &feeReserveEntry
	return entry, err
}

// Should be called in a goroutine as the tracking can potentially take a long time
func (svc *LndhubService) TrackOutgoingPaymentstatus(ctx context.Context, invoice *models.Invoice) {
	//ask lnd using TrackPaymentV2 by hash of payment
	rawHash, err := hex.DecodeString(invoice.RHash)
	if err != nil {
		svc.Logger.Errorf("Error tracking payment %s: %s", invoice.RHash, err.Error())
		return
	}
	paymentTracker, err := svc.LndClient.SubscribePayment(ctx, &routerrpc.TrackPaymentRequest{
		PaymentHash:       rawHash,
		NoInflightUpdates: true,
	})
	if err != nil {
		svc.Logger.Errorf("Error tracking payment %s: %s", invoice.RHash, err.Error())
		return
	}

	//call HandleFailedPayment or HandleSuccesfulPayment
	for {
		payment, err := paymentTracker.Recv()
		if err != nil {
			svc.Logger.Errorf("Error tracking payment with hash %s: %s", invoice.RHash, err.Error())
			return
		}
		entry, err := svc.GetTransactionEntryByInvoiceId(ctx, invoice.ID)
		if err != nil {
			svc.Logger.Errorf("Error tracking payment %s: %s", invoice.RHash, err.Error())
			return

		}
		if entry.UserID != invoice.UserID {
			svc.Logger.Errorf("User ID's don't match : entry %v, invoice %v", entry, invoice)
			return
		}
		if payment.Status == lnrpc.Payment_FAILED {
			svc.Logger.Infof("Failed payment detected: hash %s, reason %s", payment.PaymentHash, payment.FailureReason)
			err = svc.HandleFailedPayment(ctx, invoice, entry, fmt.Errorf(payment.FailureReason.String()))
			if err != nil {
				sentry.CaptureException(err)
				svc.Logger.Errorf("Error handling failed payment %s: %s", invoice.RHash, err.Error())
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
				svc.Logger.Errorf("Error handling successful payment %s: %s", invoice.RHash, err.Error())
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
