package integration_tests

import (
	"context"
	"time"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/stretchr/testify/assert"
)

func (suite *PaymentTestSuite) TestOutGoingPayment() {
	aliceFundingSats := 1000
	externalSatRequested := 500
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
}

func (suite *PaymentTestSuite) TestOutGoingPaymentFailure() {
	//TODO: use a new implementation of LNDClientWrapper interface to test different scenarios
	//might be better if this has it's own suite
	//because we need a different LND client
	// - payment fails directly
	// - payment fails after some time, check that balance is locked in the meantime and is restored afterwards
	// - payment call succeeds after some time, check that balance is locked in the meantime and is _not_ restored afterwards
}
