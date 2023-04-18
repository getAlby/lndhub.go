package main

import (
	"net/http"

	"github.com/getAlby/lndhub.go/controllers"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func RegisterLegacyEndpoints(svc *service.LndhubService, e *echo.Echo, tokenMW, strictRateLimitPerMinMW, strictRateLimitPerSecMW, rateLimitPerMinMW, rateLimitPerSecMW echo.MiddlewareFunc) {
	// Public endpoints for account creation and authentication
	e.POST("/auth", controllers.NewAuthController(svc).Auth)
	if svc.Config.AllowAccountCreation {
		e.POST("/create", controllers.NewCreateUserController(svc).CreateUser, strictRateLimitPerMinMW, strictRateLimitPerSecMW)
	}
	e.POST("/invoice/:user_login", controllers.NewInvoiceController(svc).Invoice, rateLimitPerMinMW, rateLimitPerSecMW)

	// Secured endpoints which require a Authorization token (JWT)
	securedWithStrictRateLimit := e.Group("", tokenMW, strictRateLimitPerMinMW, strictRateLimitPerSecMW)
	secured := e.Group("", tokenMW, rateLimitPerMinMW, rateLimitPerSecMW)
	secured.POST("/addinvoice", controllers.NewAddInvoiceController(svc).AddInvoice)
	securedWithStrictRateLimit.POST("/payinvoice", controllers.NewPayInvoiceController(svc).PayInvoice)
	secured.GET("/gettxs", controllers.NewGetTXSController(svc).GetTXS)
	secured.GET("/getuserinvoices", controllers.NewGetTXSController(svc).GetUserInvoices)
	secured.GET("/checkpayment/:payment_hash", controllers.NewCheckPaymentController(svc).CheckPayment)
	secured.GET("/balance", controllers.NewBalanceController(svc).Balance)
	secured.GET("/getinfo", controllers.NewGetInfoController(svc).GetInfo, createCacheClient().Middleware())
	securedWithStrictRateLimit.POST("/keysend", controllers.NewKeySendController(svc).KeySend)

	// These endpoints are currently not supported and we return a blank response for backwards compatibility
	blankController := controllers.NewBlankController(svc)
	secured.GET("/getbtc", blankController.GetBtc)
	secured.GET("/getpending", blankController.GetPending)

	//Index page endpoints, no Authorization required
	homeController := controllers.NewHomeController(svc, indexHtml)
	e.GET("/", homeController.Home, createCacheClient().Middleware())
	e.GET("/qr", homeController.QR)
	//workaround, just adding /static would make a request to these resources hit the authorized group
	e.GET("/static/css/*", echo.WrapHandler(http.FileServer(http.FS(staticContent))))
	e.GET("/static/img/*", echo.WrapHandler(http.FileServer(http.FS(staticContent))))
	e.Pre(middleware.Rewrite(map[string]string{
		"/favicon.ico": "/static/img/favicon.png",
	}))
}
