package integration_tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lnd"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/stretchr/testify/assert"
)

func TestLNDCluster(t *testing.T) {
	mockClients := []lnd.LightningClientWrapper{}
	mockLND1, err := NewMockLND("1234567890abcdef", 0, make(chan (*lnrpc.Invoice)))
	assert.NoError(t, err)
	mockLND2, err := NewMockLND("1234567890abcdefff", 0, make(chan (*lnrpc.Invoice)))
	assert.NoError(t, err)
	mockClients = append(mockClients, mockLND1, mockLND2)
	lndMockCluster := lnd.LNDCluster{
		Nodes:               mockClients,
		ActiveNode:          mockClients[0],
		ActiveChannelRatio:  0.5,
		Logger:              lib.Logger(""),
		LivenessCheckPeriod: 1,
	}
	ctx, cancel := context.WithCancel(context.Background())
	go lndMockCluster.StartLivenessLoop(ctx)
	//record pubkey of lnd-1, lnd-2
	info1, err := mockLND1.GetInfo(ctx, &lnrpc.GetInfoRequest{})
	assert.NoError(t, err)
	info2, err := mockLND2.GetInfo(ctx, &lnrpc.GetInfoRequest{})
	assert.NoError(t, err)
	//call getinfo - should be lnd1 that responds
	resp, err := lndMockCluster.GetInfo(ctx, &lnrpc.GetInfoRequest{})
	assert.NoError(t, err)
	assert.Equal(t, info1.IdentityPubkey, resp.IdentityPubkey)
	//make lnd-1 return the error
	mockLND1.GetInfoError = fmt.Errorf("some error")
	//sleep a bit again
	time.Sleep(2 * time.Second)
	//call getinfo, should be lnd2 that responds
	resp, err = lndMockCluster.GetInfo(ctx, &lnrpc.GetInfoRequest{})
	assert.NoError(t, err)
	assert.Equal(t, info2.IdentityPubkey, resp.IdentityPubkey)
	//make lnd-1 return no error
	mockLND1.GetInfoError = nil
	//sleep a bit again
	time.Sleep(2 * time.Second)
	//call getinfo, should be lnd1 that responds
	resp, err = lndMockCluster.GetInfo(ctx, &lnrpc.GetInfoRequest{})
	assert.NoError(t, err)
	assert.Equal(t, info1.IdentityPubkey, resp.IdentityPubkey)
	cancel()
}
