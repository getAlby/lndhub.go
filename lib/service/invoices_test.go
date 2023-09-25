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
