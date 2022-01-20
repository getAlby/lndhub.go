package service

import (
	"context"

	"github.com/lightningnetwork/lnd/lnrpc"
)

func (svc *LndhubService) GetInfo(ctx context.Context) (*lnrpc.GetInfoResponse, error) {
	return svc.LndClient.GetInfo(ctx, &lnrpc.GetInfoRequest{})
}
