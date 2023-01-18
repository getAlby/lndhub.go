package lnd

import "time"

type EclairInfoResponse struct {
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

type EclairChannel struct {
	NodeID    string `json:"nodeId"`
	ChannelID string `json:"channelId"`
	State     string `json:"state"`
	Data      struct {
		Type        string `json:"type"`
		Commitments struct {
			ChannelID       string   `json:"channelId"`
			ChannelConfig   []string `json:"channelConfig"`
			ChannelFeatures []string `json:"channelFeatures"`
			LocalParams     struct {
				NodeID         string `json:"nodeId"`
				FundingKeyPath struct {
					Path []interface{} `json:"path"`
				} `json:"fundingKeyPath"`
				DustLimit                  int     `json:"dustLimit"`
				MaxHtlcValueInFlightMsat   float64 `json:"maxHtlcValueInFlightMsat"`
				RequestedChannelReserveOpt int     `json:"requestedChannelReserve_opt"`
				HtlcMinimum                int     `json:"htlcMinimum"`
				ToSelfDelay                int     `json:"toSelfDelay"`
				MaxAcceptedHtlcs           int     `json:"maxAcceptedHtlcs"`
				IsInitiator                bool    `json:"isInitiator"`
				DefaultFinalScriptPubKey   string  `json:"defaultFinalScriptPubKey"`
				InitFeatures               struct {
					Activated struct {
						OptionOnionMessages        string `json:"option_onion_messages"`
						GossipQueriesEx            string `json:"gossip_queries_ex"`
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
				} `json:"initFeatures"`
			} `json:"localParams"`
			RemoteParams struct {
				NodeID                     string  `json:"nodeId"`
				DustLimit                  int     `json:"dustLimit"`
				MaxHtlcValueInFlightMsat   float64 `json:"maxHtlcValueInFlightMsat"`
				RequestedChannelReserveOpt int     `json:"requestedChannelReserve_opt"`
				HtlcMinimum                int     `json:"htlcMinimum"`
				ToSelfDelay                int     `json:"toSelfDelay"`
				MaxAcceptedHtlcs           int     `json:"maxAcceptedHtlcs"`
				FundingPubKey              string  `json:"fundingPubKey"`
				RevocationBasepoint        string  `json:"revocationBasepoint"`
				PaymentBasepoint           string  `json:"paymentBasepoint"`
				DelayedPaymentBasepoint    string  `json:"delayedPaymentBasepoint"`
				HtlcBasepoint              string  `json:"htlcBasepoint"`
				InitFeatures               struct {
					Activated struct {
						OptionOnionMessages        string `json:"option_onion_messages"`
						GossipQueriesEx            string `json:"gossip_queries_ex"`
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
				} `json:"initFeatures"`
			} `json:"remoteParams"`
			ChannelFlags struct {
				AnnounceChannel bool `json:"announceChannel"`
			} `json:"channelFlags"`
			LocalCommit struct {
				Index int `json:"index"`
				Spec  struct {
					Htlcs           []interface{} `json:"htlcs"`
					CommitTxFeerate int           `json:"commitTxFeerate"`
					ToLocal         int           `json:"toLocal"`
					ToRemote        int           `json:"toRemote"`
				} `json:"spec"`
				CommitTxAndRemoteSig struct {
					CommitTx struct {
						Txid string `json:"txid"`
						Tx   string `json:"tx"`
					} `json:"commitTx"`
					RemoteSig string `json:"remoteSig"`
				} `json:"commitTxAndRemoteSig"`
				HtlcTxsAndRemoteSigs []interface{} `json:"htlcTxsAndRemoteSigs"`
			} `json:"localCommit"`
			RemoteCommit struct {
				Index int `json:"index"`
				Spec  struct {
					Htlcs           []interface{} `json:"htlcs"`
					CommitTxFeerate int           `json:"commitTxFeerate"`
					ToLocal         int           `json:"toLocal"`
					ToRemote        int           `json:"toRemote"`
				} `json:"spec"`
				Txid                     string `json:"txid"`
				RemotePerCommitmentPoint string `json:"remotePerCommitmentPoint"`
			} `json:"remoteCommit"`
			LocalChanges struct {
				Proposed []interface{} `json:"proposed"`
				Signed   []interface{} `json:"signed"`
				Acked    []interface{} `json:"acked"`
			} `json:"localChanges"`
			RemoteChanges struct {
				Proposed []interface{} `json:"proposed"`
				Acked    []interface{} `json:"acked"`
				Signed   []interface{} `json:"signed"`
			} `json:"remoteChanges"`
			LocalNextHtlcID  int `json:"localNextHtlcId"`
			RemoteNextHtlcID int `json:"remoteNextHtlcId"`
			OriginChannels   struct {
			} `json:"originChannels"`
			RemoteNextCommitInfo string `json:"remoteNextCommitInfo"`
			CommitInput          struct {
				OutPoint       string `json:"outPoint"`
				AmountSatoshis int    `json:"amountSatoshis"`
			} `json:"commitInput"`
			RemotePerCommitmentSecrets interface{} `json:"remotePerCommitmentSecrets"`
		} `json:"commitments"`
		ShortIds struct {
			Real struct {
				Status   string `json:"status"`
				RealScid string `json:"realScid"`
			} `json:"real"`
			LocalAlias  string `json:"localAlias"`
			RemoteAlias string `json:"remoteAlias"`
		} `json:"shortIds"`
		ChannelAnnouncement struct {
			NodeSignature1    string `json:"nodeSignature1"`
			NodeSignature2    string `json:"nodeSignature2"`
			BitcoinSignature1 string `json:"bitcoinSignature1"`
			BitcoinSignature2 string `json:"bitcoinSignature2"`
			Features          struct {
				Activated struct {
				} `json:"activated"`
				Unknown []interface{} `json:"unknown"`
			} `json:"features"`
			ChainHash      string `json:"chainHash"`
			ShortChannelID string `json:"shortChannelId"`
			NodeID1        string `json:"nodeId1"`
			NodeID2        string `json:"nodeId2"`
			BitcoinKey1    string `json:"bitcoinKey1"`
			BitcoinKey2    string `json:"bitcoinKey2"`
			TlvStream      struct {
				Records []interface{} `json:"records"`
				Unknown []interface{} `json:"unknown"`
			} `json:"tlvStream"`
		} `json:"channelAnnouncement"`
		ChannelUpdate struct {
			Signature      string `json:"signature"`
			ChainHash      string `json:"chainHash"`
			ShortChannelID string `json:"shortChannelId"`
			Timestamp      struct {
				Iso  time.Time `json:"iso"`
				Unix int       `json:"unix"`
			} `json:"timestamp"`
			MessageFlags struct {
				DontForward bool `json:"dontForward"`
			} `json:"messageFlags"`
			ChannelFlags struct {
				IsEnabled bool `json:"isEnabled"`
				IsNode1   bool `json:"isNode1"`
			} `json:"channelFlags"`
			CltvExpiryDelta           int `json:"cltvExpiryDelta"`
			HtlcMinimumMsat           int `json:"htlcMinimumMsat"`
			FeeBaseMsat               int `json:"feeBaseMsat"`
			FeeProportionalMillionths int `json:"feeProportionalMillionths"`
			HtlcMaximumMsat           int `json:"htlcMaximumMsat"`
			TlvStream                 struct {
				Records []interface{} `json:"records"`
				Unknown []interface{} `json:"unknown"`
			} `json:"tlvStream"`
		} `json:"channelUpdate"`
	} `json:"data"`
}

type EclairInvoice struct {
	Prefix             string `json:"prefix"`
	Timestamp          int    `json:"timestamp"`
	NodeID             string `json:"nodeId"`
	Serialized         string `json:"serialized"`
	Description        string `json:"description"`
	DescriptionHash    string `json:"descriptionHash"`
	PaymentHash        string `json:"paymentHash"`
	PaymentMetadata    string `json:"paymentMetadata"`
	Expiry             int    `json:"expiry"`
	MinFinalCltvExpiry int    `json:"minFinalCltvExpiry"`
	Amount             int    `json:"amount"`
	Features           struct {
		Activated struct {
			PaymentSecret         string `json:"payment_secret"`
			BasicMpp              string `json:"basic_mpp"`
			OptionPaymentMetadata string `json:"option_payment_metadata"`
			VarOnionOptin         string `json:"var_onion_optin"`
		} `json:"activated"`
		Unknown []interface{} `json:"unknown"`
	} `json:"features"`
	RoutingInfo []interface{} `json:"routingInfo"`
}

type EclairPayResponse struct {
	Type            string `json:"type"`
	ID              string `json:"id"`
	PaymentHash     string `json:"paymentHash"`
	PaymentPreimage string `json:"paymentPreimage"`
	RecipientAmount int    `json:"recipientAmount"`
	RecipientNodeID string `json:"recipientNodeId"`
	Failures        []struct {
		Amount int64         `json:"amount"`
		Route  []interface{} `json:"route"`
		T      string        `json:"t"`
	} `json:"failures"`
	Parts []struct {
		ID          string `json:"id"`
		Amount      int    `json:"amount"`
		FeesPaid    int    `json:"feesPaid"`
		ToChannelID string `json:"toChannelId"`
		Route       []struct {
			ShortChannelID string `json:"shortChannelId"`
			NodeID         string `json:"nodeId"`
			NextNodeID     string `json:"nextNodeId"`
			Params         struct {
				Type          string `json:"type"`
				ChannelUpdate struct {
					Signature      string `json:"signature"`
					ChainHash      string `json:"chainHash"`
					ShortChannelID string `json:"shortChannelId"`
					Timestamp      struct {
						Iso  time.Time `json:"iso"`
						Unix int       `json:"unix"`
					} `json:"timestamp"`
					MessageFlags struct {
						DontForward bool `json:"dontForward"`
					} `json:"messageFlags"`
					ChannelFlags struct {
						IsEnabled bool `json:"isEnabled"`
						IsNode1   bool `json:"isNode1"`
					} `json:"channelFlags"`
					CltvExpiryDelta           int   `json:"cltvExpiryDelta"`
					HtlcMinimumMsat           int   `json:"htlcMinimumMsat"`
					FeeBaseMsat               int   `json:"feeBaseMsat"`
					FeeProportionalMillionths int   `json:"feeProportionalMillionths"`
					HtlcMaximumMsat           int64 `json:"htlcMaximumMsat"`
					TlvStream                 struct {
						Records []interface{} `json:"records"`
						Unknown []interface{} `json:"unknown"`
					} `json:"tlvStream"`
				} `json:"channelUpdate"`
			} `json:"params"`
		} `json:"route"`
		Timestamp struct {
			Iso  time.Time `json:"iso"`
			Unix int       `json:"unix"`
		} `json:"timestamp"`
	} `json:"parts"`
}
