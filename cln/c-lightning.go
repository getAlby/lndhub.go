package cln

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"

	"github.com/getAlby/lndhub.go/lnd"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/tidwall/gjson"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	MSAT_PER_SAT = 1000
)

type CLNClient struct {
	client  NodeClient
	handler *InvoiceHandler
}

type InvoiceHandler struct {
	invoiceChan chan (*lnrpc.Invoice)
}
type CLNClientOptions struct {
	Host          string
	CaCertHex     string
	ClientCertHex string
	ClientKeyHex  string
}

func loadTLSCredentials(caCertHex, clientCertHex, clientKeyHex string) (credentials.TransportCredentials, error) {
	// Load certificate of the CA who signed server's certificate
	pemServerCA, err := hex.DecodeString(caCertHex)
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pemServerCA) {
		return nil, fmt.Errorf("failed to add server CA's certificate")
	}

	// Load client's certificate and private key
	clientCert, err := hex.DecodeString(clientCertHex)
	if err != nil {
		return nil, err
	}
	clientKey, err := hex.DecodeString(clientKeyHex)
	if err != nil {
		return nil, err
	}
	clientKeyPair, err := tls.X509KeyPair(clientCert, clientKey)
	if err != nil {
		return nil, err
	}

	// Create the credentials and return it
	config := &tls.Config{
		Certificates: []tls.Certificate{clientKeyPair},
		RootCAs:      certPool,
	}
	return credentials.NewTLS(config), nil
}

func NewCLNClient(options CLNClientOptions) (*CLNClient, error) {
	handler := &InvoiceHandler{
		invoiceChan: make(chan *lnrpc.Invoice),
	}
	credentials, err := loadTLSCredentials(options.CaCertHex, options.ClientCertHex, options.ClientKeyHex)
	if err != nil {
		return nil, err
	}
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials),
	}

	conn, err := grpc.Dial(options.Host, opts...)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}
	return &CLNClient{
		handler: handler,
		client:  NewNodeClient(conn),
	}, nil
}

func (cln *CLNClient) Recv() (invoice *lnrpc.Invoice, err error) {
	return <-cln.handler.invoiceChan, nil
}

func (handler *InvoiceHandler) Handle(res gjson.Result) {
	invoice := &lnrpc.Invoice{}
	handler.invoiceChan <- invoice
}

func (cl *CLNClient) ListChannels(ctx context.Context, req *lnrpc.ListChannelsRequest, options ...grpc.CallOption) (*lnrpc.ListChannelsResponse, error) {
	return nil, nil
}

func (cl *CLNClient) SendPaymentSync(ctx context.Context, req *lnrpc.SendRequest, options ...grpc.CallOption) (*lnrpc.SendResponse, error) {

	return nil, nil
}

func (cl *CLNClient) AddInvoice(ctx context.Context, req *lnrpc.Invoice, options ...grpc.CallOption) (*lnrpc.AddInvoiceResponse, error) {
	return nil, nil
}

func (cl *CLNClient) SubscribeInvoices(ctx context.Context, req *lnrpc.InvoiceSubscription, options ...grpc.CallOption) (lnd.SubscribeInvoicesWrapper, error) {
	//cl.client.LastInvoiceIndex = int(req.AddIndex)
	//cl.client.ListenForInvoices()
	return cl, nil
}

func (cl *CLNClient) GetInfo(ctx context.Context, req *lnrpc.GetInfoRequest, options ...grpc.CallOption) (*lnrpc.GetInfoResponse, error) {
	return nil, nil
}

func (cl *CLNClient) DecodeBolt11(ctx context.Context, bolt11 string, options ...grpc.CallOption) (*lnrpc.PayReq, error) {
	return nil, nil
}
