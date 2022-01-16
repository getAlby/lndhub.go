package tokens

import (
	"time"

	"github.com/bumi/lndhub.go/db/models"
	"github.com/dgrijalva/jwt-go"
)

type jwtCustomClaims struct {
	ID int64 `json:"id"`

	jwt.StandardClaims
}

// GenerateAccessToken : Generate Access Token
func GenerateAccessToken(u *models.User) (string, error) {
	claims := &jwtCustomClaims{
		u.ID,
		jwt.StandardClaims{
			// one week expiration
			ExpiresAt: time.Now().Add(time.Hour * 27 * 7).Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

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
