package integration_tests

import (
	"context"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/getAlby/lndhub.go/lnd"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnwire"
	"github.com/lightningnetwork/lnd/zpay32"
	"google.golang.org/grpc"
)

type MockLND struct {
	Sub             *MockSubscribeInvoices
	fee             int64
	addIndexCounter uint64
}

type MockSubscribeInvoices struct {
	invoiceChan chan (*lnrpc.Invoice)
}

func (mockSub *MockSubscribeInvoices) Recv() (*lnrpc.Invoice, error) {
	inv := <-mockSub.invoiceChan
	return inv, nil
}

func (mlnd *MockLND) ListChannels(ctx context.Context, req *lnrpc.ListChannelsRequest, options ...grpc.CallOption) (*lnrpc.ListChannelsResponse, error) {
	return &lnrpc.ListChannelsResponse{
		Channels: []*lnrpc.Channel{},
	}, nil
}

func (mlnd *MockLND) SendPaymentSync(ctx context.Context, req *lnrpc.SendRequest, options ...grpc.CallOption) (*lnrpc.SendResponse, error) {
	return &lnrpc.SendResponse{
		PaymentError:    "",
		PaymentPreimage: []byte("preimage"),
		PaymentRoute: &lnrpc.Route{
			TotalTimeLock: 0,
			TotalFees:     mlnd.fee,
			TotalAmt:      req.Amt + mlnd.fee,
			Hops:          []*lnrpc.Hop{},
			TotalFeesMsat: 1000 * mlnd.fee,
			TotalAmtMsat:  1000 * (req.Amt + mlnd.fee),
		},
		PaymentHash: req.PaymentHash,
	}, nil
}

func (mlnd *MockLND) AddInvoice(ctx context.Context, req *lnrpc.Invoice, options ...grpc.CallOption) (*lnrpc.AddInvoiceResponse, error) {
	msat := lnwire.MilliSatoshi(req.ValueMsat)
	invoice := &zpay32.Invoice{
		Net:             &chaincfg.Params{},
		MilliSat:        &msat,
		Timestamp:       time.Now(),
		PaymentHash:     &[32]byte{},
		PaymentAddr:     &[32]byte{},
		Destination:     &btcec.PublicKey{},
		Description:     new(string),
		DescriptionHash: &[32]byte{},
		FallbackAddr:    nil,
	}
	copy(req.RHash, invoice.PaymentHash[:])
	copy(req.PaymentAddr, invoice.PaymentAddr[:])
	copy(req.DescriptionHash, invoice.DescriptionHash[:])
	pr, err := invoice.Encode(zpay32.MessageSigner{
		SignCompact: func(msg []byte) ([]byte, error) {
			return []byte{}, nil
		},
	})
	if err != nil {
		return nil, err
	}
	mlnd.addIndexCounter += 1
	return &lnrpc.AddInvoiceResponse{
		RHash:          req.RHash,
		PaymentRequest: pr,
		AddIndex:       mlnd.addIndexCounter,
		PaymentAddr:    []byte{},
	}, nil
}

func (mlnd *MockLND) SubscribeInvoices(ctx context.Context, req *lnrpc.InvoiceSubscription, options ...grpc.CallOption) (lnd.SubscribeInvoicesWrapper, error) {
	return mlnd.Sub, nil
}

func (mlnd *MockLND) GetInfo(ctx context.Context, req *lnrpc.GetInfoRequest, options ...grpc.CallOption) (*lnrpc.GetInfoResponse, error) {
	return &lnrpc.GetInfoResponse{
		Version:             "v1.0.0",
		CommitHash:          "abc123",
		IdentityPubkey:      "123pubkey",
		Alias:               "Mocky McMockface",
		Color:               "",
		NumPendingChannels:  1,
		NumActiveChannels:   10,
		NumInactiveChannels: 3,
		NumPeers:            10,
		BlockHeight:         1000,
		BlockHash:           "hashhashash",
		BestHeaderTimestamp: 123456,
		SyncedToChain:       true,
		SyncedToGraph:       true,
		Testnet:             false,
		Chains: []*lnrpc.Chain{{
			Chain:   "BTC",
			Network: "mainnet",
		}},
		Uris:     []string{"https://mocky.mcmockface.com"},
		Features: map[uint32]*lnrpc.Feature{},
	}, nil
}

func (mlnd *MockLND) DecodeBolt11(ctx context.Context, bolt11 string, options ...grpc.CallOption) (*lnrpc.PayReq, error) {
	inv, err := zpay32.Decode(bolt11, &chaincfg.MainNetParams)
	if err != nil {
		return nil, err
	}
	return &lnrpc.PayReq{
		Destination:     string(inv.Destination.SerializeCompressed()),
		PaymentHash:     string(inv.PaymentHash[:]),
		NumSatoshis:     int64(*inv.MilliSat) / 1000,
		Timestamp:       inv.Timestamp.Unix(),
		Expiry:          int64(inv.Expiry()),
		Description:     *inv.Description,
		DescriptionHash: string(inv.DescriptionHash[:]),
		FallbackAddr:    inv.FallbackAddr.EncodeAddress(),
		CltvExpiry:      int64(inv.MinFinalCLTVExpiry()),
		RouteHints:      []*lnrpc.RouteHint{},
		PaymentAddr:     []byte{},
		NumMsat:         int64(*inv.MilliSat),
		Features:        map[uint32]*lnrpc.Feature{},
	}, nil
}
