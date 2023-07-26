package main

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/getAlby/lndhub.go/db"
	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/getAlby/lndhub.go/lnd"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/labstack/echo/v4"
	"github.com/lightningnetwork/lnd/lnrpc"
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
	lndClient, err := lnd.NewLNDclient(lnd.LNDoptions{
		Address:      c.LNDAddress,
		MacaroonFile: c.LNDMacaroonFile,
		MacaroonHex:  c.LNDMacaroonHex,
		CertFile:     c.LNDCertFile,
		CertHex:      c.LNDCertHex,
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

	numDays := 30
	numMaxInvoices := 100
	ctx := context.Background()
	//for loop:
	offset := uint64(0)
	//	- fetch next 100 invoices from LND
	startTime := time.Now()
	counter := 0
	for {

		invoiceResp, err := lndClient.ListInvoices(ctx, &lnrpc.ListInvoiceRequest{
			PendingOnly:    false,
			IndexOffset:    offset,
			NumMaxInvoices: uint64(numMaxInvoices),
			Reversed:       true,
		})
		if err != nil {
			svc.Logger.Fatal(err)
		}
		//  for all invoices:
		for _, lndInvoice := range invoiceResp.Invoices {
			creationDate := time.Unix(lndInvoice.CreationDate, 0)
			//		- return if invoice older than time X
			if creationDate.Before(time.Now().Add(-1 * time.Duration(numDays) * 24 * time.Hour)) {
				fmt.Printf("time elapsed: %f total invoices %d \n", time.Now().Sub(startTime).Seconds(), counter)
				return
			}
			//		- get payment hash and do a db query
			var dbInvoice models.Invoice
			counter += 1

			err := svc.DB.NewSelect().Model(&dbInvoice).Where("invoice.r_hash = ?", hex.EncodeToString(lndInvoice.RHash)).Limit(1).Scan(ctx)
			if err != nil {
				// 	 	- if not found, dump invoice json
				if errors.Is(err, sql.ErrNoRows) {
					marshalled, err := json.Marshal(lndInvoice)
					if err != nil {
						svc.Logger.Fatal(err)
					}
					fmt.Println(string(marshalled))
					fmt.Println()
					continue
				}
				svc.Logger.Fatal(err)
			}
		}
		offset = invoiceResp.FirstIndexOffset
	}
}
