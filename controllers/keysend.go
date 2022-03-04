package controllers

import (
	"fmt"

	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
)

// KeySendController : Pay invoice controller struct
type KeySendController struct {
	svc *service.LndhubService
}

func NewKeySendController(svc *service.LndhubService) *KeySendController {
	return &KeySendController{svc: svc}
}

type KeySendRequestBody struct {
	Amount      int64  `json:"amount" validate:"required"`
	Destination string `json:"destination" validate:"required"`
	Memo        string `json:"memo" validate:"omitempty"`
}

type KeySendResponseBody struct {
	RHash              *lib.JavaScriptBuffer `json:"payment_hash,omitempty"`
	Amount             int64                 `json:"num_satoshis,omitempty"`
	Description        string                `json:"description,omitempty"`
	Destination        string                `json:"destination,omitempty"`
	DescriptionHashStr string                `json:"description_hash,omitempty"`
	PaymentError       string                `json:"payment_error,omitempty"`
	PaymentPreimage    *lib.JavaScriptBuffer `json:"payment_preimage,omitempty"`
	PaymentRoute       *service.Route        `json:"route,omitempty"`
}

// KeySend : Pay invoice Controller
func (controller *KeySendController) KeySend(c echo.Context) error {
	/*
		TODO: copy code from payinvoice.ctrl.go and modify where needed:
		- do not decode the payment request because there is no payment request.
		  Instead, construct the lnrpc.PaymentRequest manually from the KeySendRequestBody.
		- add outgoing invoice: same as payinvoice, make sure to set keysend=true
		- do a balance check: same as payinvoice, in fact do this before doing anything else
		- call svc.PayInvoice : same as payinvoice as long as keysend=true in Invoice
		- response will be slightly different due to lack of payment request
	*/
	return fmt.Errorf("TODO")
}
