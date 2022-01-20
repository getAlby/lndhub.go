package service

import (
	"context"
	"encoding/hex"

	"github.com/lightningnetwork/lnd/lnrpc"
)

func (svc *LndhubService) GetInfo(ctx context.Context) (*lnrpc.GetInfoResponse, error) {
	return svc.LndClient.GetInfo(ctx, &lnrpc.GetInfoRequest{})
}

func (svc *LndhubService) GetIdentPubKeyHex() string {
	return hex.EncodeToString(svc.IdentityPubkey.SerializeCompressed())
}
