package service

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"

	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/lib/security"
	"github.com/getAlby/lndhub.go/lib/tokens"
	"github.com/labstack/gommon/random"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"
)

const alphaNumBytes = random.Alphanumeric

type LndhubService struct {
	Config    *Config
	DB        *bun.DB
	LndClient *lnrpc.LightningClient
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
func (svc *LndhubService) CreateUser() (user *models.User, err error) {

	user = &models.User{}

	// generate user login/password (TODO: allow the user to choose a login/password?)
	user.Login = randStringBytes(8)
	password := randStringBytes(15)
	// we only store the hashed password but return the initial plain text password in the HTTP response
	hashedPassword := security.HashPassword(password)
	user.Password = hashedPassword

	// Create user and the user's accounts
	// We use double-entry bookkeeping so we use 4 accounts: incoming, current, outgoing and fees
	// Wrapping this in a transaction in case something fails
	err = svc.DB.RunInTx(context.TODO(), &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(user).Exec(ctx); err != nil {
			return err
		}
		accountTypes := []string{"incoming", "current", "outgoing", "fees"}
		for _, accountType := range accountTypes {
			account := models.Account{UserID: user.ID, Type: accountType}
			if _, err := tx.NewInsert().Model(&account).Exec(ctx); err != nil {
				return err
			}
		}
		return nil
	})
	//return actual password in the response, not the hashed one
	user.Password = password
	return user, err
}

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = alphaNumBytes[rand.Intn(len(alphaNumBytes))]
	}
	return string(b)
}
