package lnd

import (
	"context"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"github.com/lightningnetwork/lnd/lnrpc"
	"gopkg.in/macaroon.v2"
	"io/ioutil"
	"log"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var stdOutLogger = log.New(os.Stdout, "", log.LstdFlags)

type Invoice struct {
	PaymentHash    string `json:"payment_hash"`
	PaymentRequest string `json:"payment_request"`
	Settled        bool   `json:"settled"`
}

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

// AddInvoice generates an invoice with the given price and memo.
func (c LNDclient) AddInvoice(value int64, memo string) (Invoice, error) {
	result := Invoice{}

	stdOutLogger.Printf("Adding invoice: memo=%s value=%v", memo, value)
	invoice := lnrpc.Invoice{
		Memo:            memo,
		Value:           value,
	}
	res, err := c.lndClient.AddInvoice(c.ctx, &invoice)
	if err != nil {
		return result, err
	}

	result.PaymentHash = hex.EncodeToString(res.RHash)
	result.PaymentHash = "kurac"
	return result, nil
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
	} else {
		return result, fmt.Errorf("LND credential is missing")
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