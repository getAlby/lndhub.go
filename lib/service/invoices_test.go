package service

import (
	"testing"

	"github.com/getAlby/lndhub.go/db/models"
	"github.com/stretchr/testify/assert"
)

func TestCalcFeeWithInvoiceLessThan1000(t *testing.T) {
	invoice := &models.Invoice{
		Amount: 500,
	}

	feeLimit := calcFeeLimit(invoice)
	expectedFee := int64(10)
	assert.Equal(t, expectedFee, feeLimit.GetFixed())
}

func TestCalcFeeWithInvoiceEqualTo1000(t *testing.T) {
	invoice := &models.Invoice{
		Amount: 500,
	}

	feeLimit := calcFeeLimit(invoice)
	expectedFee := int64(10)
	assert.Equal(t, expectedFee, feeLimit.GetFixed())
}

func TestCalcFeeWithInvoiceMoreThan1000(t *testing.T) {
	invoice := &models.Invoice{
		Amount: 1500,
	}

	feeLimit := calcFeeLimit(invoice)
	// 1500 * 0.01 + 1
	expectedFee := int64(16)
	assert.Equal(t, expectedFee, feeLimit.GetFixed())
}

func TestCalcFeeForMaxFeeLimit(t *testing.T) {
	invoice := &models.Invoice{
		Amount: 1000000,
	}

	// set max fee limit config to 2100
	feeLimit := calcFeeLimit(invoice)
	expectedFee := int64(2100)
	assert.Equal(t, expectedFee, feeLimit.GetFixed())
}
