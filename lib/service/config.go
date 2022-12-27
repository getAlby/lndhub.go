package service

import (
	"fmt"
	"strings"
)

type Config struct {
	DatabaseUri            string `envconfig:"DATABASE_URI" required:"true"`
	SentryDSN              string `envconfig:"SENTRY_DSN"`
	SentryTracesSampleRate string `envconfig:"SENTRY_TRACES_SAMPLE_RATE"`
	LogFilePath            string `envconfig:"LOG_FILE_PATH"`
	JWTSecret              []byte `envconfig:"JWT_SECRET" required:"true"`
	AdminToken             string `envconfig:"ADMIN_TOKEN"`
	JWTRefreshTokenExpiry  int    `envconfig:"JWT_REFRESH_EXPIRY" default:"604800"` // in seconds, default 7 days
	JWTAccessTokenExpiry   int    `envconfig:"JWT_ACCESS_EXPIRY" default:"172800"`  // in seconds, default 2 days
	LNDAddress             string `envconfig:"LND_ADDRESS" required:"true"`
	LNDMacaroonFile        string `envconfig:"LND_MACAROON_FILE"`
	LNDCertFile            string `envconfig:"LND_CERT_FILE"`
	LNDMacaroonHex         string `envconfig:"LND_MACAROON_HEX"`
	LNDCertHex             string `envconfig:"LND_CERT_HEX"`
	CustomName             string `envconfig:"CUSTOM_NAME"`
	Host                   string `envconfig:"HOST" default:"localhost:3000"`
	Port                   int    `envconfig:"PORT" default:"3000"`
	EnableGRPC             bool   `envconfig:"ENABLE_GRPC" default:"false"`
	GRPCPort               int    `envconfig:"GRPC_PORT" default:"10009"`
	DefaultRateLimit       int    `envconfig:"DEFAULT_RATE_LIMIT" default:"10"`
	StrictRateLimit        int    `envconfig:"STRICT_RATE_LIMIT" default:"10"`
	BurstRateLimit         int    `envconfig:"BURST_RATE_LIMIT" default:"1"`
	EnablePrometheus       bool   `envconfig:"ENABLE_PROMETHEUS" default:"false"`
	PrometheusPort         int    `envconfig:"PROMETHEUS_PORT" default:"9092"`
	WebhookUrl             string `envconfig:"WEBHOOK_URL"`
	FeeReserve             bool   `envconfig:"FEE_RESERVE" default:"false"`
	AllowAccountCreation   bool   `envconfig:"ALLOW_ACCOUNT_CREATION" default:"true"`
	MinPasswordEntropy     int    `envconfig:"MIN_PASSWORD_ENTROPY" default:"0"`
	MaxReceiveAmount       int64  `envconfig:"MAX_RECEIVE_AMOUNT" default:"0"`
	MaxSendAmount          int64  `envconfig:"MAX_SEND_AMOUNT" default:"0"`
	MaxAccountBalance      int64  `envconfig:"MAX_ACCOUNT_BALANCE" default:"0"`
	Branding               BrandingConfig
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
