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
	return &lnrpc.PayReq{
		Destination: bolt12.NodeID,
		PaymentHash: bolt12.PaymentHash,
		//todo see if CLN really can't return an int here
		NumSatoshis:     0,
		Timestamp:       bolt12.Timestamp,
		Expiry:          bolt12.RelativeExpiry, //todo is this correct?
		Description:     bolt12.Description,
		DescriptionHash: "", // not supported by bolt 12?
		//todo see if CLN really can't return an int here
		NumMsat: 0,
	}, nil
}
