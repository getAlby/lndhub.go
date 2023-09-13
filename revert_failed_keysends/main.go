package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/getAlby/lndhub.go/db"
	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

func main() {
	ctx := context.Background()
	// init service

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

	svc := &service.LndhubService{
		Config:        c,
		DB:            dbConn,
		Logger:        logger,
		InvoicePubSub: service.NewPubsub(),
	}
	// fetch invoices from database
	failedKeysends := []models.Invoice{}
	err = svc.DB.NewRaw(
		`
		SELECT DISTINCT invoices.id, invoices.created_at, invoices.amount, invoices.user_id, invoices.state 
		from invoices join transaction_entries on invoices.id = transaction_entries.invoice_id
		where state = 'initialized' 
		and keysend = true 
		and "type" = 'outgoing'
		`,
	).Scan(ctx, &failedKeysends)
	if err != nil {
		logger.Fatalf("Error fetching failed keysends: %v", err)
	}
	// dry-run: print details
	isDryRun := os.Getenv("DRY_RUN") == "true"
	if isDryRun {
		for _, fk := range failedKeysends {
			fmt.Printf("user id %d, id %d, amount %d, state %s \n", fk.UserID, fk.ID, fk.Amount, fk.State)
		}
		fmt.Printf("Dry run: Found %d invoices \n", len(failedKeysends))
		return
	}
	// call handleFailedPayment on all of them
	for _, fk := range failedKeysends {
		entry, err := svc.GetTransactionEntryByInvoiceId(ctx, fk.ID)
		if err != nil {
			logger.Fatalf("Error fetching failed keysends: %v", err)
		}
		err = svc.HandleFailedPayment(ctx, &fk, entry, fmt.Errorf("payment failed"))
		if err != nil {
			logger.Fatalf("Error fetching failed keysends: %v", err)
		}
	}
}
