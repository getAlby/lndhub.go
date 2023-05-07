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
	echoSwagger "github.com/swaggo/echo-swagger"
	"github.com/uptrace/bun/migrate"
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
		logger.Fatal().Err(err).Msgf("Error initializing db connection: %v", err)
	}

	// Migrate the DB
	//Todo: use timeout for startupcontext
	startupCtx := context.Background()
	migrator := migrate.NewMigrator(dbConn, migrations.Migrations)
	err = migrator.Init(startupCtx)
	if err != nil {
		logger.Fatal().Err(err).Msgf("Error initializing db migrator: %v", err)
	}
	_, err = migrator.Migrate(startupCtx)
	if err != nil {
		logger.Fatal().Err(err).Msgf("Error migrating database: %v", err)
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
			logger.Error().Err(err).Msgf("sentry init error: %v", err)
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

	//e.Logger = logger
	e.Use(middleware.RequestID())
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:    true,
		LogStatus: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			userId := c.Get("UserID").(int64)
			logger.Info().
				Str("URI", v.URI).
				Int("status", v.Status).
				Str("request_id", v.RequestID).
				Int("status", v.Status).
				Str("path", v.URI).
				Int("duration", int(v.Latency)).
				Str("referrer", v.Referer).
				Str("user_agent", v.UserAgent).
				Int64("user_id", userId).
				Msgf("%s %s", v.Method, v.URI)

			return nil
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
		logger.Fatal().Err(err).Msgf("Error getting node info: %v", err)
	}
	logger.Info().Msgf("Connected to LND: %s - %s", getInfo.Alias, getInfo.IdentityPubkey)

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
			logger.Fatal().Err(err).Msgf("Failed to init RabbitMQ client")
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
				svc.Logger.Fatal().Err(err)
			}

		case "grpc":
			err = svc.InvoiceUpdateSubscription(backGroundCtx)
			if err != nil && err != context.Canceled {
				// in case of an error in this routine, we want to restart LNDhub
				svc.Logger.Fatal().Err(err)
			}

		default:
			svc.Logger.Fatal().Err(err).Msgf("Unrecognized subscription consumer type %s", svc.Config.SubscriptionConsumerType)
		}

		svc.Logger.Info().Msg("Invoice routine done")
		backgroundWg.Done()
	}()

	// Check the status of all pending outgoing payments
	// A goroutine will be spawned for each one
	backgroundWg.Add(1)
	go func() {
		err = svc.CheckAllPendingOutgoingPayments(backGroundCtx)
		if err != nil {
			svc.Logger.Error().Err(err)
		}
		svc.Logger.Info().Msg("Pending payment check routines done")
		backgroundWg.Done()
	}()

	//Start webhook subscription
	if svc.Config.WebhookUrl != "" {
		backgroundWg.Add(1)
		go func() {
			svc.StartWebhookSubscription(backGroundCtx, svc.Config.WebhookUrl)
			svc.Logger.Info().Msg("Webhook routine done")
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
				svc.Logger.Error().Err(err)
				sentry.CaptureException(err)
			}

			svc.Logger.Info().Msg("Rabbit invoice publisher done")
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
			//echoPrometheus.Logger = logger
			//echoPrometheus.Logger.Infof("Starting prometheus on port %d", svc.Config.PrometheusPort)
			//echoPrometheus.Logger.Fatal(echoPrometheus.Start(fmt.Sprintf(":%d", svc.Config.PrometheusPort)))
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
	svc.Logger.Info().Msg("LNDhub exiting gracefully. Goodbye.")
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
