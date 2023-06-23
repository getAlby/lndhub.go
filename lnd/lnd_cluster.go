package lnd

import (
	"context"
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnrpc/routerrpc"
	"github.com/ziflex/lecho/v3"
	"google.golang.org/grpc"
)

type LNDCluster struct {
	nodes               []*LNDWrapper
	activeNode          *LNDWrapper
	activeChannelRatio  float64
	logger              *lecho.Logger
	livenessCheckPeriod int
}

func (cluster *LNDCluster) startLivenessLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(cluster.livenessCheckPeriod) * time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cluster.checkClusterStatus(ctx)
		}
	}
}
func (cluster *LNDCluster) checkClusterStatus(ctx context.Context) {
	//for all nodes
	var i int
	for _, node := range cluster.nodes {
		//call getinfo
		resp, err := node.GetInfo(ctx, &lnrpc.GetInfoRequest{})
		//if we get an error here, the node is probably offline
		//so we move to the next node
		if err != nil {
			cluster.logger.Infof("Error connecting to node, node id %s, error %s", resp.IdentityPubkey, err.Error())
			continue
		}
		//if the context has been canceled, return
		if ctx.Err() == context.Canceled {
			return
		}
		//if num_active_channels / num_total_channels < x % (50?)
		//not booted yet, go to next
		nrActiveChannels := resp.NumActiveChannels
		totalChannels := resp.NumActiveChannels + resp.NumInactiveChannels
		if float64(nrActiveChannels/totalChannels) < cluster.activeChannelRatio {
			cluster.logger.Infof("Node does not have enough active channels yet, node id %s, active channels %d, total channels %d", resp.IdentityPubkey, nrActiveChannels, totalChannels)
			continue
		}
		//node is online and has enough active channels, set this node to active
		//log & send notification to Sentry in case we're switching
		if cluster.activeNode != node {
			cluster.activeNode = node
			message := fmt.Sprintf("Switched nodes: new node id %s", node.GetMainPubkey())
			cluster.logger.Info(message)
			sentry.CaptureMessage(message)
			break
		}
		i++

	}
	if i == len(cluster.nodes)-1 {
		message := "Cluster is offline, could not find an active node"
		cluster.logger.Info(message)
		sentry.CaptureMessage(message)
	}
}
func (cluster *LNDCluster) ListChannels(ctx context.Context, req *lnrpc.ListChannelsRequest, options ...grpc.CallOption) (*lnrpc.ListChannelsResponse, error) {
	return cluster.activeNode.ListChannels(ctx, req, options...)
}

func (cluster *LNDCluster) SendPaymentSync(ctx context.Context, req *lnrpc.SendRequest, options ...grpc.CallOption) (*lnrpc.SendResponse, error) {
	return cluster.activeNode.SendPaymentSync(ctx, req, options...)
}

func (cluster *LNDCluster) AddInvoice(ctx context.Context, req *lnrpc.Invoice, options ...grpc.CallOption) (*lnrpc.AddInvoiceResponse, error) {
	return cluster.activeNode.AddInvoice(ctx, req, options...)
}

func (cluster *LNDCluster) SubscribeInvoices(ctx context.Context, req *lnrpc.InvoiceSubscription, options ...grpc.CallOption) (SubscribeInvoicesWrapper, error) {
	return nil, fmt.Errorf("not implemented")
}

func (cluster *LNDCluster) SubscribePayment(ctx context.Context, req *routerrpc.TrackPaymentRequest, options ...grpc.CallOption) (SubscribePaymentWrapper, error) {
	return nil, fmt.Errorf("not implemented")
}

func (cluster *LNDCluster) GetInfo(ctx context.Context, req *lnrpc.GetInfoRequest, options ...grpc.CallOption) (*lnrpc.GetInfoResponse, error) {
	return cluster.activeNode.GetInfo(ctx, req, options...)
}

func (cluster *LNDCluster) DecodeBolt11(ctx context.Context, bolt11 string, options ...grpc.CallOption) (*lnrpc.PayReq, error) {
	return cluster.activeNode.DecodeBolt11(ctx, bolt11, options...)
}

func (cluster *LNDCluster) IsIdentityPubkey(pubkey string) (isOurPubkey bool) {
	for _, node := range cluster.nodes {
		if node.GetMainPubkey() == pubkey {
			return true
		}
	}
	return false
}

func (cluster *LNDCluster) GetMainPubkey() (pubkey string) {
	//the first node should always be our primary node
	//which we will use for our main pubkey
	return cluster.nodes[0].GetMainPubkey()
}
