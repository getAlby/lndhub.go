package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/getAlby/lndhub.go/lnd"
	"github.com/getAlby/lndhub.go/rabbitmq"
	"github.com/getAlby/lndhub.go/tapd"
	ddEcho "gopkg.in/DataDog/dd-trace-go.v1/contrib/labstack/echo.v4"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"github.com/getAlby/lndhub.go/db"
	"github.com/getAlby/lndhub.go/db/migrations"
	"github.com/getAlby/lndhub.go/docs"
	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/getAlby/lndhub.go/lib/tokens"
	"github.com/getAlby/lndhub.go/lib/transport"
	"github.com/getsentry/sentry-go"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"
	"github.com/uptrace/bun/migrate"
)

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
	// Init new LND client
	lnCfg, err := lnd.LoadConfig()
	if err != nil {
		logger.Fatalf("Error loading LN config: %v", err)
	}
	lndClient, err := lnd.InitLNClient(lnCfg, logger, startupCtx)
	if err != nil {
		logger.Fatalf("Error initializing the %s connection: %v", lnCfg.LNClientType, err)
	}

	logger.Infof("Connected to %s: %s", lnCfg.LNClientType, lndClient.GetMainPubkey())
	// Init new TAPD client
	tapdConfig, err := tapd.LoadConfig()
	if err != nil {
		logger.Fatalf("Error loading LN config: %v", err)
	}
	tapdClient, err := tapd.InitTAPDClient(tapdConfig, logger, startupCtx)
	if err != nil {
		logger.Fatalf("Error initializating the %s connection: %v", tapdConfig.TAPDClientType, err)
	}
	// If no RABBITMQ_URI was provided we will not attempt to create a client
	// No rabbitmq features will be available in this case.
	var rabbitmqClient rabbitmq.Client
	if c.RabbitMQUri != "" {
		amqpClient, err := rabbitmq.DialAMQP(c.RabbitMQUri, rabbitmq.WithAmqpLogger(logger))
		if err != nil {
			logger.Fatal(err)
		}

		defer amqpClient.Close()

		rabbitmqClient, err = rabbitmq.NewClient(amqpClient,
			rabbitmq.WithLogger(logger),
			rabbitmq.WithLndInvoiceExchange(c.RabbitMQLndInvoiceExchange),
			rabbitmq.WithLndHubInvoiceExchange(c.RabbitMQLndhubInvoiceExchange),
			rabbitmq.WithLndInvoiceConsumerQueueName(c.RabbitMQInvoiceConsumerQueueName),
			rabbitmq.WithLndPaymentExchange(c.RabbitMQLndPaymentExchange),
			rabbitmq.WithLndPaymentConsumerQueueName(c.RabbitMQPaymentConsumerQueueName),
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
		TapdClient:     tapdClient,
		Logger:         logger,
		InvoicePubSub:  service.NewPubsub(),
		RabbitMQClient: rabbitmqClient,
	}

	//init echo server
	e := transport.InitEcho(c, logger)
	//if Datadog is configured, add datadog middleware
	if c.DatadogAgentUrl != "" {
		tracer.Start(tracer.WithAgentAddr(c.DatadogAgentUrl))
		defer tracer.Stop()
		e.Use(ddEcho.Middleware(ddEcho.WithServiceName("lndhub.go")))
	}

	logMw := transport.CreateLoggingMiddleware(logger)
	// strict rate limit for requests for sending payments
	strictRateLimitMiddleware := transport.CreateRateLimitMiddleware(c.StrictRateLimit, c.BurstRateLimit)

	secured := e.Group("", tokens.Middleware(c.JWTSecret), logMw)
	securedWithStrictRateLimit := e.Group("", tokens.Middleware(c.JWTSecret), strictRateLimitMiddleware, logMw)

	transport.RegisterLegacyEndpoints(svc, e, secured, securedWithStrictRateLimit, strictRateLimitMiddleware, tokens.AdminTokenMiddleware(c.AdminToken), logMw)
	transport.RegisterV2Endpoints(svc, e, secured, securedWithStrictRateLimit, strictRateLimitMiddleware, tokens.AdminTokenMiddleware(c.AdminToken), logMw)
	// inital nostr gateway

	//Swagger API spec
	docs.SwaggerInfo.Host = c.Host
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	var backgroundWg sync.WaitGroup
	backGroundCtx, _ := signal.NotifyContext(context.Background(), os.Interrupt)
	// Subscribe to LND invoice updates in the background
	backgroundWg.Add(1)
	go func() {
		err = svc.StartInvoiceRoutine(backGroundCtx)
		if err != nil {
			sentry.CaptureException(err)
			//we want to restart in case of an error here
			svc.Logger.Fatal(err)
		}
		svc.Logger.Info("Invoice routine done")
		backgroundWg.Done()
	}()

	// Check the status of all pending outgoing payments
	backgroundWg.Add(1)
	go func() {
		err = svc.StartPendingPaymentRoutine(backGroundCtx)
		if err != nil {
			sentry.CaptureException(err)
			//in case of an error here no restart is necessary
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
		go transport.StartPrometheusEcho(logger, svc, e)
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
