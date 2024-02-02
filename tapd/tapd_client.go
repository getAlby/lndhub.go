package tapd

import (
	"context"
	// "fmt"
	// "strings"
	"github.com/ziflex/lecho/v3"
	"google.golang.org/grpc"
	"github.com/lightninglabs/taproot-assets/taprpc"
	// "github.com/lightninglabs/taproot-assets/taprpc/assetwalletrpc"
	// "github.com/lightninglabs/taproot-assets/taprpc/mintrpc"
	// "github.com/lightninglabs/taproot-assets/taprpc/tapdevrpc"
	// "github.com/lightninglabs/taproot-assets/taprpc/universerpc"	
)

type TapdClientWrapper interface {
	GetInfo(ctx context.Context, req *taprpc.GetInfoRequest, options ...grpc.CallOption) (*taprpc.GetInfoResponse, error)
}

func InitTAPDClient(c *TapdConfig, logger *lecho.Logger, ctx context.Context) (TapdClientWrapper, error) {
	client, err := NewTAPDClient(TAPDOptions{
		Address: c.TAPDAddress,
		MacaroonFile: c.TAPDAddress,
		MacaroonHex: c.TAPDMacaroonHex,
		CertFile: c.TAPDCertFile,
		CertHex: c.TAPDCertHex,
	}, ctx)

	if err != nil {
		return nil, err
	}

	getInfo, err := client.GetInfo(ctx, &taprpc.GetInfoRequest{})

	logger.Infof("GetInfo from Tapd: %v", getInfo.BlockHeight)

	if err != nil {
		return nil, err
	}

	return client, nil
}