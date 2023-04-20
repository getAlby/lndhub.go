package main

import (
	v2controllers "github.com/getAlby/lndhub.go/controllers_v2"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
)

func RegisterV2Endpoints(svc *service.LndhubService, e *echo.Echo, tokenMW, strictRateLimitPerMinMW, strictRateLimitPerSecMW, rateLimitPerMinMW, rateLimitPerSecMW, signatureMW echo.MiddlewareFunc) {
	// TODO: v2 auth endpoint: generalized oauth token generation
	// e.POST("/auth", controllers.NewAuthController(svc).Auth)

	securedWithStrictRateLimit := e.Group("", tokenMW, strictRateLimitPerMinMW, strictRateLimitPerSecMW)
	secured := e.Group("", tokenMW, rateLimitPerMinMW, rateLimitPerSecMW)
	if svc.Config.AllowAccountCreation {
		e.POST("/v2/create", v2controllers.NewCreateUserController(svc).CreateUser, strictRateLimitPerMinMW, strictRateLimitPerSecMW, signatureMW)
	}
	e.GET("/v2/lnurlp/:user", v2controllers.NewLnurlController(svc).Lnurlp, strictRateLimitPerMinMW, strictRateLimitPerSecMW)
	invoiceCtrl := v2controllers.NewInvoiceController(svc)
	keysendCtrl := v2controllers.NewKeySendController(svc)
	e.GET("/v2/invoice", invoiceCtrl.Lud6Invoice, rateLimitPerMinMW, rateLimitPerSecMW)
	secured.POST("/v2/invoices", invoiceCtrl.AddInvoice)
	secured.GET("/v2/invoices/incoming", invoiceCtrl.GetIncomingInvoices)
	secured.GET("/v2/invoices/outgoing", invoiceCtrl.GetOutgoingInvoices)
	secured.GET("/v2/invoices/:payment_hash", invoiceCtrl.GetInvoice)
	securedWithStrictRateLimit.POST("/v2/payments/bolt11", v2controllers.NewPayInvoiceController(svc).PayInvoice)
	securedWithStrictRateLimit.POST("/v2/payments/keysend", keysendCtrl.KeySend)
	securedWithStrictRateLimit.POST("/v2/payments/keysend/multi", keysendCtrl.MultiKeySend)
	secured.GET("/v2/balance", v2controllers.NewBalanceController(svc).Balance)
}
