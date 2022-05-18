package service

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/getAlby/lndhub.go/common"
	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/lnd"
	"github.com/getsentry/sentry-go"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/uptrace/bun"
)

func (svc *LndhubService) HandleInternalKeysendPayment(ctx context.Context, invoice *models.Invoice) error {
	//Find the payee user
	user, err := svc.FindUserByLogin(ctx, string(invoice.DestinationCustomRecords[TLV_WALLET_ID]))
	if err != nil {
		return err
	}
	expiry := time.Hour * 24
	incomingInvoice := models.Invoice{
		Type:                     common.InvoiceTypeIncoming,
		UserID:                   user.ID,
		Amount:                   invoice.Amount,
		Internal:                 true,
		Memo:                     "Keysend payment",
		State:                    common.InvoiceStateInitialized,
		ExpiresAt:                bun.NullTime{Time: time.Now().Add(expiry)},
		Keysend:                  true,
		RHash:                    invoice.RHash,
		Preimage:                 invoice.Preimage,
		DestinationCustomRecords: invoice.DestinationCustomRecords,
		DestinationPubkeyHex:     svc.IdentityPubkey,
		AddIndex:                 invoice.AddIndex,
	}
	//persist the incoming invoice
	_, err = svc.DB.NewInsert().Model(&incomingInvoice).Exec(ctx)
	return err
}

func (svc *LndhubService) HandleKeysendPayment(ctx context.Context, rawInvoice *lnrpc.Invoice) error {
	var invoice models.Invoice
	rHashStr := hex.EncodeToString(rawInvoice.RHash)
	//First check if this keysend payment was already processed
	count, err := svc.DB.NewSelect().Model(&invoice).Where("type = ? AND r_hash = ? AND state = ?",
		common.InvoiceTypeIncoming,
		rHashStr,
		common.InvoiceStateSettled).Count(ctx)
	if err != nil {
		return err
	}
	if count != 0 {
		return fmt.Errorf("Already processed keysend payment %s", rHashStr)
	}

	//construct the invoice
	invoice, err = svc.createKeysendInvoice(ctx, rawInvoice)
	if err != nil {
		return err
	}
	//persist the invoice
	_, err = svc.DB.NewInsert().Model(&invoice).Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (svc *LndhubService) ProcessInvoiceUpdate(ctx context.Context, rawInvoice *lnrpc.Invoice) error {
	var invoice models.Invoice
	rHashStr := hex.EncodeToString(rawInvoice.RHash)

	svc.Logger.Infof("Invoice update: r_hash:%s state:%v", rHashStr, rawInvoice.State.String())

	//Check if it's a keysend payment
	//If it is, an invoice will be created on-the-fly
	if rawInvoice.IsKeysend {
		err := svc.HandleKeysendPayment(ctx, rawInvoice)
		if err != nil {
			return err
		}
	}
	// Search for an incoming invoice with the r_hash that is NOT settled in our DB
	err := svc.DB.NewSelect().Model(&invoice).Where("type = ? AND r_hash = ? AND state <> ? AND expires_at > ?",
		common.InvoiceTypeIncoming,
		rHashStr,
		common.InvoiceStateSettled,
		time.Now()).Limit(1).Scan(ctx)
	if err != nil {
		svc.Logger.Infof("Invoice not found. Ignoring. r_hash:%s", rHashStr)
		return nil
	}

	// Update the DB entry of the invoice
	// If the invoice is settled we save the settle date and the status otherwise we just store the lnd status
	// Additionally to the invoice update we create a transaction entry from the user's incoming account to the user's current account
	// This transaction entry makes the balance available for the user
	svc.Logger.Infof("Invoice update: invoice_id:%v settled:%v value:%v state:%v", invoice.ID, rawInvoice.Settled, rawInvoice.AmtPaidSat, rawInvoice.State)

	// Get the user's current account for the transaction entry
	creditAccount, err := svc.AccountFor(ctx, common.AccountTypeCurrent, invoice.UserID)
	if err != nil {
		svc.Logger.Errorf("Could not find current account user_id:%v invoice_id:%v", invoice.UserID, invoice.ID)
		return err
	}
	// Get the user's incoming account for the transaction entry
	debitAccount, err := svc.AccountFor(ctx, common.AccountTypeIncoming, invoice.UserID)
	if err != nil {
		svc.Logger.Errorf("Could not find incoming account user_id:%v invoice_id:%v", invoice.UserID, invoice.ID)
		return err
	}

	// Process any update in a DB transaction
	tx, err := svc.DB.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		svc.Logger.Errorf("Failed to update the invoice invoice_id:%v r_hash:%s %v", invoice.ID, rHashStr, err)
		return err
	}

	// if the invoice is NOT settled we just update the invoice state
	if !rawInvoice.Settled {
		svc.Logger.Infof("Invoice not settled invoice_id:%v state: %s", invoice.ID, rawInvoice.State.String())
		invoice.State = strings.ToLower(rawInvoice.State.String())

	} else {
		// if the invoice is settled we update the state and create an transaction entry to the current account
		invoice.SettledAt = bun.NullTime{Time: time.Unix(rawInvoice.SettleDate, 0)}
		invoice.State = common.InvoiceStateSettled
		_, err = tx.NewUpdate().Model(&invoice).WherePK().Exec(ctx)
		if err != nil {
			tx.Rollback()
			svc.Logger.Errorf("Could not update invoice invoice_id:%v", invoice.ID)
			return err
		}

		// Transfer the amount from the user's incoming account to the user's current account
		entry := models.TransactionEntry{
			UserID:          invoice.UserID,
			InvoiceID:       invoice.ID,
			CreditAccountID: creditAccount.ID,
			DebitAccountID:  debitAccount.ID,
			Amount:          rawInvoice.AmtPaidSat,
		}

		if rawInvoice.AmtPaidSat != invoice.Amount {
			svc.Logger.Infof("Incoming invoice amount mismatch. user_id:%v invoice_id:%v, amt:%d, amt_paid:%d.", invoice.UserID, invoice.ID, invoice.Amount, rawInvoice.AmtPaidSat)
		}

		// Save the transaction entry
		_, err = tx.NewInsert().Model(&entry).Exec(ctx)
		if err != nil {
			tx.Rollback()
			svc.Logger.Errorf("Could not create incoming->current transaction user_id:%v invoice_id:%v  %v", invoice.UserID, invoice.ID, err)
			return err
		}
	}
	// Commit the DB transaction. Done, everything worked
	err = tx.Commit()
	if err != nil {
		svc.Logger.Errorf("Failed to commit DB transaction user_id:%v invoice_id:%v  %v", invoice.UserID, invoice.ID, err)
		return err
	}
	svc.InvoicePubSub.Publish(strconv.FormatInt(invoice.UserID, 10), invoice)
	svc.InvoicePubSub.Publish(common.InvoiceTypeIncoming, invoice)

	return nil
}

func (svc *LndhubService) createKeysendInvoice(ctx context.Context, rawInvoice *lnrpc.Invoice) (result models.Invoice, err error) {
	//Look for the user-identifying TLV record
	//which are located in the HTLC's.
	//TODO: can the records differe from HTLC to HTLC? Probably not
	if len(rawInvoice.Htlcs) == 0 {
		return result, fmt.Errorf("Invoice's HTLC array has length 0")
	}
	userLoginCustomRecord := rawInvoice.Htlcs[0].CustomRecords[TLV_WALLET_ID]
	//Find user. Our convention here is that the TLV
	//record should contain the user's LNDhub login string
	//(LND already returns the decoded string so there is no need to hex-decode it)
	user, err := svc.FindUserByLogin(ctx, string(userLoginCustomRecord))
	if err != nil {
		return result, err
	}

	expiry := time.Hour * 24 // not really relevant here, the invoice will be updated immediately
	result = models.Invoice{
		Type:                     common.InvoiceTypeIncoming,
		UserID:                   user.ID,
		Amount:                   rawInvoice.AmtPaidSat,
		Memo:                     "Keysend payment", //TODO: also extract this from the custom records?
		State:                    common.InvoiceStateInitialized,
		ExpiresAt:                bun.NullTime{Time: time.Now().Add(expiry)},
		Keysend:                  true,
		RHash:                    hex.EncodeToString(rawInvoice.RHash),
		Preimage:                 hex.EncodeToString(rawInvoice.RPreimage),
		DestinationCustomRecords: rawInvoice.Htlcs[0].CustomRecords,
		DestinationPubkeyHex:     svc.IdentityPubkey,
		AddIndex:                 rawInvoice.AddIndex,
	}
	return result, nil
}

func (svc *LndhubService) ConnectInvoiceSubscription(ctx context.Context) (lnd.SubscribeInvoicesWrapper, error) {
	var invoice models.Invoice
	invoiceSubscriptionOptions := lnrpc.InvoiceSubscription{}
	// Find the oldest NOT settled invoice with an add_index
	err := svc.DB.NewSelect().Model(&invoice).Where("invoice.settled_at IS NULL AND invoice.add_index IS NOT NULL").OrderExpr("invoice.id ASC").Limit(1).Scan(ctx)
	// IF we found an invoice we use that index to start the subscription
	if err == nil {
		invoiceSubscriptionOptions = lnrpc.InvoiceSubscription{AddIndex: invoice.AddIndex - 1} // -1 because we want updates for that invoice already
	}
	svc.Logger.Infof("Starting invoice subscription from index: %v", invoiceSubscriptionOptions.AddIndex)
	return svc.LndClient.SubscribeInvoices(ctx, &invoiceSubscriptionOptions)
}

func (svc *LndhubService) InvoiceUpdateSubscription(ctx context.Context) error {
	invoiceSubscriptionStream, err := svc.ConnectInvoiceSubscription(ctx)
	if err != nil {
		sentry.CaptureException(err)
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("Context was canceled")
		default:
			// receive the next invoice update
			rawInvoice, err := invoiceSubscriptionStream.Recv()
			if err != nil {
				svc.Logger.Errorf("Error processing invoice update subscription: %v", err)
				sentry.CaptureException(err)
				// TODO: close the stream somehoe before retrying?
				// Wait 30 seconds and try to reconnect
				// TODO: implement some backoff
				time.Sleep(30 * time.Second)
				invoiceSubscriptionStream, _ = svc.ConnectInvoiceSubscription(ctx)
				continue
			}

			// Ignore updates for open invoices
			// We store the invoice details in the AddInvoice call
			// Processing open invoices here could cause a race condition:
			// We could get this notification faster than we finish the AddInvoice call
			if rawInvoice.State == lnrpc.Invoice_OPEN {
				svc.Logger.Infof("Invoice state is open. Ignoring update. r_hash:%v", hex.EncodeToString(rawInvoice.RHash))
				continue
			}

			processingError := svc.ProcessInvoiceUpdate(ctx, rawInvoice)
			if processingError != nil {
				svc.Logger.Error(fmt.Errorf("Error %s, invoice hash %s", processingError.Error(), hex.EncodeToString(rawInvoice.RHash)))
				sentry.CaptureException(fmt.Errorf("Error %s, invoice hash %s", processingError.Error(), hex.EncodeToString(rawInvoice.RHash)))
			}
		}
	}
}
