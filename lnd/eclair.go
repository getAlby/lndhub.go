package lnd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnrpc/routerrpc"
	"google.golang.org/grpc"
)

type EclairClient struct {
	host     string
	password string
}

type EclairInvoicesSubscriber struct {
	ctx context.Context
}

func (eis *EclairInvoicesSubscriber) Recv() (*lnrpc.Invoice, error) {
	//placeholder
	//block indefinitely
	<-eis.ctx.Done()
	return nil, fmt.Errorf("context canceled")
}

type EclairPaymentsTracker struct {
	ctx context.Context
}

func (ept *EclairPaymentsTracker) Recv() (*lnrpc.Payment, error) {
	//placeholder
	//block indefinitely
	<-ept.ctx.Done()
	return nil, fmt.Errorf("context canceled")
}

func NewEclairClient(host, password string) *EclairClient {
	return &EclairClient{
		host:     host,
		password: password,
	}
}

func (eclair *EclairClient) ListChannels(ctx context.Context, req *lnrpc.ListChannelsRequest, options ...grpc.CallOption) (*lnrpc.ListChannelsResponse, error) {
	channels := []EclairChannel{}
	err := eclair.Request(ctx, http.MethodPost, "/channels", nil, &channels)
	if err != nil {
		return nil, err
	}
	convertedChannels := []*lnrpc.Channel{}
	for _, ch := range channels {
		convertedChannels = append(convertedChannels, &lnrpc.Channel{
			Active:                ch.State == "NORMAL",
			RemotePubkey:          ch.NodeID,
			ChannelPoint:          "",
			ChanId:                0,
			Capacity:              int64(ch.Data.Commitments.LocalCommit.Spec.ToLocal)/1000 + int64(ch.Data.Commitments.LocalCommit.Spec.ToRemote)/1000,
			LocalBalance:          int64(ch.Data.Commitments.LocalCommit.Spec.ToLocal) / 1000,
			RemoteBalance:         int64(ch.Data.Commitments.LocalCommit.Spec.ToRemote) / 1000,
			CommitFee:             0,
			CommitWeight:          0,
			FeePerKw:              0,
			UnsettledBalance:      0,
			TotalSatoshisSent:     0,
			TotalSatoshisReceived: 0,
			NumUpdates:            0,
			PendingHtlcs:          []*lnrpc.HTLC{},
			CsvDelay:              0,
			Private:               false,
			Initiator:             false,
			ChanStatusFlags:       "",
			LocalChanReserveSat:   0,
			RemoteChanReserveSat:  0,
			StaticRemoteKey:       false,
			CommitmentType:        0,
			Lifetime:              0,
			Uptime:                0,
			CloseAddress:          "",
			PushAmountSat:         0,
			ThawHeight:            0,
			LocalConstraints:      &lnrpc.ChannelConstraints{},
			RemoteConstraints:     &lnrpc.ChannelConstraints{},
			AliasScids:            []uint64{},
			ZeroConf:              false,
			ZeroConfConfirmedScid: 0,
		})
	}
	return &lnrpc.ListChannelsResponse{
		Channels: convertedChannels,
	}, nil
}

func (eclair *EclairClient) SendPaymentSync(ctx context.Context, req *lnrpc.SendRequest, options ...grpc.CallOption) (*lnrpc.SendResponse, error) {
	panic("not implemented") // TODO: Implement
}

func (eclair *EclairClient) AddInvoice(ctx context.Context, req *lnrpc.Invoice, options ...grpc.CallOption) (*lnrpc.AddInvoiceResponse, error) {
	panic("not implemented") // TODO: Implement
}

func (eclair *EclairClient) SubscribeInvoices(ctx context.Context, req *lnrpc.InvoiceSubscription, options ...grpc.CallOption) (SubscribeInvoicesWrapper, error) {
	return &EclairInvoicesSubscriber{
		ctx: ctx,
	}, nil
}

func (eclair *EclairClient) SubscribePayment(ctx context.Context, req *routerrpc.TrackPaymentRequest, options ...grpc.CallOption) (SubscribePaymentWrapper, error) {
	return &EclairPaymentsTracker{
		ctx: ctx,
	}, nil
}

func (eclair *EclairClient) GetInfo(ctx context.Context, req *lnrpc.GetInfoRequest, options ...grpc.CallOption) (*lnrpc.GetInfoResponse, error) {
	info := EclairInfoResponse{}
	err := eclair.Request(ctx, http.MethodPost, "/getinfo", nil, &info)
	if err != nil {
		return nil, err
	}
	addresses := []string{}
	for _, addr := range info.PublicAddresses {
		addresses = append(addresses, fmt.Sprintf("%s@%s", info.NodeID, addr))
	}
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
		SyncedToChain:       true,
		SyncedToGraph:       true,
		Testnet:             info.Network == "testnet",
		Chains: []*lnrpc.Chain{{
			Chain:   "bitcoin",
			Network: info.Network,
		}},
		Uris:                   addresses,
		Features:               map[uint32]*lnrpc.Feature{},
		RequireHtlcInterceptor: false,
	}, nil
}

func (eclair *EclairClient) Request(ctx context.Context, method, endpoint string, body io.Reader, response interface{}) error {
	httpReq, err := http.NewRequestWithContext(ctx, method, fmt.Sprintf("%s%s", eclair.host, endpoint), body)
	httpReq.SetBasicAuth("", eclair.password)
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Got a bad http response status code from Eclair %d for request %s", resp.StatusCode, httpReq.URL)
	}
	return json.NewDecoder(resp.Body).Decode(response)
}

func (eclair *EclairClient) DecodeBolt11(ctx context.Context, bolt11 string, options ...grpc.CallOption) (*lnrpc.PayReq, error) {
	panic("not implemented") // TODO: Implement
}
