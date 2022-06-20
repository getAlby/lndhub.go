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
	"github.com/getAlby/lndhub.go/docs"
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
	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/lightningnetwork/lnd/lnrpc"
	echoSwagger "github.com/swaggo/echo-swagger"
	"github.com/uptrace/bun/migrate"
	"github.com/ziflex/lecho/v3"
	"golang.org/x/time/rate"
)

//go:embed templates/index.html
var indexHtml string

//go:embed static/*
var staticContent embed.FS

// @title        LNDhub.go
// @version      0.6.1
// @description  Accounting wrapper for the Lightning Network providing separate accounts for end-users.

// @contact.name   Alby
// @contact.url    https://getalby.com
// @contact.email  hello@getalby.com

// @license.name  GNU GPL
// @license.url   https://www.gnu.org/licenses/gpl-3.0.en.html

// @BasePath  /

// @securitydefinitions.oauth2.password  OAuth2Password
// @tokenUrl                             /auth
// @schemes                              https http
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
			Dsn:          c.SentryDSN,
			IgnoreErrors: []string{"401"},
		}); err != nil {
			logger.Errorf("sentry init error: %v", err)
		}
		defer sentry.Flush(2 * time.Second)
		e.Use(sentryecho.New(sentryecho.Options{}))
	}

	// Init new LND client
	lndClient, err := lnd.NewLNDclient(lnd.LNDoptions{
		Address:     c.LNDAddress,
		MacaroonHex: c.LNDMacaroonHex,
		CertHex:     c.LNDCertHex,
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
		Config:         c,
		DB:             dbConn,
		LndClient:      lndClient,
		Logger:         logger,
		IdentityPubkey: getInfo.IdentityPubkey,
		InvoicePubSub:  service.NewPubsub(),
	}

	strictRateLimitMiddleware := createRateLimitMiddleware(c.StrictRateLimit, c.BurstRateLimit)
	secured := e.Group("", tokens.Middleware(c.JWTSecret), middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(rate.Limit(c.DefaultRateLimit))))
	securedWithStrictRateLimit := e.Group("", tokens.Middleware(c.JWTSecret), strictRateLimitMiddleware)

	RegisterLegacyEndpoints(svc, e, secured, securedWithStrictRateLimit, strictRateLimitMiddleware)
	RegisterV2Endpoints(svc, e, secured, securedWithStrictRateLimit, strictRateLimitMiddleware)

	//invoice streaming
	//Authentication should be done through the query param because this is a websocket
	e.GET("/invoices/stream", controllers.NewInvoiceStreamController(svc).StreamInvoices)

	//Swagger API spec
	docs.SwaggerInfo.Host = c.Host
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	// Subscribe to LND invoice updates in the background
	go svc.InvoiceUpdateSubscription(context.Background())

	//Start webhook subscription
	if svc.Config.WebhookUrl != "" {
		webhookCtx, cancelWebhook := context.WithCancel(context.Background())
		go svc.StartWebhookSubscribtion(webhookCtx, svc.Config.WebhookUrl)
		defer cancelWebhook()
	}

	//Start Prometheus server if necessary
	var echoPrometheus *echo.Echo
	if svc.Config.EnablePrometheus {
		// Create Prometheus server and Middleware
		echoPrometheus = echo.New()
		echoPrometheus.HideBanner = true
		prom := prometheus.NewPrometheus("echo", nil)
		// Scrape metrics from Main Server
		e.Use(prom.HandlerFunc)
		// Setup metrics endpoint at another server
		prom.SetMetricsPath(echoPrometheus)
		go func() {
			echoPrometheus.Logger = logger
			echoPrometheus.Logger.Infof("Starting prometheus on port %d", svc.Config.PrometheusPort)
			echoPrometheus.Logger.Fatal(echoPrometheus.Start(fmt.Sprintf(":%d", svc.Config.PrometheusPort)))
		}()
	}

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
	if echoPrometheus != nil {
		if err := echoPrometheus.Shutdown(ctx); err != nil {
			e.Logger.Fatal(err)
		}
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
