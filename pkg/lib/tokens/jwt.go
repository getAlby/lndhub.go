package tokens

import (
	"database/sql"
	models2 "github.com/bumi/lndhub.go/pkg/database/models"

	"github.com/dgrijalva/jwt-go"
)

type jwtCustomClaims struct {
	ID    uint   `json:"id"`
	Email string `json:"email"`
	Login string `json:"login"`

	jwt.StandardClaims
}

// GenerateAccessToken : Generate Access Token
func GenerateAccessToken(u *models2.User) error {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtCustomClaims{
		ID:    u.ID,
		Email: u.Email.String,
		Login: u.Login,
	})

	t, err := token.SignedString([]byte("secret"))
	if err != nil {
		return err
	}
	u.AccessToken = sql.NullString{String: t, Valid: true}

	return err
}

// GenerateRefreshToken : Generate Refresh Token
func GenerateRefreshToken(u *models2.User) error {
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
