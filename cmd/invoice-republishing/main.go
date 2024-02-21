package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/getAlby/lndhub.go/db"
	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/lib"
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
	logger := lib.Logger(c.LogFilePath)
	startDate, endDate, err := loadStartAndEndIdFromEnv()
	if err != nil {
		logger.Fatalf("Could not load start and end id from env %v", err)
	}
	err = envconfig.Process("", c)
	if err != nil {
		logger.Fatalf("Error loading environment variables: %v", err)
	}
	// Open a DB connection based on the configured DATABASE_URI
	dbConn, err := db.Open(c)
	if err != nil {
		logger.Fatalf("Error initializing db connection: %v", err)
	}
	amqpClient, err := rabbitmq.DialAMQP(c.RabbitMQUri, rabbitmq.WithAmqpLogger(logger))
	if err != nil {
		logger.Fatal(err)
	}

	defer amqpClient.Close()

	rabbitmqClient, err := rabbitmq.NewClient(amqpClient,
		rabbitmq.WithLogger(logger),
		rabbitmq.WithLndInvoiceExchange(c.RabbitMQLndInvoiceExchange),
		rabbitmq.WithLndHubInvoiceExchange(c.RabbitMQLndhubInvoiceExchange),
		rabbitmq.WithLndInvoiceConsumerQueueName(c.RabbitMQInvoiceConsumerQueueName),
		rabbitmq.WithLndPaymentExchange(c.RabbitMQLndPaymentExchange),
		rabbitmq.WithLndPaymentConsumerQueueName(c.RabbitMQPaymentConsumerQueueName),
	)
	if err != nil {
		logger.Fatal(err)
	}
	//small hack to get the script to work decently
	defaultClient := rabbitmqClient.(*rabbitmq.DefaultClient)

	// close the connection gently at the end of the runtime
	defer rabbitmqClient.Close()

	result := []models.Invoice{}
	err = dbConn.NewSelect().Model(&result).Where("settled_at > ?", startDate).Where("settled_at < ?", endDate).Scan(context.Background())
	if err != nil {
		logger.Fatal(err)
	}
	logrus.Infof("Found %d invoices", len(result))
	svc := &service.LndhubService{
		Config:         c,
		DB:             dbConn,
		Logger:         logger,
		RabbitMQClient: rabbitmqClient,
		InvoicePubSub:  service.NewPubsub(),
	}
	ctx := context.Background()
	dryRun := os.Getenv("DRY_RUN") == "true"
	errCount := 0
	for _, inv := range result {
		logger.Infof("Publishing invoice with hash %s", inv.RHash)
		if dryRun {
			continue
		}
		err = defaultClient.PublishToLndhubExchange(ctx, inv, svc.EncodeInvoiceWithUserLogin)
		if err != nil {
			logrus.WithError(err).Error("errror publishing to lndhub exchange")
		}
	}
	logger.Infof("Published %d invoices, # errors %d", len(result), errCount)

}

func loadStartAndEndIdFromEnv() (start, end time.Time, err error) {
	start, err = time.Parse(time.RFC3339, os.Getenv("START_DATE"))
	if err != nil {
		return
	}
	end, err = time.Parse(time.RFC3339, os.Getenv("END_DATE"))
	return
}
