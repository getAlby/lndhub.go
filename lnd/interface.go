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
	DecodeBolt12(ctx context.Context, bolt12 string) (*Bolt12, error)
	FetchBolt12Invoice(ctx context.Context, offer, memo string, amount int64) (*Bolt12, error)
}

type SubscribeInvoicesWrapper interface {
	Recv() (*lnrpc.Invoice, error)
}

//Bolt12 can be both an offer or an invoice
//depending on Type
type Bolt12 struct {
	Type               string   `json:"type"`
	OfferID            string   `json:"offer_id"`
	Chains             []string `json:"chains"`
	Description        string   `json:"description"`
	NodeID             string   `json:"node_id"`
	Signature          string   `json:"signature"`
	Vendor             string   `json:"vendor"`
	Valid              bool     `json:"valid"`
	AmountMsat         string   `json:"amount_msat"`
	Features           string   `json:"features"`
	PayerKey           string   `json:"payer_key"`
	PayerInfo          string   `json:"payer_info"`
	PayerNote          string   `json:"payer_note"`
	Timestamp          int64    `json:"timestamp"`
	CreatedAt          int64    `json:"created_at"`
	PaymentHash        string   `json:"payment_hash"`
	RelativeExpiry     int64    `json:"relative_expiry"`
	MinFinalCltvExpiry int64    `json:"min_final_cltv_expiry"`
	Encoded            string   `json:"encoded"`
}
