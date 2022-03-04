package integration_tests

import (
	"context"
	"fmt"
	"time"

	"github.com/getAlby/lndhub.go/common"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/stretchr/testify/assert"
)

func (suite *PaymentTestSuite) TestOutGoingPayment() {
	aliceFundingSats := 1000
	externalSatRequested := 500
	// 1 sat + 1 ppm
	fee := 1
	//fund alice account
	invoiceResponse := suite.createAddInvoiceReq(aliceFundingSats, "integration test external payment alice", suite.aliceToken)
	sendPaymentRequest := lnrpc.SendRequest{
		PaymentRequest: invoiceResponse.PayReq,
		FeeLimit:       nil,
	}
	_, err := suite.fundingClient.SendPaymentSync(context.Background(), &sendPaymentRequest)
	assert.NoError(suite.T(), err)

	//wait a bit for the callback event to hit
	time.Sleep(100 * time.Millisecond)

	//create external invoice
	externalInvoice := lnrpc.Invoice{
		Memo:  "integration tests: external pay from alice",
		Value: int64(externalSatRequested),
	}
	invoice, err := suite.fundingClient.AddInvoice(context.Background(), &externalInvoice)
	assert.NoError(suite.T(), err)
	//pay external from alice
	payResponse := suite.createPayInvoiceReq(invoice.PaymentRequest, suite.aliceToken)
	assert.NotEmpty(suite.T(), payResponse.PaymentPreimage)

	// check that balance was reduced
	userId := getUserIdFromToken(suite.aliceToken)
	aliceBalance, err := suite.service.CurrentUserBalance(context.Background(), userId)
	if err != nil {
		fmt.Printf("Error when getting balance %v\n", err.Error())
	}
	assert.Equal(suite.T(), int64(aliceFundingSats)-int64(externalSatRequested+fee), aliceBalance)

	// check that no additional transaction entry was created
	transactonEntries, err := suite.service.TransactionEntriesFor(context.Background(), userId)
	if err != nil {
		fmt.Printf("Error when getting transaction entries %v\n", err.Error())
	}
	// verify transaction entries data
	feeAccount, _ := suite.service.AccountFor(context.Background(), common.AccountTypeFees, userId)
	incomingAccount, _ := suite.service.AccountFor(context.Background(), common.AccountTypeIncoming, userId)
	outgoingAccount, _ := suite.service.AccountFor(context.Background(), common.AccountTypeOutgoing, userId)
	currentAccount, _ := suite.service.AccountFor(context.Background(), common.AccountTypeCurrent, userId)

	outgoingInvoices, _ := suite.service.InvoicesFor(context.Background(), userId, common.InvoiceTypeOutgoing)
	incomingInvoices, _ := suite.service.InvoicesFor(context.Background(), userId, common.InvoiceTypeIncoming)
	assert.Equal(suite.T(), 1, len(outgoingInvoices))
	assert.Equal(suite.T(), 1, len(incomingInvoices))

	assert.Equal(suite.T(), 3, len(transactonEntries))

	assert.Equal(suite.T(), int64(aliceFundingSats), transactonEntries[0].Amount)
	assert.Equal(suite.T(), currentAccount.ID, transactonEntries[0].CreditAccountID)
	assert.Equal(suite.T(), incomingAccount.ID, transactonEntries[0].DebitAccountID)
	assert.Equal(suite.T(), int64(0), transactonEntries[0].ParentID)
	assert.Equal(suite.T(), incomingInvoices[0].ID, transactonEntries[0].InvoiceID)

	assert.Equal(suite.T(), int64(externalSatRequested), transactonEntries[1].Amount)
	assert.Equal(suite.T(), outgoingAccount.ID, transactonEntries[1].CreditAccountID)
	assert.Equal(suite.T(), currentAccount.ID, transactonEntries[1].DebitAccountID)
	assert.Equal(suite.T(), int64(0), transactonEntries[1].ParentID)
	assert.Equal(suite.T(), outgoingInvoices[0].ID, transactonEntries[1].InvoiceID)

	assert.Equal(suite.T(), int64(fee), transactonEntries[2].Amount)
	assert.Equal(suite.T(), feeAccount.ID, transactonEntries[2].CreditAccountID)
	assert.Equal(suite.T(), currentAccount.ID, transactonEntries[2].DebitAccountID)
	assert.Equal(suite.T(), outgoingInvoices[0].ID, transactonEntries[2].InvoiceID)

	// make sure fee entry parent id is previous entry
	assert.Equal(suite.T(), transactonEntries[1].ID, transactonEntries[2].ParentID)
}
