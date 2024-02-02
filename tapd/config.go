package tapd

import (
	"github.com/kelseyhightower/envconfig"
)

const (
	TAPD_CLIENT_TYPE = "tapd"
)
/// TODO do we need to import anything from lnd.config ?
type TapdConfig struct {
	TAPDClientType   string `envconfig:"TAPD_CLIENT_TYPE" default:"tapd"`
	TAPDAddress      string `envconfig:"TAPD_ADDRESS" required:"true"`
	TAPDMacaroonFile string `envconfig:"TAPD_MACAROON_FILE"`
	TAPDCertFile     string `envconfig:"TAPD_CERT_FILE"`
	TAPDMacaroonHex  string `envconfig:"TAPD_MACAROON_HEX"`
	TAPDCertHex      string `envconfig:"TAPD_CERT_HEX"`  
}

func LoadConfig() (c *TapdConfig, err error) {
	c = &TapdConfig{}

	err = envconfig.Process("", c)
	if err != nil {
		return nil, err
	}
	return c, nil
}

