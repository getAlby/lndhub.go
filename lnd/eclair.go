package lnd

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

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
	err := eclair.Request(ctx, http.MethodPost, "/channels", "", nil, &channels)
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
	payload := url.Values{}
	payload.Add("invoice", req.PaymentRequest)
	payload.Add("amountMsat", strconv.Itoa(int(req.Amt)*1000))
	payload.Add("maxFeeFlatSat", strconv.Itoa(int(req.FeeLimit.GetFixed())))
	payload.Add("blocking", "true") //wtf
	resp := &EclairPayResponse{}
	err := eclair.Request(ctx, http.MethodPost, "/payinvoice", "application/x-www-form-urlencoded", payload, resp)
	if err != nil {
		return nil, err
	}
	errString := ""
	if resp.Type == "payment-failed" && len(resp.Failures) > 0 {
		errString = resp.Failures[0].T
	}
	//todo sum all parts
	return &lnrpc.SendResponse{
		PaymentError:    errString,
		PaymentPreimage: []byte(resp.PaymentPreimage),
		PaymentRoute: &lnrpc.Route{
			TotalFees: int64(resp.Parts[0].FeesPaid),
			TotalAmt:  int64(resp.RecipientAmount) + int64(resp.Parts[0].FeesPaid),
		},
		PaymentHash: []byte(resp.PaymentHash),
	}, nil
}

func (eclair *EclairClient) AddInvoice(ctx context.Context, req *lnrpc.Invoice, options ...grpc.CallOption) (*lnrpc.AddInvoiceResponse, error) {
	payload := url.Values{}
	if req.Memo != "" {
		payload.Add("description", req.Memo)
	}
	if len(req.DescriptionHash) != 0 {
		payload.Add("descriptionHash", string(req.DescriptionHash))
	}
	payload.Add("amountMsat", strconv.Itoa(int(req.Value*1000)))
	payload.Add("paymentPreimage", hex.EncodeToString(req.RPreimage))
	payload.Add("expireIn", strconv.Itoa(int(req.Expiry)))
	invoice := &EclairInvoice{}
	err := eclair.Request(ctx, http.MethodPost, "/createinvoice", "application/x-www-form-urlencoded", payload, invoice)
	if err != nil {
		return nil, err
	}
	rHash, err := hex.DecodeString(invoice.PaymentHash)
	if err != nil {
		return nil, err
	}
	return &lnrpc.AddInvoiceResponse{
		RHash:          rHash,
		PaymentRequest: invoice.Serialized,
		AddIndex:       uint64(invoice.Timestamp),
	}, nil
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
	err := eclair.Request(ctx, http.MethodPost, "/getinfo", "", nil, &info)
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

func (eclair *EclairClient) Request(ctx context.Context, method, endpoint, contentType string, body url.Values, response interface{}) error {
	httpReq, err := http.NewRequestWithContext(ctx, method, fmt.Sprintf("%s%s", eclair.host, endpoint), strings.NewReader(body.Encode()))
	httpReq.Header.Set("Content-type", contentType)
	httpReq.SetBasicAuth("", eclair.password)
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		response := map[string]interface{}{}
		json.NewDecoder(resp.Body).Decode(&response)
		return fmt.Errorf("Got a bad http response status code from Eclair %d for request %s. Body: %s", resp.StatusCode, httpReq.URL, response)
	}
	return json.NewDecoder(resp.Body).Decode(response)
}

func (eclair *EclairClient) DecodeBolt11(ctx context.Context, bolt11 string, options ...grpc.CallOption) (*lnrpc.PayReq, error) {
	invoice := &EclairInvoice{}
	payload := url.Values{}
	payload.Add("invoice", bolt11)
	err := eclair.Request(ctx, http.MethodPost, "/parseinvoice", "application/x-www-form-urlencoded", payload, invoice)
	if err != nil {
		return nil, err
	}
	return &lnrpc.PayReq{
		Destination:     invoice.NodeID,
		PaymentHash:     invoice.PaymentHash,
		NumSatoshis:     int64(invoice.Amount) / 1000,
		Timestamp:       int64(invoice.Timestamp),
		Expiry:          int64(invoice.Expiry),
		Description:     invoice.Description,
		DescriptionHash: invoice.DescriptionHash,
		NumMsat:         int64(invoice.Amount),
	}, nil
}
