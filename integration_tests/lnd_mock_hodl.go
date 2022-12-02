package integration_tests

import (
	"context"
	"errors"

	"github.com/getAlby/lndhub.go/lnd"
	"github.com/lightningnetwork/lnd/lnrpc"
	"google.golang.org/grpc"
)

type LNDMockHodlWrapper struct {
	lnd.LightningClientWrapper
}

func NewLNDMockHodlWrapper(lnd lnd.LightningClientWrapper) (result *LNDMockWrapper, err error) {
	return &LNDMockWrapper{
		lnd,
	}, nil
}

func (wrapper *LNDMockHodlWrapper) SendPaymentSync(ctx context.Context, req *lnrpc.SendRequest, options ...grpc.CallOption) (*lnrpc.SendResponse, error) {
	return nil, errors.New(SendPaymentMockError)
}

// mock where send payment sync failure is controlled by channel
// even though send payment method is still sync, suffix "Async" here is used to show intention of using this mock
var paymentResultChannel = make(chan bool, 1)

type LNDMockHodlWrapperAsync struct {
	lnd.LightningClientWrapper
}

func NewLNDMockHodlWrapperAsync(lnd lnd.LightningClientWrapper) (result *LNDMockWrapperAsync, err error) {
	return &LNDMockWrapperAsync{
		lnd,
	}, nil
}

func (wrapper *LNDMockHodlWrapperAsync) SendPaymentSync(ctx context.Context, req *lnrpc.SendRequest, options ...grpc.CallOption) (*lnrpc.SendResponse, error) {
	//block indefinetely
	select {}
}

func (wrapper *LNDMockHodlWrapperAsync) SettlePayment(success bool) {
	paymentResultChannel <- success
}

//TODO: payment tracker implemantation: read from channel, return to receive method
//write test that completes payment
//write test that fails payment
