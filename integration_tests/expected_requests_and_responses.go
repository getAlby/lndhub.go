package integration_tests

import (
	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lib/service"
)

type ExpectedKeySendRequestBody struct {
	Amount        int64             `json:"amount" validate:"required"`
	Destination   string            `json:"destination" validate:"required"`
	Memo          string            `json:"memo" validate:"omitempty"`
	CustomRecords map[string]string `json:"customRecords" validate:"omitempty"`
}

type ExpectedKeySendResponseBody struct {
	RHash              *lib.JavaScriptBuffer `json:"payment_hash,omitempty"`
	Amount             int64                 `json:"num_satoshis,omitempty"`
	Description        string                `json:"description,omitempty"`
	Destination        string                `json:"destination,omitempty"`
	DescriptionHashStr string                `json:"description_hash,omitempty"`
	PaymentError       string                `json:"payment_error,omitempty"`
	PaymentPreimage    *lib.JavaScriptBuffer `json:"payment_preimage,omitempty"`
	PaymentRoute       *service.Route        `json:"payment_route,omitempty"`
}

type ExpectedAddInvoiceRequestBody struct {
	Amount          interface{} `json:"amt"` // amount in Satoshi
	Memo            string      `json:"memo"`
	DescriptionHash string      `json:"description_hash" validate:"omitempty,hexadecimal,len=64"`
}

type ExpectedAddInvoiceResponseBody struct {
	RHash          string `json:"r_hash"`
	PaymentRequest string `json:"payment_request"`
	PayReq         string `json:"pay_req"`
}

type ExpectedAuthRequestBody struct {
	Login        string `json:"login"`
	Password     string `json:"password"`
	RefreshToken string `json:"refresh_token"`
}

type ExpectedAuthResponseBody struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
}

type ExpectedBalanceResponse struct {
	BTC struct {
		AvailableBalance int64
	}
}

type ExpectedCheckPaymentResponseBody struct {
	IsPaid bool `json:"paid"`
}

type ExpectedCreateUserResponseBody struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}
type ExpectedCreateUserRequestBody struct {
	Login       string `json:"login"`
	Password    string `json:"password"`
	PartnerID   string `json:"partnerid"`
	AccountType string `json:"accounttype"`
}

type ExpectedOutgoingInvoice struct {
	RHash           interface{} `json:"r_hash"`
	PaymentHash     interface{} `json:"payment_hash"`
	PaymentPreimage string      `json:"payment_preimage"`
	Value           int64       `json:"value"`
	Type            string      `json:"type"`
	Fee             int64       `json:"fee"`
	Timestamp       int64       `json:"timestamp"`
	Memo            string      `json:"memo"`
}

type ExpectedIncomingInvoice struct {
	RHash          interface{} `json:"r_hash"`
	PaymentHash    interface{} `json:"payment_hash"`
	PaymentRequest string      `json:"payment_request"`
	Description    string      `json:"description"`
	PayReq         string      `json:"pay_req"`
	Timestamp      int64       `json:"timestamp"`
	Type           string      `json:"type"`
	ExpireTime     int64       `json:"expire_time"`
	Amount         int64       `json:"amt"`
	IsPaid         bool        `json:"ispaid"`
}
type ExpectedInvoiceEventWrapper struct {
	Type    string                   `json:"type"`
	Invoice *ExpectedIncomingInvoice `json:"invoice,omitempty"`
}

type ExpectedPayInvoiceRequestBody struct {
	Invoice string      `json:"invoice" validate:"required"`
	Amount  interface{} `json:"amount" validate:"omitempty"`
}

type ExpectedPayInvoiceResponseBody struct {
	RHash              *lib.JavaScriptBuffer `json:"payment_hash,omitempty"`
	PaymentRequest     string                `json:"payment_request,omitempty"`
	PayReq             string                `json:"pay_req,omitempty"`
	Amount             int64                 `json:"num_satoshis,omitempty"`
	Description        string                `json:"description,omitempty"`
	DescriptionHashStr string                `json:"description_hash,omitempty"`
	PaymentError       string                `json:"payment_error,omitempty"`
	PaymentPreimage    *lib.JavaScriptBuffer `json:"payment_preimage,omitempty"`
	PaymentRoute       *service.Route        `json:"payment_route,omitempty"`
}
