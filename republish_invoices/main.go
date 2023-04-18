package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/getAlby/lndhub.go/db"
	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/getAlby/lndhub.go/rabbitmq"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
)

func main() {

	c := &service.Config{}
	// Load configruation from environment variables
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Failed to load .env file")
	}
	startId, endId, err := loadStartAndEndIdFromEnv()
	if err != nil {
		log.Fatalf("Could not load start and end id from env %v", err)
	}
	err = envconfig.Process("", c)
	if err != nil {
		log.Fatalf("Error loading environment variables: %v", err)
	}
	// Open a DB connection based on the configured DATABASE_URI
	dbConn, err := db.Open(c)
	if err != nil {
		log.Fatalf("Error initializing db connection: %v", err)
	}
	rabbitmqClient, err := rabbitmq.Dial(c.RabbitMQUri,
		rabbitmq.WithLndInvoiceExchange(c.RabbitMQLndInvoiceExchange),
		rabbitmq.WithLndHubInvoiceExchange(c.RabbitMQLndhubInvoiceExchange),
		rabbitmq.WithLndInvoiceConsumerQueueName(c.RabbitMQInvoiceConsumerQueueName),
	)
	if err != nil {
		log.Fatal(err)
	}

	// close the connection gently at the end of the runtime
	defer rabbitmqClient.Close()

	result := []models.Invoice{}
	err = dbConn.NewSelect().Model(&result).Where("id > ?", startId).Where("id < ?", endId).Scan(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	logrus.Infof("Found %d invoices", len(result))
	svc := &service.LndhubService{
		Config:         c,
		DB:             dbConn,
		RabbitMQClient: rabbitmqClient,
		InvoicePubSub:  service.NewPubsub(),
	}
	for _, inv := range result {
		logrus.Infof("Publishing invoice with hash %s", inv.RHash)
		err = svc.RabbitMQClient.PublishToLndhubExchange(context.Background(), inv, svc.EncodeInvoiceWithUserLogin)
		if err != nil {
			logrus.Error(err)
		}
	}

}

func loadStartAndEndIdFromEnv() (start, end int, err error) {
	start, err = strconv.Atoi(os.Getenv("START_ID"))
	if err != nil {
		return 0, 0, err
	}
	end, err = strconv.Atoi(os.Getenv("END_ID"))
	if err != nil {
		return 0, 0, err
	}
	return start, end, nil
}
