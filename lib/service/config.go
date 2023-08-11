package service

import (
	"fmt"
	"strings"
)

const (
<<<<<<< HEAD
	LND_CLIENT_TYPE    = "lnd"
	ECLAIR_CLIENT_TYPE = "eclair"
=======
	LND_CLIENT_TYPE         = "lnd"
	LND_CLUSTER_CLIENT_TYPE = "lnd_cluster"
	ECLAIR_CLIENT_TYPE      = "eclair"
>>>>>>> main
)

type Config struct {
	DatabaseUri                      string  `envconfig:"DATABASE_URI" required:"true"`
	DatabaseMaxConns                 int     `envconfig:"DATABASE_MAX_CONNS" default:"10"`
	DatabaseMaxIdleConns             int     `envconfig:"DATABASE_MAX_IDLE_CONNS" default:"5"`
	DatabaseConnMaxLifetime          int     `envconfig:"DATABASE_CONN_MAX_LIFETIME" default:"1800"` // 30 minutes
	DatabaseTimeout                  int     `envconfig:"DATABASE_TIMEOUT" default:"60"`             // 60 seconds
	SentryDSN                        string  `envconfig:"SENTRY_DSN"`
	DatadogAgentUrl                  string  `envconfig:"DATADOG_AGENT_URL"`
	SentryTracesSampleRate           float64 `envconfig:"SENTRY_TRACES_SAMPLE_RATE"`
	LogFilePath                      string  `envconfig:"LOG_FILE_PATH"`
	JWTSecret                        []byte  `envconfig:"JWT_SECRET" required:"true"`
	AdminToken                       string  `envconfig:"ADMIN_TOKEN"`
	JWTRefreshTokenExpiry            int     `envconfig:"JWT_REFRESH_EXPIRY" default:"604800"` // in seconds, default 7 days
	JWTAccessTokenExpiry             int     `envconfig:"JWT_ACCESS_EXPIRY" default:"172800"`  // in seconds, default 2 days
<<<<<<< HEAD
	LNClientType                     string  `envconfig:"LN_CLIENT_TYPE" default:"lnd"`        //lnd, eclair?
	LNDAddress                       string  `envconfig:"LND_ADDRESS"`
=======
	LNClientType                     string  `envconfig:"LN_CLIENT_TYPE" default:"lnd"`        //lnd, lnd_cluster, eclair
	LNDAddress                       string  `envconfig:"LND_ADDRESS" required:"true"`
>>>>>>> main
	LNDMacaroonFile                  string  `envconfig:"LND_MACAROON_FILE"`
	LNDCertFile                      string  `envconfig:"LND_CERT_FILE"`
	LNDMacaroonHex                   string  `envconfig:"LND_MACAROON_HEX"`
	LNDCertHex                       string  `envconfig:"LND_CERT_HEX"`
<<<<<<< HEAD
	EclairHost                       string  `envconfig:"ECLAIR_HOST"`
	EclairPassword                   string  `envconfig:"ECLAIR_PASSWORD"`
=======
	LNDClusterLivenessPeriod         int     `envconfig:"LND_CLUSTER_LIVENESS_PERIOD" default:"10"`
	LNDClusterActiveChannelRatio     float64 `envconfig:"LND_CLUSTER_ACTIVE_CHANNEL_RATIO" default:"0.5"`
>>>>>>> main
	CustomName                       string  `envconfig:"CUSTOM_NAME"`
	Host                             string  `envconfig:"HOST" default:"localhost:3000"`
	Port                             int     `envconfig:"PORT" default:"3000"`
	EnableGRPC                       bool    `envconfig:"ENABLE_GRPC" default:"false"`
	GRPCPort                         int     `envconfig:"GRPC_PORT" default:"10009"`
	DefaultRateLimit                 int     `envconfig:"DEFAULT_RATE_LIMIT" default:"10"`
	StrictRateLimit                  int     `envconfig:"STRICT_RATE_LIMIT" default:"10"`
	BurstRateLimit                   int     `envconfig:"BURST_RATE_LIMIT" default:"1"`
	EnablePrometheus                 bool    `envconfig:"ENABLE_PROMETHEUS" default:"false"`
	PrometheusPort                   int     `envconfig:"PROMETHEUS_PORT" default:"9092"`
	WebhookUrl                       string  `envconfig:"WEBHOOK_URL"`
	FeeReserve                       bool    `envconfig:"FEE_RESERVE" default:"false"`
	AllowAccountCreation             bool    `envconfig:"ALLOW_ACCOUNT_CREATION" default:"true"`
	MinPasswordEntropy               int     `envconfig:"MIN_PASSWORD_ENTROPY" default:"0"`
	MaxReceiveAmount                 int64   `envconfig:"MAX_RECEIVE_AMOUNT" default:"0"`
	MaxSendAmount                    int64   `envconfig:"MAX_SEND_AMOUNT" default:"0"`
	MaxAccountBalance                int64   `envconfig:"MAX_ACCOUNT_BALANCE" default:"0"`
	RabbitMQUri                      string  `envconfig:"RABBITMQ_URI"`
	RabbitMQLndhubInvoiceExchange    string  `envconfig:"RABBITMQ_INVOICE_EXCHANGE" default:"lndhub_invoice"`
	RabbitMQLndInvoiceExchange       string  `envconfig:"RABBITMQ_LND_INVOICE_EXCHANGE" default:"lnd_invoice"`
	RabbitMQLndPaymentExchange       string  `envconfig:"RABBITMQ_LND_PAYMENT_EXCHANGE" default:"lnd_payment"`
	RabbitMQInvoiceConsumerQueueName string  `envconfig:"RABBITMQ_INVOICE_CONSUMER_QUEUE_NAME" default:"lnd_invoice_consumer"`
	RabbitMQPaymentConsumerQueueName string  `envconfig:"RABBITMQ_PAYMENT_CONSUMER_QUEUE_NAME" default:"lnd_payment_consumer"`
	Branding                         BrandingConfig
}

type BrandingConfig struct {
	Title   string        `envconfig:"BRANDING_TITLE" default:"LndHub.go - Alby Lightning"`
	Desc    string        `envconfig:"BRANDING_DESC" default:"Alby server for the Lightning Network"`
	Url     string        `envconfig:"BRANDING_URL" default:"https://ln.getalby.com"`
	Logo    string        `envconfig:"BRANDING_LOGO" default:"/static/img/alby.svg"`
	Favicon string        `envconfig:"BRANDING_FAVICON" default:"/static/img/favicon.png"`
	Footer  FooterLinkMap `envconfig:"BRANDING_FOOTER" default:"about=https://getalby.com;community=https://t.me/getAlby"`
}

// envconfig map decoder uses colon (:) as the default separator
// we have to override the decoder so we can use colon for the protocol prefix (e.g. "https:")

type FooterLinkMap map[string]string

func (flm *FooterLinkMap) Decode(value string) error {
	m := map[string]string{}
	for _, pair := range strings.Split(value, ";") {
		kvpair := strings.Split(pair, "=")
		if len(kvpair) != 2 {
			return fmt.Errorf("invalid map item: %q", pair)
		}
		m[kvpair[0]] = kvpair[1]
	}
	*flm = m
	return nil
}
