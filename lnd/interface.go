package lnd

import (
	"context"

	"github.com/lightningnetwork/lnd/lnrpc"
	"google.golang.org/grpc"
)

type LightningClientWrapper interface {
	ListChannels(ctx context.Context, req *lnrpc.ListChannelsRequest, options ...grpc.CallOption) (*lnrpc.ListChannelsResponse, error)
	SendPaymentSync(ctx context.Context, req *lnrpc.SendRequest, options ...grpc.CallOption) (*lnrpc.SendResponse, error)
	AddInvoice(ctx context.Context, req *lnrpc.Invoice, options ...grpc.CallOption) (*lnrpc.AddInvoiceResponse, error)
	SubscribeInvoices(ctx context.Context, req *lnrpc.InvoiceSubscription, options ...grpc.CallOption) (SubscribeInvoicesWrapper, error)
	GetInfo(ctx context.Context, req *lnrpc.GetInfoRequest, options ...grpc.CallOption) (*lnrpc.GetInfoResponse, error)
	DecodeBolt11(ctx context.Context, bolt11 string, options ...grpc.CallOption) (*lnrpc.PayReq, error)
	TrackPayment(ctx context.Context, hash string, options ...grpc.CallOption) (*lnrpc.Payment, error)
}

type SubscribeInvoicesWrapper interface {
	Recv() (*lnrpc.Invoice, error)
}
