package controllers

import (
	"net/http"

	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
)

//Copy over struct for swagger purposes
type GetInfoResponse struct {
	// The version of the LND software that the node is running.
	Version string `protobuf:"bytes,14,opt,name=version,proto3" json:"version,omitempty"`
	// The SHA1 commit hash that the daemon is compiled with.
	CommitHash string `protobuf:"bytes,20,opt,name=commit_hash,json=commitHash,proto3" json:"commit_hash,omitempty"`
	// The identity pubkey of the current node.
	IdentityPubkey string `protobuf:"bytes,1,opt,name=identity_pubkey,json=identityPubkey,proto3" json:"identity_pubkey,omitempty"`
	// If applicable, the alias of the current node, e.g. "bob"
	Alias string `protobuf:"bytes,2,opt,name=alias,proto3" json:"alias,omitempty"`
	// The color of the current node in hex code format
	Color string `protobuf:"bytes,17,opt,name=color,proto3" json:"color,omitempty"`
	// Number of pending channels
	NumPendingChannels uint32 `protobuf:"varint,3,opt,name=num_pending_channels,json=numPendingChannels,proto3" json:"num_pending_channels,omitempty"`
	// Number of active channels
	NumActiveChannels uint32 `protobuf:"varint,4,opt,name=num_active_channels,json=numActiveChannels,proto3" json:"num_active_channels,omitempty"`
	// Number of inactive channels
	NumInactiveChannels uint32 `protobuf:"varint,15,opt,name=num_inactive_channels,json=numInactiveChannels,proto3" json:"num_inactive_channels,omitempty"`
	// Number of peers
	NumPeers uint32 `protobuf:"varint,5,opt,name=num_peers,json=numPeers,proto3" json:"num_peers,omitempty"`
	// The node's current view of the height of the best block
	BlockHeight uint32 `protobuf:"varint,6,opt,name=block_height,json=blockHeight,proto3" json:"block_height,omitempty"`
	// The node's current view of the hash of the best block
	BlockHash string `protobuf:"bytes,8,opt,name=block_hash,json=blockHash,proto3" json:"block_hash,omitempty"`
	// Timestamp of the block best known to the wallet
	BestHeaderTimestamp int64 `protobuf:"varint,13,opt,name=best_header_timestamp,json=bestHeaderTimestamp,proto3" json:"best_header_timestamp,omitempty"`
	// Whether the wallet's view is synced to the main chain
	SyncedToChain bool `protobuf:"varint,9,opt,name=synced_to_chain,json=syncedToChain,proto3" json:"synced_to_chain,omitempty"`
	// Whether we consider ourselves synced with the public channel graph.
	SyncedToGraph bool `protobuf:"varint,18,opt,name=synced_to_graph,json=syncedToGraph,proto3" json:"synced_to_graph,omitempty"`
	//
	//Whether the current node is connected to testnet. This field is
	//deprecated and the network field should be used instead
	//
	// Deprecated: Do not use.
	Testnet bool `protobuf:"varint,10,opt,name=testnet,proto3" json:"testnet,omitempty"`
	// A list of active chains the node is connected to
	Chains []*Chain `protobuf:"bytes,16,rep,name=chains,proto3" json:"chains,omitempty"`
	// The URIs of the current node.
	Uris []string `protobuf:"bytes,12,rep,name=uris,proto3" json:"uris,omitempty"`
	//
	//Features that our node has advertised in our init message, node
	//announcements and invoices.
	Features map[uint32]*Feature `protobuf:"bytes,19,rep,name=features,proto3" json:"features,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}
type Chain struct {
	// The blockchain the node is on (eg bitcoin, litecoin)
	Chain string `protobuf:"bytes,1,opt,name=chain,proto3" json:"chain,omitempty"`
	// The network the node is on (eg regtest, testnet, mainnet)
	Network string `protobuf:"bytes,2,opt,name=network,proto3" json:"network,omitempty"`
}
type Feature struct {
	Name       string `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	IsRequired bool   `protobuf:"varint,3,opt,name=is_required,json=isRequired,proto3" json:"is_required,omitempty"`
	IsKnown    bool   `protobuf:"varint,4,opt,name=is_known,json=isKnown,proto3" json:"is_known,omitempty"`
}

// GetInfoController : GetInfoController struct
type GetInfoController struct {
	svc *service.LndhubService
}

func NewGetInfoController(svc *service.LndhubService) *GetInfoController {
	return &GetInfoController{svc: svc}
}

// GetInfo godoc
// @Summary      Get info about the Lightning node
// @Description  Returns info about the backend node powering this LNDhub instance
// @Accept       json
// @Produce      json
// @Tags         Info
// @Success      200  {object}  GetInfoResponse
// @Failure      400  {object}  responses.ErrorResponse
// @Failure      500  {object}  responses.ErrorResponse
// @Router       /getinfo [get]
// @Security     OAuth2Password
func (controller *GetInfoController) GetInfo(c echo.Context) error {

	info, err := controller.svc.GetInfo(c.Request().Context())
	if err != nil {
		return err
	}
	if controller.svc.Config.CustomName != "" {
		info.Alias = controller.svc.Config.CustomName
	}
	// BlueWallet right now requires a `identity_pubkey` in the response
	// https://github.com/BlueWallet/BlueWallet/blob/a28a2b96bce0bff6d1a24a951b59dc972369e490/class/wallets/lightning-custodian-wallet.js#L578
	return c.JSON(http.StatusOK, info)
}
