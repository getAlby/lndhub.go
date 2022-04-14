package cln

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"

	"github.com/getAlby/lndhub.go/lnd"
	"github.com/lightningnetwork/lnd/lnrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	MSAT_PER_SAT = 1000
)

type CLNClient struct {
	client  NodeClient
	handler *InvoiceHandler
}

type InvoiceHandler struct {
	invoiceChan chan (*lnrpc.Invoice)
}
type CLNClientOptions struct {
	Host          string
	CaCertHex     string
	ClientCertHex string
	ClientKeyHex  string
}

func loadTLSCredentials(caCertHex, clientCertHex, clientKeyHex string) (credentials.TransportCredentials, error) {
	// Load certificate of the CA who signed server's certificate
	pemServerCA, err := hex.DecodeString(caCertHex)
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pemServerCA) {
		return nil, fmt.Errorf("failed to add server CA's certificate")
	}

	// Load client's certificate and private key
	clientCert, err := hex.DecodeString(clientCertHex)
	if err != nil {
		return nil, err
	}
	clientKey, err := hex.DecodeString(clientKeyHex)
	if err != nil {
		return nil, err
	}
	clientKeyPair, err := tls.X509KeyPair(clientCert, clientKey)
	if err != nil {
		return nil, err
	}

	// Create the credentials and return it
	config := &tls.Config{
		Certificates: []tls.Certificate{clientKeyPair},
		RootCAs:      certPool,
	}
	return credentials.NewTLS(config), nil
}

func NewCLNClient(options CLNClientOptions) (*CLNClient, error) {
	credentials, err := loadTLSCredentials(options.CaCertHex, options.ClientCertHex, options.ClientKeyHex)
	if err != nil {
		return nil, err
	}
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials),
	}

	conn, err := grpc.Dial(options.Host, opts...)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}
	return &CLNClient{
		client: NewNodeClient(conn),
	}, nil
}

func (cln *CLNClient) Recv() (invoice *lnrpc.Invoice, err error) {
	return <-cln.handler.invoiceChan, nil
}

func (cl *CLNClient) ListChannels(ctx context.Context, req *lnrpc.ListChannelsRequest, options ...grpc.CallOption) (*lnrpc.ListChannelsResponse, error) {
	peers, err := cl.client.ListPeers(ctx, &ListpeersRequest{})
	if err != nil {
		return nil, err
	}
	channels := []*lnrpc.Channel{}
	for _, p := range peers.Peers {
		for _, ch := range p.Channels {
			channels = append(channels, &lnrpc.Channel{
				Active:        p.Connected,
				RemotePubkey:  string(p.Id),
				ChannelPoint:  fmt.Sprintf("%s:%d", string(ch.FundingTxid), *ch.FundingOutnum),
				ChanId:        convertChanId(ch.ShortChannelId),
				Capacity:      int64(ch.TotalMsat.Msat / MSAT_PER_SAT),
				LocalBalance:  int64(ch.ToUsMsat.Msat / MSAT_PER_SAT),
				RemoteBalance: int64(ch.ReceivableMsat.Msat / MSAT_PER_SAT),
			})
		}
	}
	return &lnrpc.ListChannelsResponse{
		Channels: channels,
	}, nil
}

func convertChanId(in *string) (out uint64) {
	if in == nil {
		return 0
	}
	return 0
}

func (cl *CLNClient) SendPaymentSync(ctx context.Context, req *lnrpc.SendRequest, options ...grpc.CallOption) (*lnrpc.SendResponse, error) {

	return nil, nil
}

func (cl *CLNClient) AddInvoice(ctx context.Context, req *lnrpc.Invoice, options ...grpc.CallOption) (*lnrpc.AddInvoiceResponse, error) {
	return nil, nil
}

func (cl *CLNClient) SubscribeInvoices(ctx context.Context, req *lnrpc.InvoiceSubscription, options ...grpc.CallOption) (lnd.SubscribeInvoicesWrapper, error) {
	//cl.client.LastInvoiceIndex = int(req.AddIndex)
	//cl.client.ListenForInvoices()
	return cl, nil
}

func (cl *CLNClient) GetInfo(ctx context.Context, req *lnrpc.GetInfoRequest, options ...grpc.CallOption) (*lnrpc.GetInfoResponse, error) {
	info, err := cl.client.Getinfo(ctx, &GetinfoRequest{})
	if err != nil {
		return nil, err
	}
	return &lnrpc.GetInfoResponse{
		Version:             info.Version,
		CommitHash:          info.Version,
		IdentityPubkey:      string(info.Id),
		Alias:               info.Alias,
		Color:               string(info.Color),
		NumPendingChannels:  info.NumPendingChannels,
		NumActiveChannels:   info.NumActiveChannels,
		NumInactiveChannels: info.NumInactiveChannels,
		NumPeers:            info.NumPeers,
		BlockHeight:         info.Blockheight,
		BlockHash:           "",
		BestHeaderTimestamp: 0,
		SyncedToChain:       true,
		SyncedToGraph:       true,
		Testnet:             info.Network == "mainnet",
		Chains: []*lnrpc.Chain{
			{
				Chain:   "bitcoin",
				Network: "mainnet",
			},
		},
	}, nil
}

func (cl *CLNClient) DecodeBolt11(ctx context.Context, bolt11 string, options ...grpc.CallOption) (*lnrpc.PayReq, error) {
	return nil, nil
}
