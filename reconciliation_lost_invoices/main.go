package main

import (
	"context"
	"fmt"
	"log"

	"github.com/getAlby/lndhub.go/db"
	"github.com/getAlby/lndhub.go/db/migrations"
	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/getAlby/lndhub.go/lnd"
	"github.com/getsentry/sentry-go"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/labstack/echo/v4"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/uptrace/bun/migrate"
)

// script to reconcile pending payments between the backup node and the database
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

	// New Echo app
	e := echo.New()

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
		e.Logger.Fatalf("Error getting node info: %v", err)
	}
	logger.Infof("Connected to LND: %s - %s", getInfo.Alias, getInfo.IdentityPubkey)

	svc := &service.LndhubService{
		Config:         c,
		DB:             dbConn,
		LndClient:      lndClient,
		Logger:         logger,
		IdentityPubkey: getInfo.IdentityPubkey,
		InvoicePubSub:  service.NewPubsub(),
	}

	err = svc.CheckAllPendingOutgoingPayments(startupCtx)
	if err != nil {
		sentry.CaptureException(err)
		svc.Logger.Error(err)
	}
}
