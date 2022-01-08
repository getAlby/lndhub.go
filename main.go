package main

import (
	"context"
	"net/http"
	"github.com/labstack/echo/v4/middleware"
	"os"
	"os/signal"
	"time"

	"github.com/bumi/lndhub.go/database"
	"github.com/bumi/lndhub.go/lib"
	"github.com/bumi/lndhub.go/lib/middlewares"
	"github.com/bumi/lndhub.go/routes"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

func init() {
	godotenv.Load(".env")
}

func main() {
	db, err := database.Connect(os.Getenv("DATABASE_URI"))
	if err != nil {
		logrus.Errorf("failed to connect with database: %v", err)
		return
	}

	e := echo.New()

	e.Validator = &lib.CustomValidator{Validator: validator.New()}

	e.Use(middlewares.ContextDB(db))
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20)))
	//e.Use(middlewares.IsLoggedIn)

	routes.Routes(e.Group(""))

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
