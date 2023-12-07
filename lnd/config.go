package lnd

import (
	"github.com/kelseyhightower/envconfig"
)

const (
	LND_CLIENT_TYPE         = "lnd"
	LND_CLUSTER_CLIENT_TYPE = "lnd_cluster"
	ECLAIR_CLIENT_TYPE      = "eclair"
)

type Config struct {
	LNClientType                 string  `envconfig:"LN_CLIENT_TYPE" default:"lnd"` //lnd, lnd_cluster, eclair
	LNDAddress                   string  `envconfig:"LND_ADDRESS" required:"true"`
	LNDMacaroonFile              string  `envconfig:"LND_MACAROON_FILE"`
	LNDCertFile                  string  `envconfig:"LND_CERT_FILE"`
	LNDMacaroonHex               string  `envconfig:"LND_MACAROON_HEX"`
	LNDCertHex                   string  `envconfig:"LND_CERT_HEX"`
	LNDClusterLivenessPeriod     int     `envconfig:"LND_CLUSTER_LIVENESS_PERIOD" default:"10"`
	LNDClusterActiveChannelRatio float64 `envconfig:"LND_CLUSTER_ACTIVE_CHANNEL_RATIO" default:"0.5"`
	LNDClusterPubkeys            string  `envconfig:"LND_CLUSTER_PUBKEYS"` //comma-seperated list of public keys of the cluster
}

func LoadConfig() (c *Config, err error) {
	c = &Config{}
	err = envconfig.Process("", c)
	if err != nil {
		return nil, err
	}
	return c, nil
}
