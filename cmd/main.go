package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"

	"github.com/bumi/lndhub.go/pkg/controllers"
	"github.com/bumi/lndhub.go/pkg/database"
	"github.com/bumi/lndhub.go/pkg/lib"
	"github.com/bumi/lndhub.go/pkg/lib/logging"
	"github.com/bumi/lndhub.go/pkg/lib/middlewares"
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

	e := echo.New()

	e.Validator = &lib.CustomValidator{Validator: validator.New()}

	if os.Getenv("log_file_path") != "" {
		logrus.Errorf("vleze")
		file, err := logging.GetLoggingFile(os.Getenv("LOG_FILE_PATH"))
		if err != nil {
			logrus.Errorf("failed to create logging file: %v", err)
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
