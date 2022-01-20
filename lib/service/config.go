package service

type Config struct {
	DatabaseUri        string `envconfig:"DATABASE_URI" required:"true"`
	SentryDSN          string `envconfig:"SENTRY_DSN"`
	LogFilePath        string `envconfig:"LOG_FILE_PATH"`
	JWTSecret          []byte `envconfig:"JWT_SECRET" required:"true"`
	JWTExpiry          int    `envconfig:"JWT_EXPIRY" default:"604800"` // in seconds
	LNDAddress         string `envconfig:"LND_ADDRESS" required:"true"`
	LNDMacaroonHex     string `envconfig:"LND_MACAROON_HEX" required:"true"`
	LNDCertHex         string `envconfig:"LND_CERT_HEX"`
	TestDatabaseUri    string `envconfig:"TEST_DATABASE_URI"`
	TestLNDAddress     string `envconfig:"TEST_LND_ADDRESS"`
	TestLNDMacaroonHex string `envconfig:"TEST_LND_MACAROON_HEX"`
	TestLNDCertHex     string `envconfig:"TEST_LND_CERT_HEX"`
}
