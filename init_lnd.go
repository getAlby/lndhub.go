package main

import (
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/getAlby/lndhub.go/lnd"
	"github.com/ziflex/lecho/v3"
)

func InitLNDCluster(c *service.Config, logger *lecho.Logger) (result lnd.LightningClientWrapper, err error) {
	nodes := []*lnd.LNDWrapper{}
	return &lnd.LNDCluster{
		Nodes:               nodes,
		ActiveChannelRatio:  0.5,
		Logger:              logger,
		LivenessCheckPeriod: 30,
	}, nil
}
