package lnd

import (
	"context"
	"fmt"

	cln "github.com/fiatjaf/lightningd-gjson-rpc"
	"github.com/gofrs/uuid"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/tidwall/gjson"
	"google.golang.org/grpc"
)

const (
	MSAT_PER_SAT = 1000
)

type CLNClient struct {
	client  *cln.Client
	handler *InvoiceHandler
}

type InvoiceHandler struct {
	invoiceChan chan (*lnrpc.Invoice)
}
type CLNClientOptions struct {
	SparkUrl   string
	SparkToken string
}

func NewCLNClient(options CLNClientOptions) (*CLNClient, error) {
	handler := &InvoiceHandler{
		invoiceChan: make(chan *lnrpc.Invoice),
	}
	return &CLNClient{
		handler: handler,
		client: &cln.Client{
			PaymentHandler: handler.Handle,
			//CallTimeout:           0,
			//Path:                  "",
			//LightningDir:          "",
			SparkURL:   options.SparkUrl,
			SparkToken: options.SparkToken,
		},
	}, nil
}

//todo handle errors?
func (cln *CLNClient) Recv() (invoice *lnrpc.Invoice, err error) {
	return <-cln.handler.invoiceChan, nil
}

func (handler *InvoiceHandler) Handle(res gjson.Result) {
	//todo missing or wrong fields
	invoice := &lnrpc.Invoice{
		Memo:            res.Get("description").String(),
		RPreimage:       []byte(res.Get("payment_preimage").String()),
		RHash:           []byte(res.Get("payment_hash").String()),
		Value:           res.Get("amount_msat").Int() / MSAT_PER_SAT,
		ValueMsat:       res.Get("amount_msat").Int(),
		Settled:         true,
		CreationDate:    0,
		SettleDate:      res.Get("paid_at").Int(),
		PaymentRequest:  res.Get("bolt11").String(),
		DescriptionHash: []byte{},
		Expiry:          0,
		FallbackAddr:    "",
		CltvExpiry:      0,
		RouteHints:      []*lnrpc.RouteHint{},
		Private:         false,
		AddIndex:        0,
		SettleIndex:     0,
		AmtPaid:         res.Get("amount_msat").Int() / MSAT_PER_SAT,
		AmtPaidSat:      res.Get("amount_msat").Int() / MSAT_PER_SAT,
		AmtPaidMsat:     res.Get("amount_msat").Int(),
		State:           lnrpc.Invoice_SETTLED,
		Htlcs:           []*lnrpc.InvoiceHTLC{},
		Features:        map[uint32]*lnrpc.Feature{},
		IsKeysend:       false,
		PaymentAddr:     []byte{},
		IsAmp:           false,
		AmpInvoiceState: map[string]*lnrpc.AMPInvoiceState{},
	}
	handler.invoiceChan <- invoice
}

func (cl *CLNClient) ListChannels(ctx context.Context, req *lnrpc.ListChannelsRequest, options ...grpc.CallOption) (*lnrpc.ListChannelsResponse, error) {
	result, err := cl.client.Call("listpeers")
	if err != nil {
		return nil, err
	}
	channels := []*lnrpc.Channel{}
	for _, peer := range result.Get("peers").Array() {
		for _, ch := range peer.Get("channels").Array() {
			//todo fill in missing fields
			channels = append(channels, &lnrpc.Channel{
				Active:                ch.Get("state").String() == "CHANNELD_NORMAL",
				RemotePubkey:          peer.Get("id").String(),
				ChannelPoint:          "",
				ChanId:                0,
				Capacity:              ch.Get("msatoshi_total").Int() / MSAT_PER_SAT,
				LocalBalance:          ch.Get("msatoshi_to_us").Int() / MSAT_PER_SAT,
				RemoteBalance:         ch.Get("receivable_msatoshi").Int() / MSAT_PER_SAT,
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
			})
		}
	}
	return &lnrpc.ListChannelsResponse{
		Channels: channels,
	}, nil
}

func (cl *CLNClient) SendPaymentSync(ctx context.Context, req *lnrpc.SendRequest, options ...grpc.CallOption) (*lnrpc.SendResponse, error) {
	//todo add other options
	result, err := cl.client.Call("pay", req.PaymentRequest)
	if err != nil {
		return nil, err
	}
	//todo failure modes
	return &lnrpc.SendResponse{
		PaymentError:    "",
		PaymentPreimage: []byte(result.Get("payment_preimage").String()),
		PaymentHash:     []byte(result.Get("payment_hash").String()),
	}, nil
}

func (cl *CLNClient) AddInvoice(ctx context.Context, req *lnrpc.Invoice, options ...grpc.CallOption) (*lnrpc.AddInvoiceResponse, error) {
	uuid, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	mSatAmt := MSAT_PER_SAT * req.Value
	res, err := cl.client.Call("invoicewithdescriptionhash", mSatAmt, uuid.String(), req.DescriptionHash)
	if err != nil {
		return nil, err
	}
	return &lnrpc.AddInvoiceResponse{
		RHash:          []byte(res.Get("payment_hash").String()),
		PaymentRequest: res.Get("bolt11").String(),
	}, nil
}

// Todo here: make CLNClient implement the interface (Recv())
// This method will read from a channel or block
// The handler function publishes on the channel on a received invoice
// set the client's invoice index to the one from req
func (cl *CLNClient) SubscribeInvoices(ctx context.Context, req *lnrpc.InvoiceSubscription, options ...grpc.CallOption) (SubscribeInvoicesWrapper, error) {
	return cl, nil
}

func (cl *CLNClient) GetInfo(ctx context.Context, req *lnrpc.GetInfoRequest, options ...grpc.CallOption) (*lnrpc.GetInfoResponse, error) {
	result, err := cl.client.Call("getinfo")
	if err != nil {
		return nil, err
	}
	uris := []string{}
	for _, addr := range result.Get("address").Array() {
		uris = append(uris, fmt.Sprintf("%s@%s:%s", result.Get("id").String(), addr.Get("address").String(), addr.Get("port").String()))
	}

	return &lnrpc.GetInfoResponse{
		Version:             result.Get("version").String(),
		IdentityPubkey:      result.Get("id").String(),
		Alias:               result.Get("alias").String(),
		Color:               result.Get("color").String(),
		NumPendingChannels:  uint32(result.Get("num_pending_channels").Int()),
		NumActiveChannels:   uint32(result.Get("num_active_channels").Int()),
		NumInactiveChannels: uint32(result.Get("num_inactive_channels").Int()),
		NumPeers:            uint32(result.Get("num_peers").Int()),
		BlockHeight:         uint32(result.Get("blockheight").Int()),
		// workaround
		SyncedToChain: true,
		SyncedToGraph: true,
		Testnet:       false,
		Chains: []*lnrpc.Chain{
			{
				Chain:   "bitcoin",
				Network: result.Get("network").String(),
			},
		},
		Uris: uris,
	}, nil
}
