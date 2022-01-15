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
	"github.com/bumi/lndhub.go/lib/middlewares"
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

	e.Use(middlewares.ContextDB(dbConn))
	e.Use(middleware.BodyLimit("250K"))
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20)))

	e.POST("/auth", controllers.AuthController{}.Auth)
	e.POST("/create", controllers.CreateUserController{}.CreateUser)
	e.POST("/addinvoice", controllers.AddInvoiceController{}.AddInvoice, middleware.JWT([]byte("secret")))
	e.POST("/payinvoice", controllers.PayInvoiceController{}.PayInvoice, middleware.JWT([]byte("secret")))
	e.GET("/gettxs", controllers.GetTXSController{}.GetTXS, middleware.JWT([]byte("secret")))
	e.GET("/checkpayment/:payment_hash", controllers.CheckPaymentController{}.CheckPayment, middleware.JWT([]byte("secret")))
	e.GET("/balance", controllers.BalanceController{}.Balance, middleware.JWT([]byte("secret")))

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
