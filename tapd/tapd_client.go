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
	"github.com/lightninglabs/taproot-assets/taprpc/universerpc"	
)

type TapdClientWrapper interface {
	GetInfo(ctx context.Context, req *taprpc.GetInfoRequest, options ...grpc.CallOption) (*taprpc.GetInfoResponse, error)
	ListAssets(ctx context.Context, req *taprpc.ListAssetRequest, options ...grpc.CallOption) (*taprpc.ListAssetResponse, error)
	ListBalances(ctx context.Context, req *taprpc.ListBalancesRequest, options ...grpc.CallOption) (*taprpc.ListBalancesResponse, error)
	//ListBalancesByAssetID(ctx context.Context, req *taprpc.ListBalancesRequest_AssetId, options ...grpc.CallOption) (*taprpc.ListBalancesResponse, error)
	NewAddress(ctx context.Context, req *taprpc.NewAddrRequest, options ...grpc.CallOption) (*taprpc.Addr, error)
	GetUniverseAssets(ctx context.Context, req *universerpc.AssetRootRequest, options ...grpc.CallOption) (*universerpc.AssetRootResponse, error)
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

	_, err = client.GetInfo(ctx, &taprpc.GetInfoRequest{})

	if err != nil {
		return nil, err
	}

	return client, nil
}