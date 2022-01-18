package lnd

import (
	"encoding/hex"

	"github.com/getAlby/lndhub.go/lib"
	"github.com/labstack/echo/v4"
	"github.com/lightningnetwork/lnd/lnrpc"
)

type LNDInterfaceImpl struct {
	LndHubContext *lib.LndhubContext
}

func NewLNDInterface(lndContext *lib.LndhubContext) *LNDInterfaceImpl {
	return &LNDInterfaceImpl{
		LndHubContext: lndContext,
	}
}

type LNDInterface interface {
	AddInvoice(value int64, memo string) (Invoice, error)
}

// AddInvoice generates an invoice with the given price and memo.
func (c LNDInterfaceImpl) AddInvoice(ctx echo.Context, value int64, memo string, rPreimage []byte, expiry int64) (Invoice, error) {
	lndClient := *c.LndHubContext.LndClient
	result := Invoice{}

	c.LndHubContext.Context.Logger().Printf("Adding invoice: memo=%s value=%v", memo, value)
	invoice := lnrpc.Invoice{
		Memo:      memo,
		Value:     value,
		RPreimage: rPreimage,
		Expiry:    expiry,
	}

	res, err := lndClient.AddInvoice(ctx.Request().Context(), &invoice)
	if err != nil {
		return result, err
	}

	result.PaymentHash = hex.EncodeToString(res.RHash)
	result.PaymentRequest = res.PaymentRequest
	result.Settled = true

	return result, nil
}
