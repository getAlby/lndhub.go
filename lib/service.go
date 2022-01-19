package lib

import (
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/uptrace/bun"
)

type LndhubService struct {
	DB        *bun.DB
	LndClient *lnrpc.LightningClient
}
