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
	lnd.LightningClientWrapper
}

func NewLNDMockWrapper(lnd lnd.LightningClientWrapper) (result *LNDMockWrapper, err error) {
	return &LNDMockWrapper{
		lnd,
	}, nil
}

func (wrapper *LNDMockWrapper) SendPaymentSync(ctx context.Context, req *lnrpc.SendRequest, options ...grpc.CallOption) (*lnrpc.SendResponse, error) {
	return nil, errors.New(SendPaymentMockError)
}

// mock where send payment sync failure is controlled by channel
// even though send payment method is still sync, suffix "Async" here is used to show intention of using this mock
var errorMessageChannel = make(chan string, 1)

type LNDMockWrapperAsync struct {
	lnd.LightningClientWrapper
}

func NewLNDMockWrapperAsync(lnd lnd.LightningClientWrapper) (result *LNDMockWrapperAsync, err error) {
	return &LNDMockWrapperAsync{
		lnd,
	}, nil
}

func (wrapper *LNDMockWrapperAsync) SendPaymentSync(ctx context.Context, req *lnrpc.SendRequest, options ...grpc.CallOption) (*lnrpc.SendResponse, error) {
	errorMessage := <-errorMessageChannel
	return nil, errors.New(errorMessage)
}

func (wrapper *LNDMockWrapperAsync) FailPayment(message string) {
	errorMessageChannel <- message
}
