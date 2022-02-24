package service

type Config struct {
	DatabaseUri           string `envconfig:"DATABASE_URI" required:"true"`
	SentryDSN             string `envconfig:"SENTRY_DSN"`
	LogFilePath           string `envconfig:"LOG_FILE_PATH"`
	JWTSecret             []byte `envconfig:"JWT_SECRET" required:"true"`
	JWTRefreshTokenExpiry int    `envconfig:"JWT_REFRESH_EXPIRY" default:"604800"` // in seconds, 7 days
	JWTAccessTokenExpiry  int    `envconfig:"JWT_ACCESS_EXPIRY" default:"172800"`  // in seconds, 2 days
	LNDAddress            string `envconfig:"LND_ADDRESS" required:"true"`
	LNDMacaroonHex        string `envconfig:"LND_MACAROON_HEX" required:"true"`
	LNDCertHex            string `envconfig:"LND_CERT_HEX"`
	CustomName            string `envconfig:"CUSTOM_NAME"`
	Port                  int    `envconfig:"PORT" default:"3000"`
}
