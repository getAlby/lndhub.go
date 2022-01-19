package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/getAlby/lndhub.go/controllers"
	"github.com/getAlby/lndhub.go/db"
	"github.com/getAlby/lndhub.go/db/migrations"
	"github.com/getAlby/lndhub.go/lib"
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
)

func main() {
	c := &service.Config{}

	// Load configruation from environment variables
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Failed to load .env file")
	}
	err = envconfig.Process("", c)
	if err != nil {
		panic(err)
	}

	// Open a DB connection based on the configured DATABASE_URI
	dbConn, err := db.Open(c.DatabaseUri)
	if err != nil {
		panic(err)
	}

	// Migrate the DB
	ctx := context.Background()
	migrator := migrate.NewMigrator(dbConn, migrations.Migrations)
	err = migrator.Init(ctx)
	if err != nil {
		panic(err)
	}
	_, err = migrator.Migrate(ctx)
	if err != nil {
		panic(err)
	}

	// New Echo app
	e := echo.New()
	e.HideBanner = true

	e.Validator = &lib.CustomValidator{Validator: validator.New()}

	e.Use(middleware.Recover())
	e.Use(middleware.BodyLimit("250K"))
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20)))

	// Setup logging to STDOUT or a configrued log file
	logger := lib.Logger(c.LogFilePath)
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
	lndClient, err := lnd.NewLNDclient(lnd.LNDoptions{
		Address:     c.LNDAddress,
		MacaroonHex: c.LNDMacaroonHex,
		CertHex:     c.LNDCertHex,
	})
	if err != nil {
		panic(err)
	}
	getInfo, err := lndClient.GetInfo(ctx, &lnrpc.GetInfoRequest{})
	if err != nil {
		panic(err)
	}
	logger.Infof("Connected to LND: %s - %s", getInfo.Alias, getInfo.IdentityPubkey)

	svc := &service.LndhubService{
		Config:    c,
		DB:        dbConn,
		LndClient: &lndClient,
	}

	// Public endpoints for account creation and authentication
	e.POST("/auth", controllers.NewAuthController(svc).Auth)
	e.POST("/create", controllers.NewCreateUserController(svc).CreateUser)

	// Secured endpoints which require a Authorization token (JWT)
	secured := e.Group("", tokens.Middleware(c.JWTSecret))
	secured.POST("/addinvoice", controllers.NewAddInvoiceController(svc).AddInvoice)
	secured.POST("/payinvoice", controllers.NewPayInvoiceController(svc).PayInvoice)
	secured.GET("/gettxs", controllers.NewGetTXSController(svc).GetTXS)
	secured.GET("/checkpayment/:payment_hash", controllers.NewCheckPaymentController(svc).CheckPayment)
	secured.GET("/balance", controllers.NewBalanceController(svc).Balance)

	// These endpoints are not supported and we return a blank response for backwards compatibility
	blankController := controllers.NewBlankController(svc)
	secured.GET("/getbtc", blankController.GetBtc)
	secured.GET("/getpending", blankController.GetPending)

	// Start server
	go func() {
		if err := e.Start(":3000"); err != nil && err != http.ErrServerClosed {
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
}
