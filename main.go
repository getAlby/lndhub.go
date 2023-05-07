package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/getAlby/lndhub.go/rabbitmq"

	cache "github.com/SporkHubr/echo-http-cache"
	"github.com/SporkHubr/echo-http-cache/adapter/memory"
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
	"github.com/rs/zerolog"
	echoSwagger "github.com/swaggo/echo-swagger"
	"github.com/uptrace/bun/migrate"
	"github.com/ziflex/lecho/v3"
	"golang.org/x/time/rate"
	ddEcho "gopkg.in/DataDog/dd-trace-go.v1/contrib/labstack/echo.v4"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
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

	//if Datadog is configured, add datadog middleware
	if c.DatadogAgentUrl != "" {
		tracer.Start(tracer.WithAgentAddr(c.DatadogAgentUrl))
		defer tracer.Stop()
		e.Use(ddEcho.Middleware(ddEcho.WithServiceName("lndhub.go")))
	}
	e.Use(middleware.Recover())
	e.Use(middleware.BodyLimit("250K"))
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20)))

	e.Logger = logger
	e.Use(middleware.RequestID())
	e.Use(lecho.Middleware(lecho.Config{
		Logger: logger,
		Enricher: func(c echo.Context, logger zerolog.Context) zerolog.Context {
			userId := c.Get("UserID")
			if userId != nil {
				return logger.Str("user_id", userId.(string))
			}
			return logger.Str("user_id", "")
		},
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

	// If no RABBITMQ_URI was provided we will not attempt to create a client
	// No rabbitmq features will be available in this case.
	var rabbitmqClient rabbitmq.Client
	if c.RabbitMQUri != "" {
		rabbitmqClient, err = rabbitmq.Dial(c.RabbitMQUri,
			rabbitmq.WithLogger(logger),
			rabbitmq.WithLndInvoiceExchange(c.RabbitMQLndInvoiceExchange),
			rabbitmq.WithLndHubInvoiceExchange(c.RabbitMQLndhubInvoiceExchange),
			rabbitmq.WithLndInvoiceConsumerQueueName(c.RabbitMQInvoiceConsumerQueueName),
		)
		if err != nil {
			logger.Fatal(err)
		}

		// close the connection gently at the end of the runtime
		defer rabbitmqClient.Close()
	}

	svc := &service.LndhubService{
		Config:         c,
		DB:             dbConn,
		LndClient:      lndClient,
		RabbitMQClient: rabbitmqClient,
		Logger:         logger,
		IdentityPubkey: getInfo.IdentityPubkey,
		InvoicePubSub:  service.NewPubsub(),
	}

	strictRateLimitMiddleware := createRateLimitMiddleware(c.StrictRateLimit, c.BurstRateLimit)
	secured := e.Group("", tokens.Middleware(c.JWTSecret), middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(rate.Limit(c.DefaultRateLimit))))
	securedWithStrictRateLimit := e.Group("", tokens.Middleware(c.JWTSecret), strictRateLimitMiddleware)

	RegisterLegacyEndpoints(svc, e, secured, securedWithStrictRateLimit, strictRateLimitMiddleware, tokens.AdminTokenMiddleware(c.AdminToken))
	RegisterV2Endpoints(svc, e, secured, securedWithStrictRateLimit, strictRateLimitMiddleware, tokens.AdminTokenMiddleware(c.AdminToken))

	//Swagger API spec
	docs.SwaggerInfo.Host = c.Host
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	var backgroundWg sync.WaitGroup
	backGroundCtx, _ := signal.NotifyContext(context.Background(), os.Interrupt)
	// Subscribe to LND invoice updates in the background
	backgroundWg.Add(1)
	go func() {
		switch svc.Config.SubscriptionConsumerType {
		case "rabbitmq":
			err = svc.RabbitMQClient.SubscribeToLndInvoices(backGroundCtx, svc.ProcessInvoiceUpdate)
			if err != nil && err != context.Canceled {
				// in case of an error in this routine, we want to restart LNDhub
				sentry.CaptureException(err)
				svc.Logger.Fatal(err)
			}

		case "grpc":
			err = svc.InvoiceUpdateSubscription(backGroundCtx)
			if err != nil && err != context.Canceled {
				// in case of an error in this routine, we want to restart LNDhub
				svc.Logger.Fatal(err)
			}

		default:
			svc.Logger.Fatalf("Unrecognized subscription consumer type %s", svc.Config.SubscriptionConsumerType)
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
	if svc.RabbitMQClient != nil {
		backgroundWg.Add(1)
		go func() {
			err = svc.RabbitMQClient.StartPublishInvoices(backGroundCtx,
				svc.SubscribeIncomingOutgoingInvoices,
				svc.EncodeInvoiceWithUserLogin,
			)
			if err != nil {
				svc.Logger.Error(err)
				sentry.CaptureException(err)
			}

			svc.Logger.Info("Rabbit invoice publisher done")
			backgroundWg.Done()
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
