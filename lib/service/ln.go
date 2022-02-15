package service

import (
	"context"

	"github.com/getAlby/lndhub.go/lnd"
	"github.com/lightningnetwork/lnd/lnrpc"
)

func (svc *LndhubService) GetInfo(ctx context.Context) (*lnrpc.GetInfoResponse, error) {
	return svc.LndClient.GetInfo(ctx, &lnrpc.GetInfoRequest{})
}

func (svc *LndhubService) TransformBolt12(bolt12 *lnd.Bolt12) (*lnrpc.PayReq, error) {
	return nil, nil
}
