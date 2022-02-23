package service

import (
	"context"
	"strconv"
	"strings"

	"github.com/getAlby/lndhub.go/lnd"
	"github.com/lightningnetwork/lnd/lnrpc"
)

func (svc *LndhubService) FetchBolt12Invoice(ctx context.Context, offer, memo string, amt int64) (result *lnd.Bolt12, err error) {
	return svc.LndClient.FetchBolt12Invoice(ctx, offer, memo, amt)
}

func (svc *LndhubService) PayBolt12Invoice(ctx context.Context, invoice *lnd.Bolt12) (result *lnrpc.SendResponse, err error) {
	return nil, err
}
func (svc *LndhubService) TransformBolt12(bolt12 *lnd.Bolt12) (*lnrpc.PayReq, error) {

	//todo see if CLN really can't return an int here
	msatAmt, err := strconv.Atoi(strings.Trim(bolt12.AmountMsat, "msat"))
	if err != nil {
		return nil, err
	}
	return &lnrpc.PayReq{
		Destination:     bolt12.NodeID,
		PaymentHash:     bolt12.PaymentHash,
		NumSatoshis:     int64(msatAmt / lnd.MSAT_PER_SAT),
		Timestamp:       bolt12.Timestamp,
		Expiry:          bolt12.RelativeExpiry,
		Description:     bolt12.Description,
		DescriptionHash: "", // not supported by bolt 12?
		NumMsat:         int64(msatAmt),
	}, nil
}
