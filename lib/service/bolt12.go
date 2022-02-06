package service

import (
	"context"

	"github.com/getAlby/lndhub.go/lnd"
	"github.com/lightningnetwork/lnd/lnrpc"
)

func (svc *LndhubService) FetchBolt12Invoice(ctx context.Context, offer, memo string, amt int64) (result *lnd.Bolt12Invoice, err error) {
	return nil, err
}

func (svc *LndhubService) PayBolt12Invoice(ctx context.Context, invoice string) (result *lnrpc.SendResponse, err error) {
	return nil, err
}
