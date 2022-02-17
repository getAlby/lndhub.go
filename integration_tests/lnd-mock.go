package integration_tests

import (
	"context"
	"errors"

	"github.com/getAlby/lndhub.go/lnd"
	"github.com/lightningnetwork/lnd/lnrpc"
	"google.golang.org/grpc"
)

type LNDMockWrapper struct {
	*lnd.LNDWrapper
}

func NewLNDMockWrapper(lndOptions lnd.LNDoptions) (result *LNDMockWrapper, err error) {
	lnd, err := lnd.NewLNDclient(lndOptions)
	if err != nil {
		return nil, err
	}

	return &LNDMockWrapper{
		LNDWrapper: lnd,
	}, nil
}

func (wrapper *LNDMockWrapper) SendPaymentSync(ctx context.Context, req *lnrpc.SendRequest, options ...grpc.CallOption) (*lnrpc.SendResponse, error) {
	return nil, errors.New("mocked send payment error")
}
