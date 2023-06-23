package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/getAlby/lndhub.go/lnd"
	"github.com/ziflex/lecho/v3"
)

func InitLNClient(c *service.Config, logger *lecho.Logger, ctx context.Context) (result lnd.LightningClientWrapper, err error) {
	switch c.LNClientType {
	case service.LND_CLIENT_TYPE:
		return lnd.NewLNDclient(lnd.LNDoptions{
			Address:      c.LNDAddress,
			MacaroonFile: c.LNDMacaroonFile,
			MacaroonHex:  c.LNDMacaroonHex,
			CertFile:     c.LNDCertFile,
			CertHex:      c.LNDCertHex,
		}, ctx)
	case service.LND_CLUSTER_CLIENT_TYPE:
		return InitLNDCluster(c, logger, ctx)
	default:
		return nil, fmt.Errorf("Did not recognize LN client type %s", c.LNClientType)
	}
}
func InitLNDCluster(c *service.Config, logger *lecho.Logger, ctx context.Context) (result lnd.LightningClientWrapper, err error) {
	nodes := []*lnd.LNDWrapper{}
	//interpret lnd address, macaroon file & cert file as comma seperated values
	addresses := strings.Split(c.LNDAddress, ",")
	macaroons := strings.Split(c.LNDMacaroonFile, ",")
	certs := strings.Split(c.LNDCertFile, ",")
	if len(addresses) != len(macaroons) || len(addresses) != len(certs) || len(certs) != len(macaroons) {
		return nil, fmt.Errorf("Error parsing LND cluster config: addresses, macaroons or certs array length mismatch")
	}
	for i := 0; i < len(addresses); i++ {
		n, err := lnd.NewLNDclient(lnd.LNDoptions{
			Address:      addresses[i],
			MacaroonFile: macaroons[i],
			CertFile:     certs[i],
		}, ctx)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	logger.Infof("Initialized LND cluster with %d nodes", len(nodes))
	cluster := &lnd.LNDCluster{
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
