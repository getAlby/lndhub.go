package transport

import (
	v2controllers "github.com/getAlby/lndhub.go/controllers_v2"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
)

func RegisterV2Endpoints(svc *service.LndhubService, e *echo.Echo, secured *echo.Group, validateNostrPayload  *echo.Group, securedWithStrictRateLimit *echo.Group, strictRateLimitMiddleware echo.MiddlewareFunc, adminMw echo.MiddlewareFunc, logMw echo.MiddlewareFunc) {
	// TODO: v2 auth endpoint: generalized oauth token generation
	// e.POST("/auth", controllers.NewAuthController(svc).Auth)
	if svc.Config.AllowAccountCreation {
		/// TAHUB_CREATE_USER / N.S. register modified endpoint
		e.POST("/v2/users", v2controllers.NewCreateUserController(svc).CreateUser, strictRateLimitMiddleware, adminMw, logMw)
	}
	//require admin token for update user endpoint
	if svc.Config.AdminToken != "" {
		e.PUT("/v2/admin/users", v2controllers.NewUpdateUserController(svc).UpdateUser, strictRateLimitMiddleware, adminMw)
	}
	invoiceCtrl := v2controllers.NewInvoiceController(svc)
	keysendCtrl := v2controllers.NewKeySendController(svc)
	nostrEventCtrl := v2controllers.NewNostrController(svc)

	// add the endpoint to the group 
	// NOSTR EVENT Request
	validateNostrPayload.POST("/v2/event", nostrEventCtrl.AddNostrEvent)

	secured.POST("/v2/invoices", invoiceCtrl.AddInvoice)
	secured.GET("/v2/invoices/incoming", invoiceCtrl.GetIncomingInvoices)
	secured.GET("/v2/invoices/outgoing", invoiceCtrl.GetOutgoingInvoices)
	secured.GET("/v2/invoices/:payment_hash", invoiceCtrl.GetInvoice)
	securedWithStrictRateLimit.POST("/v2/payments/bolt11", v2controllers.NewPayInvoiceController(svc).PayInvoice)
	securedWithStrictRateLimit.POST("/v2/payments/keysend", keysendCtrl.KeySend)
	securedWithStrictRateLimit.POST("/v2/payments/keysend/multi", keysendCtrl.MultiKeySend)
	secured.GET("/v2/balance", v2controllers.NewBalanceController(svc).Balance)
}
