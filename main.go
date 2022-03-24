package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	cache "github.com/SporkHubr/echo-http-cache"
	"github.com/SporkHubr/echo-http-cache/adapter/memory"
	"github.com/getAlby/lndhub.go/controllers"
	"github.com/getAlby/lndhub.go/db"
	"github.com/getAlby/lndhub.go/db/migrations"
	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/getAlby/lndhub.go/lib/tokens"
	"github.com/getAlby/lndhub.go/lnd"
	"github.com/getsentry/sentry-go"
	sentryecho "github.com/getsentry/sentry-go/echo"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/uptrace/bun/migrate"
	"github.com/ziflex/lecho/v3"
	"golang.org/x/time/rate"
)

//go:embed templates/index.html
var indexHtml string

//go:embed static/*
var staticContent embed.FS

func main() {
	c := &service.Config{}

	// Load configruation from environment variables
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Failed to load .env file")
	}
	err = envconfig.Process("", c)
	if err != nil {
		log.Fatalf("Error loading environment variables: %v", err)
	}

	// Setup logging to STDOUT or a configrued log file
	logger := lib.Logger(c.LogFilePath)

	// Open a DB connection based on the configured DATABASE_URI
	dbConn, err := db.Open(c.DatabaseUri)
	if err != nil {
		logger.Fatalf("Error initializing db connection: %v", err)
	}

	// Migrate the DB
	ctx := context.Background()
	migrator := migrate.NewMigrator(dbConn, migrations.Migrations)
	err = migrator.Init(ctx)
	if err != nil {
		logger.Fatalf("Error initializing db migrator: %v", err)
	}
	_, err = migrator.Migrate(ctx)
	if err != nil {
		logger.Fatalf("Error migrating database: %v", err)
	}

	// New Echo app
	e := echo.New()
	e.HideBanner = true

	e.HTTPErrorHandler = responses.HTTPErrorHandler
	e.Validator = &lib.CustomValidator{Validator: validator.New()}

	e.Use(middleware.Recover())
	e.Use(middleware.BodyLimit("250K"))
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20)))

	e.Logger = logger
	e.Use(middleware.RequestID())
	e.Use(lecho.Middleware(lecho.Config{
		Logger: logger,
	}))

	// Setup exception tracking with Sentry if configured
	if c.SentryDSN != "" {
		if err = sentry.Init(sentry.ClientOptions{
			Dsn: c.SentryDSN,
		}); err != nil {
			logger.Errorf("sentry init error: %v", err)
		}
		defer sentry.Flush(2 * time.Second)
		e.Use(sentryecho.New(sentryecho.Options{}))
	}

	// Init new LND client
	//lndClient, err := lnd.NewLNDclient(lnd.LNDoptions{
	//	Address:     c.LNDAddress,
	//	MacaroonHex: c.LNDMacaroonHex,
	//	CertHex:     c.LNDCertHex,
	//})

	//Init new CLN client
	//re-use other config to not make things overcomplicated
	lndClient, err := lnd.NewCLNClient(lnd.CLNClientOptions{
		SparkUrl:   c.LNDAddress,
		SparkToken: c.LNDMacaroonHex,
	})
	if err != nil {
		e.Logger.Fatalf("Error initializing the LND connection: %v", err)
	}
	getInfo, err := lndClient.GetInfo(ctx, &lnrpc.GetInfoRequest{})
	if err != nil {
		e.Logger.Fatalf("Error getting node info: %v", err)
	}
	logger.Infof("Connected to LND: %s - %s", getInfo.Alias, getInfo.IdentityPubkey)

	svc := &service.LndhubService{
		Config:             c,
		DB:                 dbConn,
		LndClient:          lndClient,
		Logger:             logger,
		IdentityPubkey:     getInfo.IdentityPubkey,
		InvoiceSubscribers: map[int64]chan models.Invoice{},
	}

	strictRateLimitMiddleware := createRateLimitMiddleware(c.StrictRateLimit, c.BurstRateLimit)
	// Public endpoints for account creation and authentication
	e.POST("/auth", controllers.NewAuthController(svc).Auth)
	e.POST("/create", controllers.NewCreateUserController(svc).CreateUser, strictRateLimitMiddleware)
	e.POST("/invoice/:user_login", controllers.NewInvoiceController(svc).Invoice, middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(rate.Limit(c.DefaultRateLimit))))

	// Secured endpoints which require a Authorization token (JWT)
	secured := e.Group("", tokens.Middleware(c.JWTSecret), middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(rate.Limit(c.DefaultRateLimit))))
	securedWithStrictRateLimit := e.Group("", tokens.Middleware(c.JWTSecret), strictRateLimitMiddleware)
	secured.POST("/addinvoice", controllers.NewAddInvoiceController(svc).AddInvoice)
	securedWithStrictRateLimit.POST("/payinvoice", controllers.NewPayInvoiceController(svc).PayInvoice)
	secured.GET("/gettxs", controllers.NewGetTXSController(svc).GetTXS)
	secured.GET("/getuserinvoices", controllers.NewGetTXSController(svc).GetUserInvoices)
	secured.GET("/checkpayment/:payment_hash", controllers.NewCheckPaymentController(svc).CheckPayment)
	secured.GET("/balance", controllers.NewBalanceController(svc).Balance)
	secured.GET("/getinfo", controllers.NewGetInfoController(svc).GetInfo, createCacheClient().Middleware())
	securedWithStrictRateLimit.POST("/keysend", controllers.NewKeySendController(svc).KeySend)
	secured.GET("/getinfo", controllers.NewGetInfoController(svc).GetInfo)
	secured.POST("/bolt12/fetchinvoice", controllers.NewBolt12Controller(svc).FetchInvoice)
	secured.POST("/bolt12/pay", controllers.NewBolt12Controller(svc).PayBolt12)

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

	e.GET("/bolt12/decode/:offer", controllers.NewBolt12Controller(svc).Decode)
	//invoice streaming
	//Authentication should be done through the query param because this is a websocket
	e.GET("/invoices/stream", controllers.NewInvoiceStreamController(svc).StreamInvoices)

	// Subscribe to LND invoice updates in the background
	// CLN: todo: re-write logic
	go svc.InvoiceUpdateSubscription(context.Background())

	// Start server
	go func() {
		if err := e.Start(fmt.Sprintf(":%v", c.Port)); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal("shutting down the server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 10 seconds.
	// Use a buffered channel to avoid missing signals as recommended for signal.Notify
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
	//close all channels
	for _, sub := range svc.InvoiceSubscribers {
		close(sub)
	}
}

func createRateLimitMiddleware(seconds int, burst int) echo.MiddlewareFunc {
	config := middleware.RateLimiterMemoryStoreConfig{
		Rate:  rate.Every(time.Duration(seconds) * time.Second),
		Burst: burst,
	}
	return middleware.RateLimiter(middleware.NewRateLimiterMemoryStoreWithConfig(config))
}

func createCacheClient() *cache.Client {
	memcached, err := memory.NewAdapter(
		memory.AdapterWithAlgorithm(memory.LRU),
		memory.AdapterWithCapacity(10000000),
	)

	if err != nil {
		log.Fatalf("Error creating cache client memory adapter: %v", err)
	}

	cacheClient, err := cache.NewClient(
		cache.ClientWithAdapter(memcached),
		cache.ClientWithTTL(10*time.Minute),
		cache.ClientWithRefreshKey("opn"),
	)

	if err != nil {
		log.Fatalf("Error creating cache client: %v", err)
	}
	return cacheClient
}
