package tapd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"io/ioutil"
	"gopkg.in/macaroon.v2"
	"github.com/lightningnetwork/lnd/macaroons"
	"github.com/lightninglabs/taproot-assets/taprpc"
	"github.com/lightninglabs/taproot-assets/taprpc/assetwalletrpc"
	"github.com/lightninglabs/taproot-assets/taprpc/mintrpc"
	"github.com/lightninglabs/taproot-assets/taprpc/tapdevrpc"
	"github.com/lightninglabs/taproot-assets/taprpc/universerpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)
// * Pattern copied from lnd.go

// Several items to address
// * should there be some abstract struct (?)/interface that creates a logical tie between tapd and lnd?
// * should there be some struct that simply carries both, if there is enough calls that require both
type TAPDOptions struct {
	Address      string
	CertFile     string
	CertHex      string 
	MacaroonFile string
	MacaroonHex  string
}

type TAPDWrapper struct {
	client taprpc.TaprootAssetsClient
	assetWallet assetwalletrpc.AssetWalletClient
	mintClient mintrpc.MintClient
	devClient tapdevrpc.TapDevClient
	universeClient universerpc.UniverseClient
}

func NewTAPDClient(tapdOptions TAPDOptions, ctx context.Context) (*TAPDWrapper, error) {
	// Get credentials either from a hex string, a file or the system's certificate store
	var creds credentials.TransportCredentials
	// if a hex string is provided
	if tapdOptions.CertHex != "" {
		cp := x509.NewCertPool()
		cert, err := hex.DecodeString(tapdOptions.CertHex)
		if err != nil {
			return nil, err
		}
		cp.AppendCertsFromPEM(cert)
		creds = credentials.NewClientTLSFromCert(cp, "")
		// if a path to a cert file is provided
	} else if tapdOptions.CertFile != "" {
		credsFromFile, err := credentials.NewClientTLSFromFile(tapdOptions.CertFile, "")
		if err != nil {
			return nil, err
		}
		creds = credsFromFile // make it available outside of the else if block		
	} else {
		creds = credentials.NewTLS(&tls.Config{})
	}
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
	}

	var macaroonData []byte
	if tapdOptions.MacaroonHex != "" {
		macBytes, err := hex.DecodeString(tapdOptions.MacaroonHex)
		if err != nil {
			return nil, err
		}
		macaroonData = macBytes
	} else if tapdOptions.MacaroonFile != "" {
		// Upgrade this
		macBytes, err := ioutil.ReadFile(tapdOptions.MacaroonFile)
		if err != nil {
			return nil, err
		}
		macaroonData = macBytes // make it available outside of the else if block		
	} else {
		return nil, errors.New("TAPD macaroon is missing")
	}
	mac := &macaroon.Macaroon{}
	if err := mac.UnmarshalBinary(macaroonData); err != nil {
		return nil, err
	}
	macCred, err := macaroons.NewMacaroonCredential(mac)
	if err != nil {
		return nil, err
	}
	// connect and return application domain client
	opts = append(opts, grpc.WithPerRPCCredentials(macCred))	

	conn, err := grpc.Dial(tapdOptions.Address, opts...)
	if err != nil {
		return nil, err
	}

	tapdClient := taprpc.NewTaprootAssetsClient(conn)
	return &TAPDWrapper{
		client: tapdClient,
		assetWallet: assetwalletrpc.NewAssetWalletClient(conn),
		mintClient: mintrpc.NewMintClient(conn),
		devClient: tapdevrpc.NewTapDevClient(conn),
		universeClient: universerpc.NewUniverseClient(conn),
	}, nil
}

func (wrapper *TAPDWrapper) GetInfo(ctx context.Context, req *taprpc.GetInfoRequest, options ...grpc.CallOption) (*taprpc.GetInfoResponse, error) {
	return wrapper.client.GetInfo(ctx, req, options...)
}

func (wrapper *TAPDWrapper) ListAssets(ctx context.Context, req *taprpc.ListAssetRequest, options ...grpc.CallOption) (*taprpc.ListAssetResponse, error) {
	return wrapper.client.ListAssets(ctx, req, options...)
}

func (wrapper *TAPDWrapper) ListBalances(ctx context.Context, req *taprpc.ListBalancesRequest, options ...grpc.CallOption) (*taprpc.ListBalancesResponse, error) {
	return wrapper.client.ListBalances(ctx, req, options...)
}

func (wrapper *TAPDWrapper) GetUniverseAssets(ctx context.Context, req *universerpc.AssetRootRequest, options ...grpc.CallOption) (*universerpc.AssetRootResponse, error) {
	return wrapper.universeClient.AssetRoots(ctx, req, options...)
}

func (wrapper *TAPDWrapper) NewAddress(ctx context.Context, req *taprpc.NewAddrRequest, options ...grpc.CallOption) (*taprpc.Addr, error) {
	return wrapper.client.NewAddr(ctx, req, options...)
}
