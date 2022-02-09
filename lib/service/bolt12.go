package service

import (
	"context"

	"github.com/getAlby/lndhub.go/lnd"
	"github.com/lightningnetwork/lnd/lnrpc"
)

func (svc *LndhubService) FetchBolt12Invoice(ctx context.Context, offer, memo string, amt int64) (result *lnd.Bolt12, err error) {
	return svc.LndClient.FetchBolt12Invoice(ctx, offer, memo, amt)
}

func (svc *LndhubService) PayBolt12Invoice(ctx context.Context, invoice *lnd.Bolt12) (result *lnrpc.SendResponse, err error) {
	return nil, err
}
