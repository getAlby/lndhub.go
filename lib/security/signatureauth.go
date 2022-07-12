package security

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"crypto/ed25519"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const (
	LOGIN_MESSAGE = "sign in into mintter lndhub"
)

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
	password, err := readFromBody(&c.Request().Body, "password")
	if err != nil {
		c.Logger().Debugf("could not read body %v", err)
		return true
	}
	if password.(string) == "" {
		c.Logger().Debugf("blank password")
		return true
	}

	v, ok := c.Request().Header[echo.HeaderAuthorization]
	if !ok || len(v) != 1 || !strings.Contains(strings.ToLower(v[0]),
		strings.ToLower(middleware.DefaultKeyAuthConfig.AuthScheme)) ||
		len(v[0]) <= len(middleware.DefaultKeyAuthConfig.AuthScheme+"  ") {
		c.Logger().Debugf("Wrong or absent signature auth format")
		return true
	}

	return false
}

func validate_signature(pubKeyStr string, c echo.Context) (bool, error) {

	pass, err := readFromBody(&c.Request().Body, "password")
	if err != nil {
		c.Logger().Debugf("could not read body %v", err)
		return false, err
	}
	var password = pass.(string)

	pub, err := hex.DecodeString(pubKeyStr)
	if err != nil {
		return false, fmt.Errorf("could not decode provided public key [%s] %v", pubKeyStr, err)
	}
	pubKey := ed25519.PublicKey(pub)
	if len(pubKey) != ed25519.PublicKeySize {
		return false, fmt.Errorf("provided pubkey %s must be of length %d", pubKey, ed25519.PublicKeySize)
	}
	signature, err := hex.DecodeString(password)
	if err != nil {
		c.Logger().Debugf("Unable to unmarshal signature [%s]: %v", password, err)
		return false, err
	}

	sig_ok := ed25519.Verify(pub, []byte(LOGIN_MESSAGE), signature)
	return sig_ok, nil
}

func readFromBody(r *io.ReadCloser, key string) (interface{}, error) {
	authBody := make(map[string]interface{})
	body_data, err := ioutil.ReadAll(*r)

	defer func(reader *io.ReadCloser, data []byte) {
		*reader = io.NopCloser(bytes.NewBuffer(data))
	}(r, body_data)

	if err != nil {
		return nil, fmt.Errorf("Couldn't read body: %v", err)
	}

	err = json.Unmarshal(body_data, &authBody)
	if err != nil {
		return nil, fmt.Errorf("Couldn't decode body: %v", err)
	}
	value, ok := authBody[key]

	if !ok {
		return nil, fmt.Errorf("key %s not present in body", key)
	}
	return value, nil
}
