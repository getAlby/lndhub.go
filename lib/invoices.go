package lib

import (
	"context"

	"github.com/getAlby/lndhub.go/db/models"
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

func (svc *LndhubService) AddInvoice(userID int64, amount uint, memo, descriptionHash string) (*models.Invoice, error) {
	invoice := &models.Invoice{
		Type:               "",
		UserID:             userID,
		TransactionEntryID: 0,
		Amount:             amount,
		Memo:               memo,
		DescriptionHash:    descriptionHash,
		PaymentRequest:     "",
		RHash:              "",
		State:              "",
	}

	// TODO: move this to a service layer and call a method
	_, err := svc.DB.NewInsert().Model(invoice).Exec(context.TODO())
	if err != nil {
		return nil, err
	}
	return invoice, nil
}
