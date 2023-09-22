package lnd

import (
	"context"
	"fmt"
	"strings"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnrpc/routerrpc"
	"github.com/ziflex/lecho/v3"
	"google.golang.org/grpc"
)

type LightningClientWrapper interface {
	ListChannels(ctx context.Context, req *lnrpc.ListChannelsRequest, options ...grpc.CallOption) (*lnrpc.ListChannelsResponse, error)
	SendPaymentSync(ctx context.Context, req *lnrpc.SendRequest, options ...grpc.CallOption) (*lnrpc.SendResponse, error)
	AddInvoice(ctx context.Context, req *lnrpc.Invoice, options ...grpc.CallOption) (*lnrpc.AddInvoiceResponse, error)
	SubscribeInvoices(ctx context.Context, req *lnrpc.InvoiceSubscription, options ...grpc.CallOption) (SubscribeInvoicesWrapper, error)
	SubscribePayment(ctx context.Context, req *routerrpc.TrackPaymentRequest, options ...grpc.CallOption) (SubscribePaymentWrapper, error)
	GetInfo(ctx context.Context, req *lnrpc.GetInfoRequest, options ...grpc.CallOption) (*lnrpc.GetInfoResponse, error)
	DecodeBolt11(ctx context.Context, bolt11 string, options ...grpc.CallOption) (*lnrpc.PayReq, error)
	IsIdentityPubkey(pubkey string) (isOurPubkey bool)
	GetMainPubkey() (pubkey string)
}

type SubscribeInvoicesWrapper interface {
	Recv() (*lnrpc.Invoice, error)
}
type SubscribePaymentWrapper interface {
	Recv() (*lnrpc.Payment, error)
}

func InitLNClient(c *Config, logger *lecho.Logger, ctx context.Context) (result LightningClientWrapper, err error) {
	switch c.LNClientType {
	case LND_CLIENT_TYPE:
		return InitSingleLNDClient(c, ctx)
	case LND_CLUSTER_CLIENT_TYPE:
		return InitLNDCluster(c, logger, ctx)
	default:
		return nil, fmt.Errorf("Did not recognize LN client type %s", c.LNClientType)
	}
}

func InitSingleLNDClient(c *Config, ctx context.Context) (result LightningClientWrapper, err error) {
	client, err := NewLNDclient(LNDoptions{
		Address:      c.LNDAddress,
		MacaroonFile: c.LNDMacaroonFile,
		MacaroonHex:  c.LNDMacaroonHex,
		CertFile:     c.LNDCertFile,
		CertHex:      c.LNDCertHex,
	}, ctx)
	if err != nil {
		return nil, err
	}
	getInfo, err := client.GetInfo(ctx, &lnrpc.GetInfoRequest{})
	if err != nil {
		return nil, err
	}
	client.IdentityPubkey = getInfo.IdentityPubkey
	return client, nil
}
func InitLNDCluster(c *Config, logger *lecho.Logger, ctx context.Context) (result LightningClientWrapper, err error) {
	nodes := []LightningClientWrapper{}
	//interpret lnd address, macaroon file & cert file as comma seperated values
	addresses := strings.Split(c.LNDAddress, ",")
	macaroons := strings.Split(c.LNDMacaroonFile, ",")
	certs := strings.Split(c.LNDCertFile, ",")
	if len(addresses) != len(macaroons) || len(addresses) != len(certs) || len(certs) != len(macaroons) {
		return nil, fmt.Errorf("Error parsing LND cluster config: addresses, macaroons or certs array length mismatch")
	}
	for i := 0; i < len(addresses); i++ {
		n, err := NewLNDclient(LNDoptions{
			Address:      addresses[i],
			MacaroonFile: macaroons[i],
			CertFile:     certs[i],
		}, ctx)
		if err != nil {
			return nil, err
		}
		getInfo, err := n.GetInfo(ctx, &lnrpc.GetInfoRequest{})
		if err != nil {
			return nil, err
		}
		n.IdentityPubkey = getInfo.IdentityPubkey
		nodes = append(nodes, n)
	}
	logger.Infof("Initialized LND cluster with %d nodes", len(nodes))
	cluster := &LNDCluster{
		Nodes:               nodes,
		ActiveChannelRatio:  c.LNDClusterActiveChannelRatio,
		ActiveNode:          nodes[0],
		Logger:              logger,
		LivenessCheckPeriod: c.LNDClusterLivenessPeriod,
	}
	//start liveness check
	go cluster.StartLivenessLoop(ctx)
	return cluster, nil
}
