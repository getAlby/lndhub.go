package tokens

import (
	"github.com/bumi/lndhub.go/db/models"
	"github.com/dgrijalva/jwt-go"
)

type jwtCustomClaims struct {
	ID    uint   `json:"id"`
	Email string `json:"email"`
	Login string `json:"login"`

	jwt.StandardClaims
}

// GenerateAccessToken : Generate Access Token
func GenerateAccessToken(u *models.User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtCustomClaims{
		ID:    u.ID,
		Email: u.Email.String,
		Login: u.Login,
	})

	t, err := token.SignedString([]byte("secret"))
	if err != nil {
		return "", err
	}

	return t, nil
}

// GenerateRefreshToken : Generate Refresh Token
func GenerateRefreshToken(u *models.User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id": u.ID,
	})

	t, err := token.SignedString([]byte("secret"))
	if err != nil {
		return "", err
	}

	return t, nil
}
