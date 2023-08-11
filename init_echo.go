package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

	cache "github.com/SporkHubr/echo-http-cache"
	"github.com/SporkHubr/echo-http-cache/adapter/memory"
	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	sentryecho "github.com/getsentry/sentry-go/echo"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"github.com/ziflex/lecho/v3"
	"golang.org/x/time/rate"
)

func initEcho(c *service.Config, logger *lecho.Logger) (e *echo.Echo) {

	// New Echo app
	e = echo.New()
	e.HideBanner = true

	e.HTTPErrorHandler = responses.HTTPErrorHandler
	e.Validator = &lib.CustomValidator{Validator: validator.New()}

	e.Use(middleware.Recover())
	e.Use(middleware.BodyLimit("250K"))
	// set the default rate limit defining the overal max requests/second
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(rate.Limit(c.DefaultRateLimit))))

	e.Logger = logger
	e.Use(middleware.RequestID())

	// Setup exception tracking with Sentry if configured
	// sentry init needs to happen before the echo middlewares are added
	if c.SentryDSN != "" {
		e.Use(sentryecho.New(sentryecho.Options{}))
	}
	return e
}
func createLoggingMiddleware(logger *lecho.Logger) echo.MiddlewareFunc {
	return lecho.Middleware(lecho.Config{
		Logger: logger,
		Enricher: func(c echo.Context, logger zerolog.Context) zerolog.Context {
			return logger.Interface("UserID", c.Get("UserID"))
		},
	})
}

func createRateLimitMiddleware(requestsPerSecond int, burst int) echo.MiddlewareFunc {
	config := middleware.RateLimiterConfig{
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{Rate: rate.Limit(requestsPerSecond), Burst: burst},
		),
		IdentifierExtractor: func(ctx echo.Context) (string, error) {
			userId := ctx.Get("UserID")
			id := ctx.RealIP()
			if userId != nil {
				userIdAsInt64 := ctx.Get("UserID").(int64)
				id = strconv.FormatInt(userIdAsInt64, 10)
			}

			return id, nil
		},
	}

	return middleware.RateLimiterWithConfig(config)
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

func startPrometheusEcho(logger *lecho.Logger, svc *service.LndhubService, e *echo.Echo) {
	// Create Prometheus server and Middleware
	echoPrometheus := echo.New()
	echoPrometheus.HideBanner = true
	prom := prometheus.NewPrometheus("echo", nil)
	// Scrape metrics from Main Server
	e.Use(prom.HandlerFunc)
	// Setup metrics endpoint at another server
	prom.SetMetricsPath(echoPrometheus)
	echoPrometheus.Logger = logger
	echoPrometheus.Logger.Infof("Starting prometheus on port %d", svc.Config.PrometheusPort)
	echoPrometheus.Logger.Fatal(echoPrometheus.Start(fmt.Sprintf(":%d", svc.Config.PrometheusPort)))
}
