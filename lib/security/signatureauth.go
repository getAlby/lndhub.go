package security

import (
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/libp2p/go-libp2p-core/crypto"
)

const (
	LOGIN_MESSAGE = "sign in into mintter lndhub"
)

type authBody struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func SignatureMiddleware() echo.MiddlewareFunc {
	config := middleware.DefaultKeyAuthConfig
	config.ErrorHandler = func(err error, c echo.Context) error {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusUnauthorized, echo.Map{
			"error":   true,
			"code":    1,
			"message": "bad auth",
		})
	}
	config.Validator = validate_signature
	config.Skipper = check_skip
	return middleware.KeyAuthWithConfig(config)
}

func check_skip(c echo.Context) bool {
	var body authBody

	if err := c.Bind(&body); err != nil {
		c.Logger().Debugf("No login and password is present in body: %v", err)
		return true
	}
	if body.Login == "" || body.Password == "" {
		c.Logger().Debugf("login [%s] or password [%s] is blank", body.Login, body.Password)
		return true
	}

	v, ok := c.Request().Header[echo.HeaderAuthorization]
	if !ok || len(v) != 1 || strings.Contains(strings.ToLower(v[0]),
		strings.ToLower(middleware.DefaultKeyAuthConfig.AuthScheme)) ||
		len(v[0]) <= len(strings.ToLower(middleware.DefaultKeyAuthConfig.AuthScheme)) {
		c.Logger().Debugf("Wrong auth format")
		return true
	}

	return false
}

func validate_signature(pubKey string, c echo.Context) (bool, error) {
	pubb, err := hex.DecodeString(pubKey)
	if err != nil {
		c.Logger().Debugf("Unable to get pubkey [%s] from header: %v", pubKey, err)
	}
	pub, err := crypto.UnmarshalPublicKey(pubb)
	if err != nil {
		c.Logger().Debugf("Unable to unmarshal pubkey [%s]: %v", pubKey, err)
	}
	var body authBody

	if err := c.Bind(&body); err != nil {
		c.Logger().Debugf("No login and password is present in body: %v", err)
		return false, err
	}
	signature, err := hex.DecodeString(body.Password)
	if err != nil {
		c.Logger().Debugf("Unable to unmarshal signature [%s]: %v", body.Password, err)
		return false, err
	}
	message, err := hex.DecodeString(LOGIN_MESSAGE)
	if err != nil {
		c.Logger().Debugf("Unable to unmarshal login message [%s]: %v", LOGIN_MESSAGE, err)
		return false, err
	}

	ok, err := pub.Verify(message, signature)
	if err != nil {
		c.Logger().Debugf("error verifying signature: %v", err)
		return false, err
	}

	return ok, nil
}
