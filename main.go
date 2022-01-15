package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/bumi/lndhub.go/controllers"
	"github.com/bumi/lndhub.go/db"
	"github.com/bumi/lndhub.go/db/migrations"
	"github.com/bumi/lndhub.go/lib"
	"github.com/bumi/lndhub.go/lib/logging"
	"github.com/getsentry/sentry-go"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
	"github.com/uptrace/bun/migrate"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		logrus.Errorf("failed to get env value")
		return
	}

	dbConn, err := db.Open(os.Getenv("DATABASE_URI"))
	if err != nil {
		logrus.Fatalf("failed to connect to database: %v", err)
		return
	}

	sentryDsn := os.Getenv("SENTRY_DSN")

	switch sentryDsn {
	case "":
		//ignore
		break
	default:
		if err = sentry.Init(sentry.ClientOptions{
			Dsn: os.Getenv("SENTRY_DSN"),
		}); err != nil {
			logrus.Fatalf("sentry init error: %v", err)
		}
		defer sentry.Flush(2 * time.Second)
	}

	e := echo.New()

	e.Validator = &lib.CustomValidator{Validator: validator.New()}

	logFilePath := os.Getenv("LOG_FILE_PATH")
	if logFilePath != "" {
		file, err := logging.GetLoggingFile(logFilePath)
		if err != nil {
			logrus.Fatalf("failed to create logging file: %v", err)
		}
		e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
			Output: io.Writer(file),
		}))
	}

	ctx := context.Background()
	migrator := migrate.NewMigrator(dbConn, migrations.Migrations)
	err = migrator.Init(ctx)
	if err != nil {
		logrus.Fatalf("failed to init migrations: %v", err)
	}

	//TODO: possibly print what has been migrated
	_, err = migrator.Migrate(ctx)
	if err != nil {
		logrus.Fatalf("failed to run migrations: %v", err)
	}

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Initialise a custom context
	// Same context we will later add the user to and possible other things
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cc := &lib.IndhubContext{Context: c, DB: dbConn}
			return next(cc)
		}
	})
	e.Use(middleware.BodyLimit("250K"))
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20)))

	e.POST("/auth", controllers.AuthController{}.Auth)
	e.POST("/create", controllers.CreateUserController{}.CreateUser)

	secured := e.Group("", middleware.JWT([]byte("secret")))
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
