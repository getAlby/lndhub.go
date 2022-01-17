package lnd

import (
	"context"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"io/ioutil"

	"github.com/lightningnetwork/lnd/lnrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/macaroon.v2"
)

// LNDoptions are the options for the connection to the lnd node.
type LNDoptions struct {
	Address      string
	CertFile     string
	CertHex      string
	MacaroonFile string
	MacaroonHex  string
}

type LNDclient struct {
	lndClient lnrpc.LightningClient
	ctx       context.Context
	conn      *grpc.ClientConn
}

func NewLNDclient(lndOptions LNDoptions) (LNDclient, error) {
	result := LNDclient{}

	// Get credentials either from a hex string or a file
	var creds credentials.TransportCredentials
	// if a hex string is provided
	if lndOptions.CertHex != "" {
		cp := x509.NewCertPool()
		cert, err := hex.DecodeString(lndOptions.CertHex)
		if err != nil {
			return result, err
		}
		cp.AppendCertsFromPEM(cert)
		creds = credentials.NewClientTLSFromCert(cp, "")
		// if a path to a cert file is provided
	} else if lndOptions.CertFile != "" {
		credsFromFile, err := credentials.NewClientTLSFromFile(lndOptions.CertFile, "")
		if err != nil {
			return result, err
		}
		creds = credsFromFile // make it available outside of the else if block
	}
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
	}

	var macaroonData []byte
	if lndOptions.MacaroonHex != "" {
		macBytes, err := hex.DecodeString(lndOptions.MacaroonHex)
		if err != nil {
			return result, err
		}
		macaroonData = macBytes
	} else if lndOptions.MacaroonFile != "" {
		macBytes, err := ioutil.ReadFile(lndOptions.MacaroonFile)
		if err != nil {
			return result, err
		}
		macaroonData = macBytes // make it available outside of the else if block
	} else {
		return result, fmt.Errorf("LND macaroon is missing")
	}

	mac := &macaroon.Macaroon{}
	if err := mac.UnmarshalBinary(macaroonData); err != nil {
		return result, err
	}

	if creds == nil {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	}
	conn, err := grpc.Dial(lndOptions.Address, opts...)
	if err != nil {
		return result, err
	}

	c := lnrpc.NewLightningClient(conn)

	result = LNDclient{
		conn:      conn,
		ctx:       context.Background(),
		lndClient: c,
	}

	return result, nil
}
