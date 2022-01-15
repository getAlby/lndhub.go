package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/signal"
	"time"

	controllers2 "github.com/bumi/lndhub.go/controllers"
	"github.com/bumi/lndhub.go/database"
	"github.com/bumi/lndhub.go/lib"
	"github.com/bumi/lndhub.go/lib/logging"
	"github.com/bumi/lndhub.go/lib/middlewares"
	"github.com/getsentry/sentry-go"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
)

func init() {
	err := godotenv.Load(".env")
	if err != nil {
		logrus.Errorf("failed to get env value")
		return
	}
}

func main() {
	db, err := database.Connect(os.Getenv("DATABASE_URI"))
	if err != nil {
		logrus.Errorf("failed to connect with database: %v", err)
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

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.Use(middlewares.ContextDB(db))
	e.Use(middleware.BodyLimit("250K"))
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20)))

	e.POST("/auth", controllers2.AuthController{}.Auth)
	e.POST("/create", controllers2.CreateUserController{}.CreateUser)
	e.POST("/addinvoice", controllers2.AddInvoiceController{}.AddInvoice, middleware.JWT([]byte("secret")))
	e.POST("/payinvoice", controllers2.PayInvoiceController{}.PayInvoice, middleware.JWT([]byte("secret")))
	e.GET("/gettxs", controllers2.GetTXSController{}.GetTXS, middleware.JWT([]byte("secret")))
	e.GET("/checkpayment/:payment_hash", controllers2.CheckPaymentController{}.CheckPayment, middleware.JWT([]byte("secret")))
	e.GET("/balance", controllers2.BalanceController{}.Balance, middleware.JWT([]byte("secret")))

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
