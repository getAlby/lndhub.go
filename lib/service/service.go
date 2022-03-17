package service

import (
	"context"
	"fmt"
	"reflect"
	"strconv"

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
	Config            *Config
	DB                *bun.DB
	LndClient         lnd.LightningClientWrapper
	Logger            *lecho.Logger
	IdentityPubkey    string
	MiddlewarePlugins map[string]reflect.Value
}

func (svc *LndhubService) GenerateToken(ctx context.Context, login, password, inRefreshToken string) (accessToken, refreshToken string, err error) {
	var user models.User

	switch {
	case login != "" || password != "":
		{
			if err := svc.DB.NewSelect().Model(&user).Where("login = ?", login).Scan(ctx); err != nil {
				return "", "", fmt.Errorf("bad auth")
			}
			if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
				return "", "", fmt.Errorf("bad auth")
			}
		}
	case inRefreshToken != "":
		{
			userId, err := tokens.GetUserIdFromToken(svc.Config.JWTSecret, inRefreshToken)
			if err != nil {
				return "", "", fmt.Errorf("bad auth")
			}

			if err := svc.DB.NewSelect().Model(&user).Where("id = ?", userId).Scan(ctx); err != nil {
				return "", "", fmt.Errorf("bad auth")
			}
		}
	default:
		{
			return "", "", fmt.Errorf("login and password or refresh token is required")
		}
	}

	accessToken, err = tokens.GenerateAccessToken(svc.Config.JWTSecret, svc.Config.JWTAccessTokenExpiry, &user)
	if err != nil {
		return "", "", err
	}

	refreshToken, err = tokens.GenerateRefreshToken(svc.Config.JWTSecret, svc.Config.JWTRefreshTokenExpiry, &user)
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
