package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/bumi/lndhub.go/controllers"
	"github.com/bumi/lndhub.go/db"
	"github.com/bumi/lndhub.go/db/migrations"
	"github.com/bumi/lndhub.go/lib"
	"github.com/bumi/lndhub.go/lib/tokens"
	"github.com/bumi/lndhub.go/lnd"
	"github.com/getsentry/sentry-go"
	sentryecho "github.com/getsentry/sentry-go/echo"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/uptrace/bun/migrate"
	"github.com/ziflex/lecho/v3"
)

type Config struct {
	DatabaseUri    string `envconfig:"DATABASE_URI" required:"true"`
	SentryDSN      string `envconfig:"SENTRY_DSN"`
	LogFilePath    string `envconfig:"LOG_FILE_PATH"`
	JWTSecret      []byte `envconfig:"JWT_SECRET" required:"true"`
	JWTExpiry      int    `envconfig:"JWT_EXPIRY" default:"604800"` // in seconds
	LNDAddress     string `envconfig:"LND_ADDRESS" required:"true"`
	LNDMacaroonHex string `envconfig:"LND_MACAROON_HEX" required:"true"`
	LNDCertHex     string `envconfig:"LND_CERT_HEX"`
}

func main() {
	var c Config
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Failed to load .env file")
	}

	err = envconfig.Process("", &c)
	if err != nil {
		panic(err)
	}

	dbConn, err := db.Open(c.DatabaseUri)
	if err != nil {
		panic(err)
	}

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

	lndClient, err := lnd.NewLNDclient(lnd.LNDoptions{
		Address:     c.LNDAddress,
		MacaroonHex: c.LNDMacaroonHex,
		CertHex:     c.LNDCertHex,
	})

	e := echo.New()
	e.HideBanner = true

	e.Validator = &lib.CustomValidator{Validator: validator.New()}

	e.Use(middleware.Recover())
	e.Use(middleware.BodyLimit("250K"))
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20)))

	logger := lib.Logger(c.LogFilePath)
	e.Logger = logger
	e.Use(middleware.RequestID())
	e.Use(lecho.Middleware(lecho.Config{
		Logger: logger,
	}))

	if c.SentryDSN != "" {
		if err = sentry.Init(sentry.ClientOptions{
			Dsn: c.SentryDSN,
		}); err != nil {
			logger.Errorf("sentry init error: %v", err)
		}
		defer sentry.Flush(2 * time.Second)
		e.Use(sentryecho.New(sentryecho.Options{}))
	}

	// Initialise a custom context
	// Same context we will later add the user to and possible other things
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cc := &lib.LndhubContext{Context: c, DB: dbConn, LndClient: &lndClient}
			return next(cc)
		}
	})

	e.POST("/auth", controllers.AuthController{JWTSecret: c.JWTSecret, JWTExpiry: c.JWTExpiry}.Auth)
	e.POST("/create", controllers.CreateUserController{}.CreateUser)

	secured := e.Group("", tokens.Middleware(c.JWTSecret), tokens.UserMiddleware(dbConn))
	secured.POST("/addinvoice", controllers.AddInvoiceController{}.AddInvoice)
	secured.POST("/payinvoice", controllers.PayInvoiceController{}.PayInvoice)
	secured.GET("/gettxs", controllers.GetTXSController{}.GetTXS)
	secured.GET("/checkpayment/:payment_hash", controllers.CheckPaymentController{}.CheckPayment)
	secured.GET("/balance", controllers.BalanceController{}.Balance)

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
