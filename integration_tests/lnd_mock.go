package integration_tests

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"math/big"
	"time"

	btcec "github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/getAlby/lndhub.go/lnd"
	"github.com/labstack/gommon/random"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnrpc/routerrpc"
	"github.com/lightningnetwork/lnd/lnwire"
	"github.com/lightningnetwork/lnd/zpay32"
	"google.golang.org/grpc"
)

type MockLND struct {
	Sub             *MockSubscribeInvoices
	fee             int64
	privKey         *btcec.PrivateKey
	pubKey          *btcec.PublicKey
	addIndexCounter uint64
}

func NewMockLND(privkey string, fee int64, invoiceChan chan (*lnrpc.Invoice)) (*MockLND, error) {
	privKeyBytes, err := hex.DecodeString(privkey)
	if err != nil {
		return nil, err
	}
	privKey, pubKey := btcec.PrivKeyFromBytes(privKeyBytes)
	return &MockLND{
		Sub: &MockSubscribeInvoices{
			invoiceChan: invoiceChan,
		},
		fee:             fee,
		privKey:         privKey,
		pubKey:          pubKey,
		addIndexCounter: 0,
	}, nil
}

func (mlnd *MockLND) signMsg(msg []byte) ([]byte, error) {
	hash := sha256.Sum256(msg)
	return ecdsa.SignCompact(mlnd.privKey, hash[:], true)
}

type MockSubscribeInvoices struct {
	invoiceChan chan (*lnrpc.Invoice)
}

func (mockSub *MockSubscribeInvoices) Recv() (*lnrpc.Invoice, error) {
	inv := <-mockSub.invoiceChan
	return inv, nil
}
func (mlnd *MockLND) SubscribePayment(ctx context.Context, req *routerrpc.TrackPaymentRequest, options ...grpc.CallOption) (lnd.SubscribePaymentWrapper, error) {
	return nil, nil
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
	zpay32.Expiry(time.Duration(req.Expiry) * time.Second)(invoice)
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
		SignCompact: mlnd.signMsg,
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

func (mlnd *MockLND) mockPaidInvoice(added *ExpectedAddInvoiceResponseBody, amtPaid int64, keysend bool, htlc *lnrpc.InvoiceHTLC) error {
	var incoming *lnrpc.Invoice
	if !keysend {
		rhash, err := hex.DecodeString(added.RHash)
		if err != nil {
			return err
		}
		inv, err := mlnd.DecodeBolt11(context.Background(), added.PayReq)
		if err != nil {
			return err
		}
		incoming = &lnrpc.Invoice{
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
			IsKeysend:       keysend,
		}
	} else {
		preimage, err := makePreimageHex()
		if err != nil {
			return err
		}
		pHash := sha256.New()
		pHash.Write(preimage)
		incoming = &lnrpc.Invoice{
			Memo:           "",
			RPreimage:      preimage,
			RHash:          pHash.Sum(nil),
			Value:          amtPaid,
			ValueMsat:      1000 * amtPaid,
			Settled:        true,
			CreationDate:   time.Now().Unix(),
			SettleDate:     time.Now().Unix(),
			PaymentRequest: "",
			AmtPaid:        amtPaid,
			AmtPaidSat:     amtPaid,
			AmtPaidMsat:    1000 * amtPaid,
			State:          lnrpc.Invoice_SETTLED,
			Htlcs:          []*lnrpc.InvoiceHTLC{htlc},
			IsKeysend:      keysend,
		}
	}

	if amtPaid != 0 {
		incoming.AmtPaidSat = amtPaid
		incoming.AmtPaidMsat = 1000 * amtPaid
	}
	mlnd.Sub.invoiceChan <- incoming
	return nil
}

func (mlnd *MockLND) SubscribeInvoices(ctx context.Context, req *lnrpc.InvoiceSubscription, options ...grpc.CallOption) (lnd.SubscribeInvoicesWrapper, error) {
	return mlnd.Sub, nil
}

func (mlnd *MockLND) GetInfo(ctx context.Context, req *lnrpc.GetInfoRequest, options ...grpc.CallOption) (*lnrpc.GetInfoResponse, error) {

	return &lnrpc.GetInfoResponse{
		Version:             "v1.0.0",
		CommitHash:          "abc123",
		IdentityPubkey:      hex.EncodeToString(mlnd.pubKey.SerializeCompressed()),
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

func (mlnd *MockLND) TrackPayment(ctx context.Context, hash []byte, options ...grpc.CallOption) (*lnrpc.Payment, error) {
	return nil, nil
}

func (mlnd *MockLND) DecodeBolt11(ctx context.Context, bolt11 string, options ...grpc.CallOption) (*lnrpc.PayReq, error) {
	inv, err := zpay32.Decode(bolt11, &chaincfg.RegressionNetParams)
	if err != nil {
		return nil, err
	}
	result := &lnrpc.PayReq{
		Destination: hex.EncodeToString(inv.Destination.SerializeCompressed()),
		PaymentHash: hex.EncodeToString(inv.PaymentHash[:]),
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
func makePreimageHex() ([]byte, error) {
	return randBytesFromStr(32, random.Hex)
}
func randBytesFromStr(length int, from string) ([]byte, error) {
	b := make([]byte, length)
	fromLenBigInt := big.NewInt(int64(len(from)))
	for i := range b {
		r, err := rand.Int(rand.Reader, fromLenBigInt)
		if err != nil {
			return nil, err
		}
		b[i] = from[r.Int64()]
	}
	return b, nil
}

func newDefaultMockLND() *MockLND {
	mockLND, err := NewMockLND("1234567890abcdef", 0, make(chan (*lnrpc.Invoice)))
	if err != nil {
		log.Fatalf("Error initializing test service: %v", err)
	}
	return mockLND
}
