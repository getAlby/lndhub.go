package service

import (
	"context"
	"encoding/hex"
	"math/rand"

	"github.com/getAlby/lndhub.go/db/models"
	"github.com/labstack/gommon/random"
	"github.com/lightningnetwork/lnd/lnrpc"
)

func (svc *LndhubService) Payinvoice(userId int64, invoice string) error {
	debitAccount, err := svc.AccountFor(context.TODO(), "current", userId)
	if err != nil {
		return err
	}
	creditAccount, err := svc.AccountFor(context.TODO(), "outgoing", userId)
	if err != nil {
		return err
	}

	entry := models.TransactionEntry{
		UserID:          userId,
		CreditAccountID: creditAccount.ID,
		DebitAccountID:  debitAccount.ID,
		Amount:          1000,
	}
	_, err = svc.DB.NewInsert().Model(&entry).Exec(context.TODO())
	return err

}

func (svc *LndhubService) AddInvoice(userID int64, amount int64, memo, descriptionHash string) (*models.Invoice, error) {
	// Initialize new DB invoice
	invoice := models.Invoice{
		Type:            "incoming",
		UserID:          userID,
		Amount:          amount,
		Memo:            memo,
		DescriptionHash: descriptionHash,
		State:           "initialized",
	}

	// Save invoice - we save the invoice early to have a record in case the LN call fails
	_, err := svc.DB.NewInsert().Model(&invoice).Exec(context.TODO())
	if err != nil {
		return nil, err
	}

	// Initialize lnrpc invoice
	lnInvoice := lnrpc.Invoice{
		Memo:      memo,
		Value:     amount,
		RPreimage: makePreimageHex(),
		Expiry:    3600 * 24, // 24h
	}
	lndClient := *svc.LndClient
	// Call LND
	lnInvoiceResult, err := lndClient.AddInvoice(context.TODO(), &lnInvoice)
	if err != nil {
		return nil, err
	}

	// Update the DB invoice with the data from the LND gRPC call
	invoice.PaymentRequest = lnInvoiceResult.PaymentRequest
	invoice.RHash = hex.EncodeToString(lnInvoiceResult.RHash)
	invoice.AddIndex = lnInvoiceResult.AddIndex
	invoice.State = "created"

	_, err = svc.DB.NewUpdate().Model(&invoice).WherePK().Exec(context.TODO())
	if err != nil {
		return nil, err
	}

	return &invoice, nil
}

const hexBytes = random.Hex

func makePreimageHex() []byte {
	b := make([]byte, 32)
	for i := range b {
		b[i] = hexBytes[rand.Intn(len(hexBytes))]
	}
	return b
}
