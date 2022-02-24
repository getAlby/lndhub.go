package service

type Config struct {
	DatabaseUri    string `envconfig:"DATABASE_URI" required:"true"`
	SentryDSN      string `envconfig:"SENTRY_DSN"`
	LogFilePath    string `envconfig:"LOG_FILE_PATH"`
	JWTSecret      []byte `envconfig:"JWT_SECRET" required:"true"`
	JWTExpiry      int    `envconfig:"JWT_EXPIRY" default:"604800"` // in seconds
	LNDAddress     string `envconfig:"LND_ADDRESS" required:"true"`
	LNDMacaroonHex string `envconfig:"LND_MACAROON_HEX" required:"true"`
	LNDCertHex     string `envconfig:"LND_CERT_HEX"`
	CustomName     string `envconfig:"CUSTOM_NAME"`
	Port           int    `envconfig:"PORT" default:"3000"`
	FixedFee       int    `envconfig:"FIXED_FEE" default:"10"`
}
