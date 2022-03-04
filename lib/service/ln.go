package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/getAlby/lndhub.go/db/models"
	"github.com/lightningnetwork/lnd/lnrpc"
)

//https://github.com/hsjoberg/blixt-wallet/blob/9fcc56a7dc25237bc14b85e6490adb9e044c009c/src/utils/constants.ts#L5
const (
	KEYSEND_CUSTOM_RECORD = 5482373484
	TLV_WHATSAT_MESSAGE   = 34349334
	TLV_RECORD_NAME       = 128100
)

func (svc *LndhubService) GetInfo(ctx context.Context) (*lnrpc.GetInfoResponse, error) {
	return svc.LndClient.GetInfo(ctx, &lnrpc.GetInfoRequest{})
}

func (svc *LndhubService) KeySendPaymentSync(ctx context.Context, invoice *models.Invoice) (result SendPaymentResponse, err error) {
	sendPaymentResponse := SendPaymentResponse{}
	// TODO: set dynamic fee limit
	feeLimit := lnrpc.FeeLimit{
		Limit: &lnrpc.FeeLimit_Fixed{
			Fixed: 300,
		},
	}
	preImage := makePreimageHex()
	pHash := sha256.New()
	pHash.Write(preImage)
	// Prepare the LNRPC call
	//See: https://github.com/hsjoberg/blixt-wallet/blob/9fcc56a7dc25237bc14b85e6490adb9e044c009c/src/lndmobile/index.ts#L251-L270
	destBytes, err := hex.DecodeString(invoice.DestinationPubkeyHex)
	if err != nil {
		return sendPaymentResponse, err
	}
	sendPaymentRequest := lnrpc.SendRequest{
		Dest:              destBytes,
		Amt:               invoice.Amount,
		PaymentHash:       pHash.Sum(nil),
		FeeLimit:          &feeLimit,
		DestFeatures:      []lnrpc.FeatureBit{lnrpc.FeatureBit_TLV_ONION_REQ},
		DestCustomRecords: map[uint64][]byte{KEYSEND_CUSTOM_RECORD: preImage, TLV_WHATSAT_MESSAGE: []byte(invoice.Memo)},
	}

	// Execute the payment
	sendPaymentResult, err := svc.LndClient.SendPaymentSync(ctx, &sendPaymentRequest)
	if err != nil {
		return sendPaymentResponse, err
	}

	// If there was a payment error we return an error
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
