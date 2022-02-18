package integration_tests

import (
	"context"
	"errors"

	"github.com/getAlby/lndhub.go/lnd"
	"github.com/lightningnetwork/lnd/lnrpc"
	"google.golang.org/grpc"
)

const SendPaymentMockError = "mocked send payment error"

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
	return nil, errors.New(SendPaymentMockError)
}

// mock where send payment sync failure is controlled by channel
var errorMessageChannel = make(chan string, 1)

type LNDMockWrapperAsync struct {
	*lnd.LNDWrapper
}

func NewLNDMockWrapperAsync(lndOptions lnd.LNDoptions) (result *LNDMockWrapperAsync, err error) {
	lnd, err := lnd.NewLNDclient(lndOptions)
	if err != nil {
		return nil, err
	}

	return &LNDMockWrapperAsync{
		LNDWrapper: lnd,
	}, nil
}

func (wrapper *LNDMockWrapperAsync) SendPaymentSync(ctx context.Context, req *lnrpc.SendRequest, options ...grpc.CallOption) (*lnrpc.SendResponse, error) {
	errorMessage := <-errorMessageChannel
	return nil, errors.New(errorMessage)
}

func (wrapper *LNDMockWrapperAsync) FailPayment(message string) {
	errorMessageChannel <- message
}
