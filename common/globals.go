package common

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
