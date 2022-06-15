package integration_tests

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"math/big"
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

const privkey = "0123456789abcdef"

func getPubkey() ecdsa.PublicKey {
	privKeyBytes, _ := hex.DecodeString(privkey)
	x, y := btcec.S256().ScalarBaseMult(privKeyBytes)
	return ecdsa.PublicKey{
		Curve: btcec.S256(),
		X:     x,
		Y:     y,
	}
}

func signMsg(msg []byte) ([]byte, error) {
	privKeyBytes, err := hex.DecodeString(privkey)
	if err != nil {
		return nil, err
	}
	ecdsaPrivKey := &ecdsa.PrivateKey{
		PublicKey: getPubkey(),
		D:         new(big.Int).SetBytes(privKeyBytes),
	}
	return btcec.SignCompact(btcec.S256(), (*btcec.PrivateKey)(ecdsaPrivKey),
		msg, true)
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
	pHash := sha256.New()
	pHash.Write(req.RPreimage)
	pHash.Sum(nil)
	msat := lnwire.MilliSatoshi(1000 * req.Value)
	invoice := &zpay32.Invoice{
		Net:         &chaincfg.RegressionNetParams,
		MilliSat:    &msat,
		Timestamp:   time.Now(),
		PaymentHash: &[32]byte{},
		PaymentAddr: &[32]byte{},
		Features: &lnwire.FeatureVector{
			RawFeatureVector: &lnwire.RawFeatureVector{},
		},
		FallbackAddr: nil,
	}
	copy(invoice.PaymentHash[:], pHash.Sum(nil))
	copy(invoice.PaymentAddr[:], req.PaymentAddr)
	if len(req.DescriptionHash) != 0 {
		invoice.DescriptionHash = &[32]byte{}
		copy(req.DescriptionHash, invoice.DescriptionHash[:])
	}
	if req.Memo != "" {
		invoice.Description = &req.Memo
	}
	pr, err := invoice.Encode(zpay32.MessageSigner{
		SignCompact: signMsg,
	})
	if err != nil {
		return nil, err
	}
	mlnd.addIndexCounter += 1
	return &lnrpc.AddInvoiceResponse{
		RHash:          invoice.PaymentHash[:],
		PaymentRequest: pr,
		AddIndex:       mlnd.addIndexCounter,
	}, nil
}

func (mlnd *MockLND) mockPaidInvoice(added *ExpectedAddInvoiceResponseBody) error {
	inv, err := mlnd.DecodeBolt11(context.Background(), added.PayReq)
	if err != nil {
		return err
	}
	rhash, err := hex.DecodeString(added.RHash)
	if err != nil {
		return err
	}
	mlnd.Sub.invoiceChan <- &lnrpc.Invoice{
		Memo:            inv.Description,
		RPreimage:       []byte("123preimage"),
		RHash:           rhash,
		Value:           inv.NumSatoshis,
		ValueMsat:       inv.NumMsat,
		Settled:         true,
		CreationDate:    time.Now().Unix(),
		SettleDate:      time.Now().Unix(),
		PaymentRequest:  added.PayReq,
		DescriptionHash: []byte(inv.DescriptionHash),
		FallbackAddr:    inv.FallbackAddr,
		CltvExpiry:      uint64(inv.CltvExpiry),
		AmtPaid:         inv.NumSatoshis,
		AmtPaidSat:      inv.NumSatoshis,
		AmtPaidMsat:     inv.NumMsat,
		State:           lnrpc.Invoice_SETTLED,
		Htlcs:           []*lnrpc.InvoiceHTLC{},
		IsKeysend:       false,
	}
	return nil
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
			Network: "regtest",
		}},
		Uris:     []string{"https://mocky.mcmockface.com"},
		Features: map[uint32]*lnrpc.Feature{},
	}, nil
}

func (mlnd *MockLND) DecodeBolt11(ctx context.Context, bolt11 string, options ...grpc.CallOption) (*lnrpc.PayReq, error) {
	inv, err := zpay32.Decode(bolt11, &chaincfg.RegressionNetParams)
	if err != nil {
		return nil, err
	}
	result := &lnrpc.PayReq{
		Destination: string(inv.Destination.SerializeCompressed()),
		PaymentHash: string(inv.PaymentHash[:]),
		NumSatoshis: int64(*inv.MilliSat) / 1000,
		Timestamp:   inv.Timestamp.Unix(),
		Expiry:      int64(inv.Expiry()),
		Description: *inv.Description,
		CltvExpiry:  int64(inv.MinFinalCLTVExpiry()),
		RouteHints:  []*lnrpc.RouteHint{},
		PaymentAddr: []byte{},
		NumMsat:     int64(*inv.MilliSat),
		Features:    map[uint32]*lnrpc.Feature{},
	}
	if inv.DescriptionHash != nil {
		result.DescriptionHash = string(inv.DescriptionHash[:])
	}
	return result, nil
}
