package service

import (
	"context"
	"database/sql"
	"encoding/hex"
	"strings"
	"time"

	"github.com/getAlby/lndhub.go/db/models"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/uptrace/bun"
)

func (svc *LndhubService) InvoiceUpdateSubscription(ctx context.Context) error {
	invoiceSubscriptionOptions := lnrpc.InvoiceSubscription{}
	invoiceSubscriptionStream, err := svc.LndClient.SubscribeInvoices(context.Background(), &invoiceSubscriptionOptions)
	if err != nil {
		return err
	}
	svc.Logger.Info("Subscribed to invoice updates starting from index: ")
	for {
		// receive the next invoice update
		rawInvoice, err := invoiceSubscriptionStream.Recv()
		if err != nil {
			// TODO: sentry notification
			svc.Logger.Errorf("Error processing invoice update subscription: %v", err)
			continue
		}
		var invoice models.Invoice
		rHashStr := hex.EncodeToString(rawInvoice.RHash)

		svc.Logger.Infof("Invoice update: r_hash:%s state:%v", rHashStr, rawInvoice.State)

		// Ignore updates for open invoices
		// We store the invoice details in the AddInvoice call
		// This could cause a race condition here where we get this notification faster than we finish the AddInvoice call
		if rawInvoice.State == lnrpc.Invoice_OPEN {
			svc.Logger.Infof("Invoice state is open. Ignoring update. r_hash:%v", rHashStr)
			continue
		}

		// Search for the invoice in our DB
		err = svc.DB.NewSelect().Model(&invoice).Where("type = ? AND r_hash = ? AND state <> ? ", "incoming", rHashStr, "settled").Limit(1).Scan(context.TODO())
		if err != nil {
			// TODO: sentry notification
			svc.Logger.Errorf("Could not find invoice: r_hash:%s payment_request:%s", rHashStr, rawInvoice.PaymentRequest)
			continue
		}

		// Update the DB entry of the invoice
		// If the invoice is settled we save the settle date and the status otherwise we just store the lnd status
		//
		// Additionally to the invoice update we create a transaction entry from the incoming account to the user's current account
		svc.Logger.Infof("Invoice update: invoice_id:%v settled:%v value:%v state:%v", invoice.ID, rawInvoice.Settled, rawInvoice.AmtPaidSat, rawInvoice.State)

		// Get the user's current and incoming account for the transaction entry
		creditAccount, err := svc.AccountFor(ctx, "current", invoice.UserID)
		if err != nil {
			svc.Logger.Errorf("Could not find current account user_id:%v invoice_id:%v", invoice.UserID, invoice.ID)
			// TODO: sentry notification
			continue
		}
		debitAccount, err := svc.AccountFor(ctx, "incoming", invoice.UserID)
		if err != nil {
			svc.Logger.Errorf("Could not find incoming account user_id:%v invoice_id:%v", invoice.UserID, invoice.ID)
			// TODO: sentry notification
			continue
		}

		tx, err := svc.DB.BeginTx(ctx, &sql.TxOptions{})
		if err != nil {
			svc.Logger.Errorf("Failed to update the invoice invoice_id:%v r_hash:%s %v", invoice.ID, rHashStr, err)
			// TODO: notify sentry
			continue
		}

		// if the invoice is NOT settled we just update the invoice state
		if !rawInvoice.Settled {
			svc.Logger.Infof("Invoice not settled invoice_id:%v", invoice.ID)
			invoice.State = strings.ToLower(rawInvoice.State.String())

			// if the invoice is settled we update the state and create an transaction entry to the current account
		} else {
			invoice.SettledAt = bun.NullTime{Time: time.Unix(rawInvoice.SettleDate, 0)}
			invoice.State = "settled"
			_, err = tx.NewUpdate().Model(&invoice).WherePK().Exec(context.TODO())
			if err != nil {
				tx.Rollback()
				svc.Logger.Errorf("Could not update invoice invoice_id:%v", invoice.ID)
				// TODO: sentry notification
				continue
			}

			// Transfer the amount from the incoming account to the current account
			entry := models.TransactionEntry{
				UserID:          invoice.UserID,
				InvoiceID:       invoice.ID,
				CreditAccountID: creditAccount.ID,
				DebitAccountID:  debitAccount.ID,
				Amount:          invoice.Amount,
			}
			// The DB constraints make sure the user actually has enough balance for the transaction
			// If the user does not have enough balance this call fails
			_, err = tx.NewInsert().Model(&entry).Exec(context.TODO())
			if err != nil {
				tx.Rollback()
				svc.Logger.Errorf("Could not create incoming->current transaction user_id:%v invoice_id:%v  %v", invoice.UserID, invoice.ID, err)
				// TODO: sentry notification
				tx.Rollback()
				continue
			}
		}
		// Commit the DB transaction. Done, everything worked
		err = tx.Commit()
		if err != nil {
			svc.Logger.Errorf("Failed to commit DB transaction user_id:%v invoice_id:%v  %v", invoice.UserID, invoice.ID, err)
			// TODO: sentry notification
			continue
		}

	}
	return nil
}
