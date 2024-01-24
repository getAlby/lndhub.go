package common
// bitcoin - init assets table row
const BTC_INTERNAL_ASSET_ID = 1
const BTC_TA_ASSET_ID       = "native-asset-bitcoin"
const BTC_ASSET_NAME        = "bitcoin"

// https://lightning.engineering/api-docs/api/taproot-assets/universe/query-asset-stats#taprpcassettype
type AssetType int64
const (
	Normal       AssetType = 0
	Collectible  AssetType = 1
)

const (
	InvoiceTypeOutgoing = "outgoing"
	InvoiceTypePaid     = "paid_invoice"
	InvoiceTypeIncoming = "incoming"
	InvoiceTypeUser     = "user_invoice"

	InvoiceStateSettled     = "settled"
	InvoiceStateInitialized = "initialized"
	InvoiceStateOpen        = "open"
	InvoiceStateError       = "error"

	AccountTypeIncoming = "incoming"
	AccountTypeCurrent  = "current"
	AccountTypeOutgoing = "outgoing"
	AccountTypeFees     = "fees"

	DestinationPubkeyHexSize = 66
)
