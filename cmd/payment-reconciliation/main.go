package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/getAlby/lndhub.go/db"
	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/getAlby/lndhub.go/lnd"
	"github.com/getsentry/sentry-go"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/labstack/echo/v4"
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

	// New Echo app
	e := echo.New()

	// Init new LND client
	lnCfg, err := lnd.LoadConfig()
	if err != nil {
		logger.Fatalf("Failed to load lnd config %v", err)
	}
	lndClient, err := lnd.NewLNDclient(lnd.LNDoptions{
		Address:      lnCfg.LNDAddress,
		MacaroonFile: lnCfg.LNDMacaroonFile,
		MacaroonHex:  lnCfg.LNDMacaroonHex,
		CertFile:     lnCfg.LNDCertFile,
		CertHex:      lnCfg.LNDCertHex,
	}, startupCtx)
	if err != nil {
		e.Logger.Fatalf("Error initializing the LND connection: %v", err)
	}
	logger.Infof("Connected to LND: %s ", lndClient.GetMainPubkey())

	svc := &service.LndhubService{
		Config:        c,
		DB:            dbConn,
		LndClient:     lndClient,
		Logger:        logger,
		InvoicePubSub: service.NewPubsub(),
	}

	//for this job, we only search for payments older than a day to avoid current in-flight payments
	ts := time.Now().Add(-1 * 24 * time.Hour)
	pending, err := svc.GetPendingPaymentsUntil(startupCtx, ts)
	err = svc.CheckPendingOutgoingPayments(startupCtx, pending)
	if err != nil {
		sentry.CaptureException(err)
		svc.Logger.Error(err)
	}
}
