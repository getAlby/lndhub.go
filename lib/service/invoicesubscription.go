package service

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
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

var AlreadyProcessedKeysendError = errors.New("already processed keysend payment")

func (svc *LndhubService) HandleInternalKeysendPayment(ctx context.Context, invoice *models.Invoice) (result *models.Invoice, err error) {
	//Find the payee user
	user, err := svc.FindUserByLogin(ctx, string(invoice.DestinationCustomRecords[TLV_WALLET_ID]))
	if err != nil {
		return nil, err
	}
	preImage, err := makePreimageHex()
	if err != nil {
		return nil, err
	}
	pHash := sha256.New()
	pHash.Write(preImage)
	expiry := time.Hour * 24
	incomingInvoice := models.Invoice{
		Type:                     common.InvoiceTypeIncoming,
		UserID:                   user.ID,
		Amount:                   invoice.Amount,
		Internal:                 true,
		Memo:                     "",
		State:                    common.InvoiceStateInitialized,
		ExpiresAt:                bun.NullTime{Time: time.Now().Add(expiry)},
		Keysend:                  true,
		RHash:                    hex.EncodeToString(pHash.Sum(nil)),
		Preimage:                 hex.EncodeToString(preImage),
		DestinationCustomRecords: invoice.DestinationCustomRecords,
		DestinationPubkeyHex:     svc.IdentityPubkey,
		AddIndex:                 invoice.AddIndex,
	}
	//persist the incoming invoice
	_, err = svc.DB.NewInsert().Model(&incomingInvoice).Exec(ctx)
	return &incomingInvoice, err
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
		return AlreadyProcessedKeysendError
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
			if err == AlreadyProcessedKeysendError {
				return nil
			}
			return err
		}
	}
	// Search for an incoming invoice with the r_hash that is NOT settled in our DB

	err := svc.DB.NewSelect().Model(&invoice).Where("(type = ? OR type = ?) AND r_hash = ? AND state <> ? AND expires_at > ?",
		common.InvoiceTypeIncoming,
		common.InvoiceTypeSubinvoice,
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
	svc.Logger.Infof("Invoice update: invoice_id:%v value:%v state:%v", invoice.ID, rawInvoice.AmtPaidSat, rawInvoice.State)

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

	if rawInvoice.AmtPaidSat != invoice.Amount {
		svc.Logger.Infof("Incoming invoice amount mismatch. user_id:%v invoice_id:%v, amt:%d, amt_paid:%d.", invoice.UserID, invoice.ID, invoice.Amount, rawInvoice.AmtPaidSat)
	}

	// if the invoice is NOT settled we just update the invoice state
	if rawInvoice.State != lnrpc.Invoice_SETTLED {
		svc.Logger.Infof("Invoice not settled invoice_id:%v state: %s", invoice.ID, rawInvoice.State.String())
		invoice.State = strings.ToLower(rawInvoice.State.String())

	} else {
		// if the invoice is settled we update the state and create an transaction entry to the current account
		invoice.SettledAt = bun.NullTime{Time: time.Unix(rawInvoice.SettleDate, 0)}
		invoice.State = common.InvoiceStateSettled
		invoice.Amount = rawInvoice.AmtPaidSat
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

	var subInvoice models.Invoice

	err = svc.DB.NewSelect().Model(&subInvoice).Where("type = ? AND preimage = ? AND add_index = ? AND state <> ? AND expires_at > ?",
		common.InvoiceTypeSubinvoice,
		invoice.Preimage,
		invoice.AddIndex,
		common.InvoiceStateSettled,
		time.Now()).Limit(1).Scan(ctx)

	if err == nil && subInvoice.AddIndex == invoice.AddIndex && rawInvoice.State == lnrpc.Invoice_SETTLED {
		svc.Logger.Infof("External Payment subinvoice found, settling it.")
		newRawInvoice := *rawInvoice
		newRawInvoice.Memo = invoice.Memo
		newRawInvoice.Value = subInvoice.Amount
		newRawInvoice.AmtPaidSat = subInvoice.Amount
		dH, _ := hex.DecodeString(invoice.DescriptionHash)
		newRawInvoice.DescriptionHash = dH
		pI, _ := hex.DecodeString(invoice.Preimage)
		newRawInvoice.DescriptionHash = dH
		newRawInvoice.RPreimage = pI
		subInvoice.RHash = invoice.RHash
		_, err = svc.DB.NewUpdate().Model(&subInvoice).WherePK().Exec(ctx)
		if err != nil {
			svc.Logger.Infof("Could not settle sub invoice %s", err.Error())
		}
		userId := subInvoice.OriginUserID

		debitAccount, err := svc.AccountFor(ctx, common.AccountTypeCurrent, userId)
		if err != nil {
			svc.Logger.Errorf("Could not find current account user_id:%v", userId)
			return err
		}
		creditAccount, err := svc.AccountFor(ctx, common.AccountTypeOutgoing, userId)
		if err != nil {
			svc.Logger.Errorf("Could not find outgoing account user_id:%v", userId)
			return err
		}

		entry := models.TransactionEntry{
			UserID:          userId,
			InvoiceID:       subInvoice.ID,
			CreditAccountID: creditAccount.ID,
			DebitAccountID:  debitAccount.ID,
			Amount:          subInvoice.Amount,
		}

		// The DB constraints make sure the user actually has enough balance for the transaction
		// If the user does not have enough balance this call fails
		_, err = svc.DB.NewInsert().Model(&entry).Exec(ctx)
		if err != nil {
			svc.Logger.Errorf("Could not insert transaction entry user_id:%v invoice_id:%v", userId, invoice.ID)
			return err
		}
		return svc.ProcessInvoiceUpdate(ctx, &newRawInvoice)
	}

	return nil
}

func (svc *LndhubService) createKeysendInvoice(ctx context.Context, rawInvoice *lnrpc.Invoice) (result models.Invoice, err error) {
	//Look for the user-identifying TLV record
	//which are located in the HTLC's.
	//TODO: can the records differe from HTLC to HTLC? Probably not
	if len(rawInvoice.Htlcs) == 0 {
		return result, fmt.Errorf("invoice's HTLC array has length 0")
	}
	userLoginCustomRecord := rawInvoice.Htlcs[0].CustomRecords[TLV_WALLET_ID]
	//Find user. Our convention here is that the TLV
	//record should contain the user's login string
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
		Memo:                     "",
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
	// Find the oldest NOT settled AND NOT expired invoice with an add_index
	// Build in a safety buffer of 14h to account for lndhub downtime
	// Note: expired invoices will not be settled anymore, so we don't care about those
	err := svc.DB.NewSelect().Model(&invoice).Where("invoice.settled_at IS NULL AND invoice.add_index IS NOT NULL AND invoice.expires_at >= (now() - interval '14 hours')").OrderExpr("invoice.id ASC").Limit(1).Scan(ctx)
	// IF we found an invoice we use that index to start the subscription
	// if we get an error there might be a serious issue here
	// and we are at risk of missing paid invoices, so we should not continue
	// if we just didn't find any unsettled invoices that's allright though
	if err != nil && err != sql.ErrNoRows {
		sentry.CaptureException(err)
		return nil, err
	}
	// subtract 1 (read invoiceSubscriptionOptions.Addindex docs)
	invoiceSubscriptionOptions.AddIndex = invoice.AddIndex - 1
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
			return context.Canceled
		default:
			// receive the next invoice update
			rawInvoice, err := invoiceSubscriptionStream.Recv()
			// in case of an error, we want to return and restart LNDhub
			// in order to try and reconnect the gRPC subscription
			if err != nil {
				svc.Logger.Errorf("Error processing invoice update subscription: %v", err)
				sentry.CaptureException(err)
				return err
			}

			// Ignore updates for open invoices
			// We store the invoice details in the AddInvoice call
			// Processing open invoices here could cause a race condition:
			// We could get this notification faster than we finish the AddInvoice call
			if rawInvoice.State == lnrpc.Invoice_OPEN {
				svc.Logger.Debugf("Invoice state is open. Ignoring update. r_hash:%v", hex.EncodeToString(rawInvoice.RHash))
				continue
			}

			processingError := svc.ProcessInvoiceUpdate(ctx, rawInvoice)
			if processingError != nil && processingError != AlreadyProcessedKeysendError {
				svc.Logger.Error(fmt.Errorf("Error %s, invoice hash %s", processingError.Error(), hex.EncodeToString(rawInvoice.RHash)))
				sentry.CaptureException(fmt.Errorf("Error %s, invoice hash %s", processingError.Error(), hex.EncodeToString(rawInvoice.RHash)))
			}
		}
	}
}
