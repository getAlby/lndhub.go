package main

import (
	"context"
	"log"

	"github.com/getAlby/lndhub.go/db"
	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/getAlby/lndhub.go/lnd"
	"github.com/kelseyhightower/envconfig"
)

func main() {
	//hardcode hash and user id for this fix
	//TODO: first test on testnet, then run with real user id and hash:
	//hash := "5cbde6f7ea043470c1b05d1b9fc2fbe50e5a86ad9782c8991ef33aca4496829b"
	//userId := 4285
	hash := ""
	userId := 0

	ctx := context.Background()
	c := &service.Config{}

	err := envconfig.Process("", c)
	if err != nil {
		log.Fatalf("Error loading environment variables: %v", err)
	}

	// Setup logging to STDOUT or a configrued log file
	logger := lib.Logger(c.LogFilePath)

	// Open a DB connection based on the configured DATABASE_URI
	dbConn, err := db.Open(c.DatabaseUri)
	if err != nil {
		logger.Fatalf("Error initializing db connection: %v", err)
	}
	// Init new LND client
	lndClient, err := lnd.NewLNDclient(lnd.LNDoptions{
		Address:      c.LNDAddress,
		MacaroonFile: c.LNDMacaroonFile,
		MacaroonHex:  c.LNDMacaroonHex,
		CertFile:     c.LNDCertFile,
		CertHex:      c.LNDCertHex,
	})
	if err != nil {
		logger.Fatalf("Error initializing the LND connection: %v", err)
	}
	svc := &service.LndhubService{
		Config:        c,
		DB:            dbConn,
		LndClient:     lndClient,
		Logger:        logger,
		InvoicePubSub: service.NewPubsub(),
	}
	invoice, err := svc.FindInvoiceByPaymentHash(ctx, int64(userId), hash)
	if err != nil {
		logger.Fatal(err)
	}
	//call svc.TrackPayment
	err = svc.TrackOutgoingPaymentstatus(ctx, invoice)
	if err != nil {
		logger.Error(err)
	}
}
