package lnd

import (
	"context"
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/ziflex/lecho/v3"
	"gorm.io/gorm/logger"
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
			logger.Infof("Error connecting to node, node id %s, error %s", resp.IdentityPubkey, err.Error())
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
			logger.Infof("Node does not have enough active channels yet, node id %s, active channels %d, total channels %d", resp.IdentityPubkey, nrActiveChannels, totalChannels)
			continue
		}
		//node is online and has enough active channels, set this node to active
		//log & send notification to Sentry in case we're switching
		if cluster.activeNode != node {
			cluster.activeNode = node
			message := fmt.Sprintf("Switched nodes: new node id %s", node.GetMainPubkey())
			logger.Info(message)
			sentry.CaptureMessage(message)
			break
		}
		i++

	}
	if i == len(cluster.nodes)-1 {
		message := "Cluster is offline, could not find an active node"
		logger.Info(message)
		sentry.CaptureMessage(message)
	}
}

//make cluster implement interface
//no subscriber functionality
//loop over members, use first online member to make the payment
//start loop to check cluster status every 30s
