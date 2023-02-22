package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
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
	"google.golang.org/grpc"
)

//go:embed templates/index.html
var indexHtml string

//go:embed static/*
var staticContent embed.FS

// @title        LndHub.go
// @version      0.9.0
// @description  Accounting wrapper for the Lightning Network providing separate accounts for end-users.

// @contact.name   Alby
// @contact.url    https://getalby.com
// @contact.email  hello@getalby.com

// @license.name  GNU GPLv3
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
	dbConn, err := db.Open(c)
	if err != nil {
		logger.Fatalf("Error initializing db connection: %v", err)
	}

	// Migrate the DB
	//Todo: use timeout for startupcontext
	startupCtx := context.Background()
	migrator := migrate.NewMigrator(dbConn, migrations.Migrations)
	err = migrator.Init(startupCtx)
	if err != nil {
		logger.Fatalf("Error initializing db migrator: %v", err)
	}
	_, err = migrator.Migrate(startupCtx)
	if err != nil {
		logger.Fatalf("Error migrating database: %v", err)
	}

	// Setup exception tracking with Sentry if configured
	// sentry init needs to happen before the echo middlewares are added
	if c.SentryDSN != "" {
		if err = sentry.Init(sentry.ClientOptions{
			Dsn:              c.SentryDSN,
			IgnoreErrors:     []string{"401"},
			EnableTracing:    c.SentryTracesSampleRate > 0,
			TracesSampleRate: c.SentryTracesSampleRate,
		}); err != nil {
			logger.Errorf("sentry init error: %v", err)
		}
	}

	// New Echo app
	e := echo.New()
	e.HideBanner = true

	e.HTTPErrorHandler = responses.HTTPErrorHandler
	e.Validator = &lib.CustomValidator{Validator: validator.New()}

	e.Use(Middleware(WithServiceName("my-web-app")))
	e.Use(middleware.Recover())
	e.Use(middleware.BodyLimit("250K"))
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20)))

	e.Logger = logger
	e.Use(middleware.RequestID())
	e.Use(lecho.Middleware(lecho.Config{
		Logger: logger,
	}))

	// Setup exception tracking with Sentry if configured
	// sentry init needs to happen before the echo middlewares are added
	if c.SentryDSN != "" {
		e.Use(sentryecho.New(sentryecho.Options{}))
	}

	// Init new LND client
	lndClient, err := lnd.NewLNDclient(lnd.LNDoptions{
		Address:      c.LNDAddress,
		MacaroonFile: c.LNDMacaroonFile,
		MacaroonHex:  c.LNDMacaroonHex,
		CertFile:     c.LNDCertFile,
		CertHex:      c.LNDCertHex,
	})
	if err != nil {
		e.Logger.Fatalf("Error initializing the LND connection: %v", err)
	}
	getInfo, err := lndClient.GetInfo(startupCtx, &lnrpc.GetInfoRequest{})
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

	RegisterLegacyEndpoints(svc, e, secured, securedWithStrictRateLimit, strictRateLimitMiddleware, tokens.AdminTokenMiddleware(c.AdminToken))
	RegisterV2Endpoints(svc, e, secured, securedWithStrictRateLimit, strictRateLimitMiddleware, tokens.AdminTokenMiddleware(c.AdminToken))

	//invoice streaming
	//Authentication should be done through the query param because this is a websocket
	e.GET("/invoices/stream", controllers.NewInvoiceStreamController(svc).StreamInvoices)

	//Swagger API spec
	docs.SwaggerInfo.Host = c.Host
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	var backgroundWg sync.WaitGroup
	backGroundCtx, _ := signal.NotifyContext(context.Background(), os.Interrupt)
	// Subscribe to LND invoice updates in the background
	backgroundWg.Add(1)
	go func() {
		err = svc.InvoiceUpdateSubscription(backGroundCtx)
		if err != nil {
			svc.Logger.Error(err)
		}
		svc.Logger.Info("Invoice routine done")
		backgroundWg.Done()
	}()

	// Check the status of all pending outgoing payments
	// A goroutine will be spawned for each one
	backgroundWg.Add(1)
	go func() {
		err = svc.CheckAllPendingOutgoingPayments(backGroundCtx)
		if err != nil {
			svc.Logger.Error(err)
		}
		svc.Logger.Info("Pending payment check routines done")
		backgroundWg.Done()
	}()

	//Start webhook subscription
	if svc.Config.WebhookUrl != "" {
		backgroundWg.Add(1)
		go func() {
			svc.StartWebhookSubscription(backGroundCtx, svc.Config.WebhookUrl)
			svc.Logger.Info("Webhook routine done")
			backgroundWg.Done()
		}()
	}
	//Start rabbit publisher
	if svc.Config.RabbitMQUri != "" {
		backgroundWg.Add(1)
		go func() {
			err = svc.StartRabbitMqPublisher(backGroundCtx)
			if err != nil {
				svc.Logger.Error(err)
				sentry.CaptureException(err)
			}
			svc.Logger.Info("Rabbit routine done")
			backgroundWg.Done()
		}()
	}

	var grpcServer *grpc.Server
	if svc.Config.EnableGRPC {
		//start grpc server
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", svc.Config.GRPCPort))
		if err != nil {
			svc.Logger.Fatalf("Failed to start grpc server: %v", err)
		}
		grpcServer = svc.NewGrpcServer(startupCtx)
		go func() {
			svc.Logger.Infof("Starting grpc server at port %d", svc.Config.GRPCPort)
			err = grpcServer.Serve(lis)
			if err != nil {
				svc.Logger.Error(err)
			}
		}()
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

	<-backGroundCtx.Done()
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
	if c.EnableGRPC {
		grpcServer.Stop()
		svc.Logger.Info("GRPC server exited.")
	}
	//Wait for graceful shutdown of background routines
	backgroundWg.Wait()
	svc.Logger.Info("LNDhub exiting gracefully. Goodbye.")
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
