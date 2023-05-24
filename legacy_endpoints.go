package main

import (
	"net/http"

	"github.com/getAlby/lndhub.go/controllers"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
)

func RegisterLegacyEndpoints(svc *service.LndhubService, e *echo.Echo, secured *echo.Group, securedWithStrictRateLimit *echo.Group, strictRateLimitMiddleware echo.MiddlewareFunc, adminMw echo.MiddlewareFunc, logMw echo.MiddlewareFunc) {
	// Public endpoints for account creation and authentication
	e.POST("/auth", controllers.NewAuthController(svc).Auth, logMw)
	if svc.Config.AllowAccountCreation {
		e.POST("/create", controllers.NewCreateUserController(svc).CreateUser, strictRateLimitMiddleware, adminMw, logMw)
	}
	e.POST("/invoice/:user_login", controllers.NewInvoiceController(svc).Invoice, middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(rate.Limit(svc.Config.DefaultRateLimit))), logMw)

	// Secured endpoints which require a Authorization token (JWT)
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
