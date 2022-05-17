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

	UserIdCustomRecordType = 696969 //cfr. https://github.com/satoshisstream/satoshis.stream/blob/main/TLV_registry.md#field-696969---lnpay
)
