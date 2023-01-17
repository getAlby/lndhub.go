package lnd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnrpc/routerrpc"
	"google.golang.org/grpc"
)

type EclairClient struct {
	host     string
	password string
}

func NewEclairClient(host, password string) *EclairClient {
	return &EclairClient{
		host:     host,
		password: password,
	}
}

func (eclair *EclairClient) ListChannels(ctx context.Context, req *lnrpc.ListChannelsRequest, options ...grpc.CallOption) (*lnrpc.ListChannelsResponse, error) {
	panic("not implemented") // TODO: Implement
}

func (eclair *EclairClient) SendPaymentSync(ctx context.Context, req *lnrpc.SendRequest, options ...grpc.CallOption) (*lnrpc.SendResponse, error) {
	panic("not implemented") // TODO: Implement
}

func (eclair *EclairClient) AddInvoice(ctx context.Context, req *lnrpc.Invoice, options ...grpc.CallOption) (*lnrpc.AddInvoiceResponse, error) {
	panic("not implemented") // TODO: Implement
}

func (eclair *EclairClient) SubscribeInvoices(ctx context.Context, req *lnrpc.InvoiceSubscription, options ...grpc.CallOption) (SubscribeInvoicesWrapper, error) {
	panic("not implemented") // TODO: Implement
}

func (eclair *EclairClient) SubscribePayment(ctx context.Context, req *routerrpc.TrackPaymentRequest, options ...grpc.CallOption) (SubscribePaymentWrapper, error) {
	panic("not implemented") // TODO: Implement
}

func (eclair *EclairClient) GetInfo(ctx context.Context, req *lnrpc.GetInfoRequest, options ...grpc.CallOption) (*lnrpc.GetInfoResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/getinfo", eclair.host), nil)
	httpReq.SetBasicAuth("", eclair.password)
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Got a bad http response status code from Eclair %d", resp.StatusCode)
	}
	info := InfoResponse{}
	json.NewDecoder(resp.Body).Decode(&info)
	return &lnrpc.GetInfoResponse{
		Version:             info.Version,
		CommitHash:          "",
		IdentityPubkey:      info.NodeID,
		Alias:               info.Alias,
		Color:               info.Color,
		NumPendingChannels:  0,
		NumActiveChannels:   0,
		NumInactiveChannels: 0,
		NumPeers:            0,
		BlockHeight:         uint32(info.BlockHeight),
		BlockHash:           "",
		BestHeaderTimestamp: 0,
		SyncedToChain:       false,
		SyncedToGraph:       false,
		Testnet:             false,
		Chains: []*lnrpc.Chain{{
			Chain:   "bitcoin",
			Network: info.Network,
		}},
		Uris:                   []string{},
		Features:               map[uint32]*lnrpc.Feature{},
		RequireHtlcInterceptor: false,
	}, nil
}

func (eclair *EclairClient) DecodeBolt11(ctx context.Context, bolt11 string, options ...grpc.CallOption) (*lnrpc.PayReq, error) {
	panic("not implemented") // TODO: Implement
}

type InfoResponse struct {
	Version  string `json:"version"`
	NodeID   string `json:"nodeId"`
	Alias    string `json:"alias"`
	Color    string `json:"color"`
	Features struct {
		Activated struct {
			OptionOnionMessages        string `json:"option_onion_messages"`
			GossipQueriesEx            string `json:"gossip_queries_ex"`
			OptionPaymentMetadata      string `json:"option_payment_metadata"`
			OptionDataLossProtect      string `json:"option_data_loss_protect"`
			VarOnionOptin              string `json:"var_onion_optin"`
			OptionStaticRemotekey      string `json:"option_static_remotekey"`
			OptionSupportLargeChannel  string `json:"option_support_large_channel"`
			OptionAnchorsZeroFeeHtlcTx string `json:"option_anchors_zero_fee_htlc_tx"`
			PaymentSecret              string `json:"payment_secret"`
			OptionShutdownAnysegwit    string `json:"option_shutdown_anysegwit"`
			OptionChannelType          string `json:"option_channel_type"`
			BasicMpp                   string `json:"basic_mpp"`
			GossipQueries              string `json:"gossip_queries"`
		} `json:"activated"`
		Unknown []interface{} `json:"unknown"`
	} `json:"features"`
	ChainHash       string   `json:"chainHash"`
	Network         string   `json:"network"`
	BlockHeight     int      `json:"blockHeight"`
	PublicAddresses []string `json:"publicAddresses"`
	InstanceID      string   `json:"instanceId"`
}
