package integration_tests

import (
	"context"

	"github.com/getAlby/lndhub.go/lnd"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnrpc/routerrpc"
	"google.golang.org/grpc"
)

func NewLNDMockHodlWrapper(lnd lnd.LightningClientWrapper) (result *LNDMockWrapper, err error) {
	return &LNDMockWrapper{
		lnd,
	}, nil
}

type LNDMockHodlWrapperAsync struct {
	hps *HodlPaymentSubscriber
	lnd.LightningClientWrapper
}

type HodlPaymentSubscriber struct {
	ch chan (lnrpc.Payment)
}

// wait for channel, then return
func (hps *HodlPaymentSubscriber) Recv() (*lnrpc.Payment, error) {
	result := <-hps.ch
	return &result, nil
}

func NewLNDMockHodlWrapperAsync(lnd lnd.LightningClientWrapper) (result *LNDMockHodlWrapperAsync, err error) {
	return &LNDMockHodlWrapperAsync{
		hps: &HodlPaymentSubscriber{
			ch: make(chan lnrpc.Payment, 5),
		},
		LightningClientWrapper: lnd,
	}, nil
}

func (wrapper *LNDMockHodlWrapperAsync) SubscribePayment(ctx context.Context, req *routerrpc.TrackPaymentRequest, options ...grpc.CallOption) (lnd.SubscribePaymentWrapper, error) {
	return wrapper.hps, nil
}

func (wrapper *LNDMockHodlWrapperAsync) SendPaymentSync(ctx context.Context, req *lnrpc.SendRequest, options ...grpc.CallOption) (*lnrpc.SendResponse, error) {
	//block indefinetely
	//because we don't want this function to ever return something here
	//the payments should be processed asynchronously by the payment tracker
	select {}
}

func (wrapper *LNDMockHodlWrapperAsync) SettlePayment(payment lnrpc.Payment) {
	wrapper.hps.ch <- payment
}

//write test that completes payment
//write test that fails payment
