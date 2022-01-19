package service

import (
	"context"

	"github.com/lightningnetwork/lnd/lnrpc"
)

func (svc *LndhubService) GetInfo(ctx context.Context) (*lnrpc.GetInfoResponse, error) {
	lndClient := *svc.LndClient
	return lndClient.GetInfo(ctx, &lnrpc.GetInfoRequest{})
}
