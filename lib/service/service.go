package service

import (
	"context"
	"fmt"
	"strconv"

	"github.com/btcsuite/btcd/btcec"
	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/lib/tokens"
	"github.com/getAlby/lndhub.go/lnd"
	"github.com/labstack/gommon/random"
	"github.com/uptrace/bun"
	"github.com/ziflex/lecho/v3"
	"golang.org/x/crypto/bcrypt"
)

const alphaNumBytes = random.Alphanumeric

type LndhubService struct {
	Config         *Config
	DB             *bun.DB
	LndClient      lnd.LightningClientWrapper
	Logger         *lecho.Logger
	IdentityPubkey *btcec.PublicKey
}

func (svc *LndhubService) GenerateToken(login, password, inRefreshToken string) (accessToken, refreshToken string, err error) {
	var user models.User

	switch {
	case login != "" || password != "":
		{
			if err := svc.DB.NewSelect().Model(&user).Where("login = ?", login).Scan(context.TODO()); err != nil {
				return "", "", fmt.Errorf("bad auth")
			}
			if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
				return "", "", fmt.Errorf("bad auth")

			}
		}
	case inRefreshToken != "":
		{
			// TODO: currently not supported
			// I'd love to remove this from the auth handler, as the refresh token
			// is usually a part of the JWT middleware: https://webdevstation.com/posts/user-authentication-with-go-using-jwt-token/
			// if the current client depends on that - we can incorporate the refresh JWT code into here
			return "", "", fmt.Errorf("bad auth")
		}
	default:
		{
			return "", "", fmt.Errorf("login and password or refresh token is required")
		}
	}

	accessToken, err = tokens.GenerateAccessToken(svc.Config.JWTSecret, svc.Config.JWTExpiry, &user)
	if err != nil {
		return "", "", err
	}

	refreshToken, err = tokens.GenerateRefreshToken(svc.Config.JWTSecret, svc.Config.JWTExpiry, &user)
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}

func (svc *LndhubService) ParseInt(value interface{}) (int64, error) {
	switch v := value.(type) {
	case float64:
		return int64(v), nil
	case string:
		c, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, err
		}
		return c, nil
	default:
		return 0, fmt.Errorf("conversion to int from %T not supported", v)
	}
}
