package tokens

import (
	"database/sql"
	"time"

	"github.com/bumi/lndhub.go/pkg/database/models"

	"github.com/dgrijalva/jwt-go"
)

type jwtCustomClaims struct {
	ID    uint   `json:"id"`
	Email string `json:"email"`
	Login string `json:"login"`

	jwt.StandardClaims
}

// GenerateAccessToken : Generate Access Token
func GenerateAccessToken(u *models.User) error {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtCustomClaims{
		ID:    u.ID,
		StandardClaims: jwt.StandardClaims{
			// one week expiration
			ExpiresAt: time.Now().Add(time.Hour * 168).Unix(),
		},
	})

	t, err := token.SignedString([]byte("secret"))
	if err != nil {
		return err
	}
	u.AccessToken = sql.NullString{String: t, Valid: true}

	return err
}

// GenerateRefreshToken : Generate Refresh Token
func GenerateRefreshToken(u *models.User) error {
	rToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id": u.ID,
	})

	rt, err := rToken.SignedString([]byte("secret"))
	if err != nil {
		return err
	}

	u.RefreshToken = sql.NullString{String: rt, Valid: true}

	return err
}
