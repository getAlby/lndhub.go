package service

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/lnd"
	"github.com/labstack/gommon/random"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnwire"
	"github.com/lightningnetwork/lnd/zpay32"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"
)

type Route struct {
	TotalAmt  int64 `json:"total_amt"`
	TotalFees int64 `json:"total_fees"`
}

type SendPaymentResponse struct {
	PaymentPreimage    []byte `json:"payment_preimage,omitempty"`
	PaymentPreimageStr string
	PaymentError       string `json:"payment_error,omitempty"`
	PaymentHash        []byte `json:"payment_hash,omitempty"`
	PaymentHashStr     string
	PaymentRoute       *Route
	TransactionEntry   *models.TransactionEntry
	Invoice            *models.Invoice
}

func (svc *LndhubService) FindInvoiceByPaymentHash(userId int64, rHash string) (*models.Invoice, error) {
	var invoice models.Invoice

	err := svc.DB.NewSelect().Model(&invoice).Where("invoice.user_id = ? AND invoice.r_hash = ?", userId, rHash).Limit(1).Scan(context.TODO())
	if err != nil {
		return &invoice, err
	}
	return &invoice, nil
}

func (svc *LndhubService) SendInternalPayment(tx *bun.Tx, invoice *models.Invoice) (SendPaymentResponse, error) {
	sendPaymentResponse := SendPaymentResponse{}
	//SendInternalPayment()
	// find invoice
	var incomingInvoice models.Invoice
	err := svc.DB.NewSelect().Model(&incomingInvoice).Where("type = ? AND payment_request = ? AND state = ? ", "incoming", invoice.PaymentRequest, "open").Limit(1).Scan(context.TODO())
	if err != nil {
		// invoice not found or already settled
		// TODO: logging
		return sendPaymentResponse, err
	}
	// Get the user's current and incoming account for the transaction entry
	recipientCreditAccount, err := svc.AccountFor(context.TODO(), "current", incomingInvoice.UserID)
	if err != nil {
		return sendPaymentResponse, err
	}
	recipientDebitAccount, err := svc.AccountFor(context.TODO(), "incoming", incomingInvoice.UserID)
	if err != nil {
		return sendPaymentResponse, err
	}
	// create recipient entry
	recipientEntry := models.TransactionEntry{
		UserID:          incomingInvoice.UserID,
		InvoiceID:       incomingInvoice.ID,
		CreditAccountID: recipientCreditAccount.ID,
		DebitAccountID:  recipientDebitAccount.ID,
		Amount:          invoice.Amount,
	}
	_, err = tx.NewInsert().Model(&recipientEntry).Exec(context.TODO())
	if err != nil {
		return sendPaymentResponse, err
	}

	// For internal invoices we know the preimage and we use that as a response
	// This allows wallets to get the correct preimage for a payment request even though NO lightning transaction was involved
	preimage, _ := hex.DecodeString(incomingInvoice.Preimage)
	sendPaymentResponse.PaymentPreimageStr = incomingInvoice.Preimage
	sendPaymentResponse.PaymentPreimage = preimage
	sendPaymentResponse.Invoice = invoice
	paymentHash, _ := hex.DecodeString(invoice.RHash)
	sendPaymentResponse.PaymentHashStr = invoice.RHash
	sendPaymentResponse.PaymentHash = paymentHash
	sendPaymentResponse.PaymentRoute = &Route{TotalAmt: invoice.Amount, TotalFees: 0}

	incomingInvoice.Internal = true // mark incoming invoice as internal, just for documentation/debugging
	incomingInvoice.State = "settled"
	incomingInvoice.SettledAt = schema.NullTime{Time: time.Now()}
	_, err = tx.NewUpdate().Model(&incomingInvoice).WherePK().Exec(context.TODO())
	if err != nil {
		// could not save the invoice of the recipient
		return sendPaymentResponse, err
	}

	return sendPaymentResponse, nil
}

func (svc *LndhubService) SendPaymentSync(tx *bun.Tx, invoice *models.Invoice) (SendPaymentResponse, error) {
	sendPaymentResponse := SendPaymentResponse{}
	// TODO: set dynamic fee limit
	feeLimit := lnrpc.FeeLimit{
		//Limit: &lnrpc.FeeLimit_Percent{
		//	Percent: 2,
		//},
		Limit: &lnrpc.FeeLimit_Fixed{
			Fixed: 300,
		},
	}

	// Prepare the LNRPC call
	sendPaymentRequest := lnrpc.SendRequest{
		PaymentRequest: invoice.PaymentRequest,
		Amt:            invoice.Amount,
		FeeLimit:       &feeLimit,
	}

	// Execute the payment
	sendPaymentResult, err := svc.LndClient.SendPaymentSync(context.TODO(), &sendPaymentRequest)
	if err != nil {
		return sendPaymentResponse, err
	}

	// If there was a payment error we rollback and return an error
	if sendPaymentResult.GetPaymentError() != "" || sendPaymentResult.GetPaymentPreimage() == nil {
		return sendPaymentResponse, errors.New(sendPaymentResult.GetPaymentError())
	}

	preimage := sendPaymentResult.GetPaymentPreimage()
	sendPaymentResponse.PaymentPreimage = preimage
	sendPaymentResponse.PaymentPreimageStr = hex.EncodeToString(preimage[:])
	paymentHash := sendPaymentResult.GetPaymentHash()
	sendPaymentResponse.PaymentHash = paymentHash
	sendPaymentResponse.PaymentHashStr = hex.EncodeToString(paymentHash[:])
	sendPaymentResponse.PaymentRoute = &Route{TotalAmt: sendPaymentResult.PaymentRoute.TotalAmt, TotalFees: sendPaymentResult.PaymentRoute.TotalFees}
	return sendPaymentResponse, nil
}

func (svc *LndhubService) PayInvoice(invoice *models.Invoice) (*SendPaymentResponse, error) {
	userId := invoice.UserID

	// Get the user's current and outgoing account for the transaction entry
	debitAccount, err := svc.AccountFor(context.TODO(), "current", userId)
	if err != nil {
		return nil, err
	}
	creditAccount, err := svc.AccountFor(context.TODO(), "outgoing", userId)
	if err != nil {
		return nil, err
	}

	entry := models.TransactionEntry{
		UserID:          userId,
		InvoiceID:       invoice.ID,
		CreditAccountID: creditAccount.ID,
		DebitAccountID:  debitAccount.ID,
		Amount:          invoice.Amount,
	}

	// Start a DB transaction
	// We rollback anything on error (only the invoice that was passed in to the PayInvoice calls stays in the DB)
	tx, err := svc.DB.BeginTx(context.TODO(), &sql.TxOptions{})
	if err != nil {
		return nil, err
	}

	// The DB constraints make sure the user actually has enough balance for the transaction
	// If the user does not have enough balance this call fails
	_, err = tx.NewInsert().Model(&entry).Exec(context.TODO())
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// TODO: maybe save errors on the invoice?

	var paymentResponse SendPaymentResponse
	// Check the destination pubkey if it is an internal invoice and going to our node
	destinationPubkey, err := invoice.DestinationPubkey()
	if err != nil {
		// TODO: logging
		return nil, err
	}
	if svc.IdentityPubkey.IsEqual(destinationPubkey) {
		paymentResponse, err = svc.SendInternalPayment(&tx, invoice)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
	} else {
		paymentResponse, err = svc.SendPaymentSync(&tx, invoice)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	paymentResponse.TransactionEntry = &entry

	// The payment was successful.
	invoice.Preimage = paymentResponse.PaymentPreimageStr
	invoice.State = "settled"
	invoice.SettledAt = schema.NullTime{Time: time.Now()}

	_, err = tx.NewUpdate().Model(invoice).WherePK().Exec(context.TODO())
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// Commit the DB transaction. Done, everything worked
	err = tx.Commit()

	if err != nil {
		return nil, err
	}

	return &paymentResponse, err
}

func (svc *LndhubService) AddOutgoingInvoice(userID int64, paymentRequest string, decodedInvoice *zpay32.Invoice) (*models.Invoice, error) {
	// Initialize new DB invoice
	destinationPubkeyHex := hex.EncodeToString(decodedInvoice.Destination.SerializeCompressed())
	expiresAt := decodedInvoice.Timestamp.Add(decodedInvoice.Expiry())
	invoice := models.Invoice{
		Type:                 "outgoing",
		UserID:               userID,
		PaymentRequest:       paymentRequest,
		State:                "initialized",
		DestinationPubkeyHex: destinationPubkeyHex,
		ExpiresAt:            bun.NullTime{Time: expiresAt},
	}
	if decodedInvoice.Description != nil {
		invoice.Memo = *decodedInvoice.Description
	}
	if decodedInvoice.DescriptionHash != nil {
		dh := *decodedInvoice.DescriptionHash
		invoice.DescriptionHash = hex.EncodeToString(dh[:])
	}
	if decodedInvoice.PaymentHash != nil {
		ph := *decodedInvoice.PaymentHash
		invoice.RHash = hex.EncodeToString(ph[:])
	}
	if decodedInvoice.MilliSat != nil {
		msat := decodedInvoice.MilliSat
		invoice.Amount = int64(msat.ToSatoshis())
	}

	// Save invoice
	_, err := svc.DB.NewInsert().Model(&invoice).Exec(context.TODO())
	if err != nil {
		return nil, err
	}
	return &invoice, nil
}

func (svc *LndhubService) AddIncomingInvoice(userID int64, amount int64, memo, descriptionHashStr string) (*models.Invoice, error) {
	preimage := makePreimageHex()
	expiry := time.Hour * 24 // invoice expires in 24h
	// Initialize new DB invoice
	invoice := models.Invoice{
		Type:            "incoming",
		UserID:          userID,
		Amount:          amount,
		Memo:            memo,
		DescriptionHash: descriptionHashStr,
		State:           "initialized",
		ExpiresAt:       bun.NullTime{Time: time.Now().Add(expiry)},
	}

	// Save invoice - we save the invoice early to have a record in case the LN call fails
	_, err := svc.DB.NewInsert().Model(&invoice).Exec(context.TODO())
	if err != nil {
		return nil, err
	}

	descriptionHash, err := hex.DecodeString(descriptionHashStr)
	if err != nil {
		return nil, err
	}
	// Initialize lnrpc invoice
	lnInvoice := lnrpc.Invoice{
		Memo:            memo,
		DescriptionHash: descriptionHash,
		Value:           amount,
		RPreimage:       preimage,
		Expiry:          int64(expiry.Seconds()),
	}
	// Call LND
	lnInvoiceResult, err := svc.LndClient.AddInvoice(context.TODO(), &lnInvoice)
	if err != nil {
		return nil, err
	}

	// Update the DB invoice with the data from the LND gRPC call
	invoice.PaymentRequest = lnInvoiceResult.PaymentRequest
	invoice.RHash = hex.EncodeToString(lnInvoiceResult.RHash)
	invoice.Preimage = hex.EncodeToString(preimage)
	invoice.AddIndex = lnInvoiceResult.AddIndex
	invoice.DestinationPubkeyHex = svc.GetIdentPubKeyHex() // Our node pubkey for incoming invoices
	invoice.State = "open"

	_, err = svc.DB.NewUpdate().Model(&invoice).WherePK().Exec(context.TODO())
	if err != nil {
		return nil, err
	}

	return &invoice, nil
}

func (svc *LndhubService) TransformBolt12(bolt12 *lnd.Bolt12) (result *zpay32.Invoice, err error) {
	//shoehorn msat into lnwire data type
	msatAmt, err := strconv.Atoi(strings.Trim(bolt12.AmountMsat, "msat"))
	if err != nil {
		return nil, err
	}
	msat := lnwire.MilliSatoshi(msatAmt)
	pubkey, err := constructPubkey(bolt12)
	if err != nil {
		return nil, err
	}
	payerNote := bolt12.PayerNote
	result = &zpay32.Invoice{
		MilliSat:    &msat,
		Timestamp:   time.Unix(bolt12.Timestamp, 0),
		PaymentHash: &[32]byte{},
		Destination: pubkey,
		Description: &payerNote,
	}
	paymentHash := [32]byte{}
	copy(paymentHash[:], bolt12.PaymentHash)
	result.PaymentHash = &paymentHash
	return result, nil
}

func (svc *LndhubService) DecodePaymentRequest(bolt11 string) (*zpay32.Invoice, error) {
	return zpay32.Decode(bolt11, ChainFromCurrency(bolt11[2:]))
}

const hexBytes = random.Hex

func constructPubkey(bolt12 *lnd.Bolt12) (*btcec.PublicKey, error) {
	//horrible code that should be yeeted later
	hexPubkey, err := hex.DecodeString("02" + bolt12.NodeID)
	if err != nil {
		return nil, err
	}
	pubkey, err := btcec.ParsePubKey(hexPubkey[:], btcec.S256())
	if err != nil {
		return nil, err
	}

	sig, err := btcec.ParseDERSignature([]byte(bolt12.Signature), btcec.S256())
	if err != nil {
		return nil, err
	}
	if !sig.Verify([]byte(bolt12.NodeID), pubkey) {
		fmt.Println("should not be here")
		//we made the wrong pick
		hexPubkey, err = hex.DecodeString("03" + bolt12.NodeID)
		if err != nil {
			return nil, err
		}
		pubkey, err = btcec.ParsePubKey(hexPubkey[:], btcec.S256())
		if err != nil {
			return nil, err
		}
	}
	return pubkey, nil
}

func makePreimageHex() []byte {
	b := make([]byte, 32)
	for i := range b {
		b[i] = hexBytes[rand.Intn(len(hexBytes))]
	}
	return b
}

func ChainFromCurrency(currency string) *chaincfg.Params {
	if strings.HasPrefix(currency, "bcrt") {
		return &chaincfg.RegressionNetParams
	} else if strings.HasPrefix(currency, "tb") {
		return &chaincfg.TestNet3Params
	} else if strings.HasPrefix(currency, "sb") {
		return &chaincfg.SimNetParams
	} else {
		return &chaincfg.MainNetParams
	}
}
