package lnd

import (
	"context"

	cln "github.com/fiatjaf/lightningd-gjson-rpc"
	"github.com/lightningnetwork/lnd/lnrpc"
	"google.golang.org/grpc"
)

type CLNClient struct {
	client *cln.Client
}

func (cl *CLNClient) ListChannels(ctx context.Context, req *lnrpc.ListChannelsRequest, options ...grpc.CallOption) (*lnrpc.ListChannelsResponse, error) {
	panic("not implemented") // TODO: Implement
}

func (cl *CLNClient) SendPaymentSync(ctx context.Context, req *lnrpc.SendRequest, options ...grpc.CallOption) (*lnrpc.SendResponse, error) {
	panic("not implemented") // TODO: Implement
}

func (cl *CLNClient) AddInvoice(ctx context.Context, req *lnrpc.Invoice, options ...grpc.CallOption) (*lnrpc.AddInvoiceResponse, error) {
	panic("not implemented") // TODO: Implement
}

// Todo here: make CLNClient implement the interface (Recv())
// This method will read from a channel or block
// The handler function publishes on the channel on a received invoice
// set the client's invoice index to the one from req
func (cl *CLNClient) SubscribeInvoices(ctx context.Context, req *lnrpc.InvoiceSubscription, options ...grpc.CallOption) (lnrpc.Lightning_SubscribeInvoicesClient, error) {
	panic("not implemented") // TODO: Implement
}

func (cl *CLNClient) GetInfo(ctx context.Context, req *lnrpc.GetInfoRequest, options ...grpc.CallOption) (*lnrpc.GetInfoResponse, error) {
	panic("not implemented") // TODO: Implement
}
