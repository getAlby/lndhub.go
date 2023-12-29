package service

import (
	"testing"

	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/lnd"
	"github.com/stretchr/testify/assert"
)

var svc = &LndhubService{
	LndClient: &lnd.LNDWrapper{IdentityPubkey: "123pubkey"},
	Config: &Config{
		MaxFeeAmount: 1e6,
	},
}

func TestCalcFeeWithInvoiceLessThan1000(t *testing.T) {
	invoice := &models.Invoice{
		Amount: 500,
	}

	feeLimit := svc.CalcFeeLimit("dummy", invoice.Amount)
	expectedFee := int64(10)
	assert.Equal(t, expectedFee, feeLimit)
}

func TestCalcFeeWithInvoiceEqualTo1000(t *testing.T) {
	invoice := &models.Invoice{
		Amount: 500,
	}

	feeLimit := svc.CalcFeeLimit("dummy", invoice.Amount)
	expectedFee := int64(10)
	assert.Equal(t, expectedFee, feeLimit)
}

func TestCalcFeeWithInvoiceMoreThan1000(t *testing.T) {
	invoice := &models.Invoice{
		Amount: 1500,
	}

	feeLimit := svc.CalcFeeLimit("dummy", invoice.Amount)
	// 1500 * 0.01 + 1
	expectedFee := int64(16)
	assert.Equal(t, expectedFee, feeLimit)
}

func TestCalcFeeWithMaxGlobalFee(t *testing.T) {
	invoice := &models.Invoice{
		Amount: 1500,
	}
	svc.Config.MaxFeeAmount = 1

	feeLimit := svc.CalcFeeLimit("dummy", invoice.Amount)
	expectedFee := svc.Config.MaxFeeAmount
	assert.Equal(t, expectedFee, feeLimit)
}

func TestCalcServiceFee(t *testing.T) {
	var serviceFee int64

	svc.Config.ServiceFee = 0
	serviceFee = svc.CalcServiceFee(10000)
	assert.Equal(t, int64(0), serviceFee)

	svc.Config.ServiceFee = 5
	serviceFee = svc.CalcServiceFee(1000)
	assert.Equal(t, int64(5), serviceFee)

	serviceFee = svc.CalcServiceFee(100)
	assert.Equal(t, int64(1), serviceFee)

	serviceFee = svc.CalcServiceFee(212121)
	assert.Equal(t, int64(1061), serviceFee)

	svc.Config.ServiceFee = 1
	serviceFee = svc.CalcServiceFee(1000)
	assert.Equal(t, int64(1), serviceFee)

	serviceFee = svc.CalcServiceFee(100)
	assert.Equal(t, int64(1), serviceFee)

	serviceFee = svc.CalcServiceFee(212121)
	assert.Equal(t, int64(213), serviceFee)
}

func TestCalcServiceFeeWithFreeAmounts(t *testing.T) {
	var serviceFee int64
	svc.Config.ServiceFee = 5
	svc.Config.NoServiceFeeUpToAmount = 2121

	serviceFee = svc.CalcServiceFee(2100)
	assert.Equal(t, int64(0), serviceFee)

	serviceFee = svc.CalcServiceFee(2121)
	assert.Equal(t, int64(0), serviceFee)

	serviceFee = svc.CalcServiceFee(2122)
	assert.Equal(t, int64(11), serviceFee)
}

func TestSetFeeOnInvoice(t *testing.T) {
	invoice := &models.Invoice{
		Amount: 500,
	}
	entry := &models.TransactionEntry{}
	entry.ServiceFee = &models.TransactionEntry{
		Amount: 42,
	}
	invoice.SetFee(*entry, 21)
	assert.Equal(t, int64(21), invoice.RoutingFee)
	assert.Equal(t, int64(42), invoice.ServiceFee)
	assert.Equal(t, int64(63), invoice.Fee)
}
