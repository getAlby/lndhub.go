package service

import (
	"context"

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
